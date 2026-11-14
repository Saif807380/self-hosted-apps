// Package tracker is the worker's logic: link games to IGDB, find their store
// listings, check prices and record sale events.
package tracker

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/saifkazi/trove/backend/internal/alerts"
	"github.com/saifkazi/trove/backend/internal/catalog"
	"github.com/saifkazi/trove/backend/internal/dao"
	"github.com/saifkazi/trove/backend/internal/model"
	"github.com/saifkazi/trove/backend/internal/stores"
)

const (
	linkBatch     = 50             // games linked per run, to stay well inside IGDB's 4 req/s
	fxMaxAge      = 20 * time.Hour // rates are refreshed about daily
	maxCandidates = 3              // options shown when a match is ambiguous
)

// Tracker runs the price-tracking pipeline against the gaming tables.
type Tracker struct {
	Store *dao.GamingStore
	IGDB  *catalog.Client
	Steam *stores.SteamClient
	Eshop *stores.EshopClient
	FX    *stores.FXClient
	Log   *slog.Logger
	// UploadsDir is where covers are saved; empty disables cover filling.
	UploadsDir string
}

// Refresh runs the whole pipeline for one game: link it if it never was,
// look up its listings, and check its prices.
func (t *Tracker) Refresh(ctx context.Context, id uuid.UUID) error {
	ref, err := t.Store.GetGameRef(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // deleted since it was queued
	}
	if err != nil {
		return err
	}
	if _, err := t.Store.GetMeta(ctx, id); errors.Is(err, pgx.ErrNoRows) {
		if err := t.Link(ctx, ref); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if err := t.FillBlanks(ctx, &id); err != nil {
		return err
	}
	if err := t.FindListings(ctx, &id); err != nil {
		return err
	}
	return t.CheckPrices(ctx, &id)
}

// LinkPending links games that have never been looked up in IGDB.
func (t *Tracker) LinkPending(ctx context.Context) error {
	refs, err := t.Store.GamesToLink(ctx, linkBatch)
	if err != nil {
		return err
	}
	for _, ref := range refs {
		if err := t.Link(ctx, ref); err != nil {
			t.Log.Warn("link failed", "game", ref.Title, "err", err)
		}
	}
	return t.FillBlanks(ctx, nil)
}

// Link searches IGDB for a game and stores either a confident match or, when
// ambiguous, the top candidates for you to choose from.
func (t *Tracker) Link(ctx context.Context, ref dao.GameRef) error {
	candidates, err := t.IGDB.Search(ctx, catalog.LinkQuery(ref.Title), catalog.LinkPool)
	if err != nil {
		return err
	}
	studio := ""
	if ref.Studio != nil {
		studio = *ref.Studio
	}
	if g, ok := catalog.ChooseMatch(ref.Title, studio, candidates); ok {
		t.Log.Info("linked", "game", ref.Title, "igdb", g.Title, "year", g.ReleaseYear)
		return t.Store.SaveMeta(ctx, MetaFromGame(ref.ID, g, model.MatchAuto))
	}
	t.Log.Info("ambiguous match", "game", ref.Title, "candidates", len(candidates))
	return t.Store.SaveMeta(ctx, UnresolvedMeta(ref.ID, ref.Title, candidates))
}

// FindListings looks up store listings for wishlisted games that need it
// (or just the given one).
func (t *Tracker) FindListings(ctx context.Context, id *uuid.UUID) error {
	targets, err := t.Store.GamesNeedingListings(ctx, id)
	if err != nil {
		return err
	}
	for _, target := range targets {
		listings, err := t.listingsFor(ctx, target)
		if err != nil {
			// Not marked as looked up, so the next run retries it.
			t.Log.Warn("listing lookup failed", "game", target.Title, "err", err)
			continue
		}
		if err := t.Store.SaveListings(ctx, target.ID, listings); err != nil {
			return err
		}
		t.Log.Info("listings found", "game", target.Title, "count", len(listings))
	}
	return nil
}

func (t *Tracker) listingsFor(ctx context.Context, target dao.ListingTarget) ([]stores.Listing, error) {
	g := catalog.Game{Platforms: intSlice(target.Platforms)}
	var out []stores.Listing
	if target.SteamAppID != nil && g.HasPlatform(catalog.PlatformPC) {
		out = append(out, t.Steam.ListingFor(*target.SteamAppID, target.Title))
	}
	if g.HasPlatform(catalog.PlatformSwitch, catalog.PlatformSwitch2) {
		found, err := t.Eshop.Find(ctx, target.Title)
		if err != nil {
			return nil, err
		}
		out = append(out, found...)
	}
	if g.HasPlatform(catalog.PlatformPS4, catalog.PlatformPS5) {
		existing, err := t.Store.Listings(ctx, target.ID)
		if err != nil {
			return nil, err
		}
		if !hasStore(existing, stores.PSN) { // keep a pasted product link
			out = append(out, stores.PSNSearchListing(target.Title))
		}
	}
	return out, nil
}

// CheckPrices quotes the Steam and eShop listings of wishlisted games (or
// just the given one), records the prices and raises sale events.
func (t *Tracker) CheckPrices(ctx context.Context, id *uuid.UUID) error {
	listings, err := t.Store.ListingsToPrice(ctx, id)
	if err != nil || len(listings) == 0 {
		return err
	}
	usdToINR, err := t.usdToINR(ctx)
	if err != nil {
		return err
	}

	refs := map[string][]string{}
	for _, l := range listings {
		refs[l.Store] = append(refs[l.Store], l.StoreRef)
	}
	quotes := map[string]map[string]stores.Quote{}
	if len(refs[stores.Steam]) > 0 {
		if quotes[stores.Steam], err = t.Steam.Quotes(ctx, refs[stores.Steam]); err != nil {
			t.Log.Warn("steam prices failed", "err", err)
		}
	}
	if len(refs[stores.Eshop]) > 0 {
		if quotes[stores.Eshop], err = t.Eshop.Quotes(ctx, refs[stores.Eshop]); err != nil {
			t.Log.Warn("eshop prices failed", "err", err)
		}
	}

	for _, l := range listings {
		q, ok := quotes[l.Store][l.StoreRef]
		if !ok {
			continue // not purchasable right now, or the store call failed
		}
		if err := t.record(ctx, l, q, usdToINR); err != nil {
			return err
		}
	}
	return nil
}

func (t *Tracker) record(ctx context.Context, l dao.PricedListing, q stores.Quote, usdToINR float64) error {
	prev, lowest, err := t.Store.RecordQuote(ctx, l.ID, q)
	if err != nil {
		return err
	}
	check := alerts.Check{New: q, Previous: prev, LowestMinor: lowest, TargetINR: l.TargetINR}
	if _, ok := ToINRMinor(0, q.Currency, usdToINR); ok {
		check.ToINRMinor = func(minor int64) int64 { paise, _ := ToINRMinor(minor, q.Currency, usdToINR); return paise }
	}
	reasons, key := alerts.Evaluate(check)
	if len(reasons) == 0 {
		return nil
	}
	added, err := t.Store.AddSaleEvent(ctx, l.ID, key, reasons, q.FinalMinor)
	if added {
		t.Log.Info("sale event", "listing", l.ID, "store", l.Store, "reasons", reasons, "final", q.FinalMinor)
	}
	return err
}

// RefreshFX fetches the USD->INR rate if the stored one is stale.
func (t *Tracker) RefreshFX(ctx context.Context) error {
	_, err := t.usdToINR(ctx)
	return err
}

func (t *Tracker) usdToINR(ctx context.Context) (float64, error) {
	rate, fetched, err := t.Store.FXRate(ctx, "USD", "INR")
	if err == nil && time.Since(fetched) < fxMaxAge {
		return rate, nil
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}
	fresh, fetchErr := t.FX.USDToINR(ctx)
	if fetchErr != nil {
		if rate > 0 {
			t.Log.Warn("fx refresh failed, using stale rate", "err", fetchErr)
			return rate, nil
		}
		return 0, fetchErr
	}
	return fresh, t.Store.SetFXRate(ctx, "USD", "INR", fresh)
}

// ToINRMinor converts an amount in minor units of currency to paise. ok is
// false for a currency it can't convert; the target-price check is then
// skipped rather than guessed.
func ToINRMinor(minor int64, currency string, usdToINR float64) (paise int64, ok bool) {
	switch currency {
	case "INR":
		return minor, true
	case "USD":
		return int64(math.Round(float64(minor) * usdToINR)), usdToINR > 0
	default:
		return 0, false
	}
}

// MetaFromGame turns an IGDB game into the metadata stored for a Trove game.
func MetaFromGame(id uuid.UUID, g catalog.Game, method string) model.GameMeta {
	m := model.GameMeta{
		VideoGameID: id,
		MatchMethod: method,
		IGDBID:      &g.IGDBID,
		Title:       &g.Title,
		Platforms:   int32Slice(g.Platforms),
		Developers:  g.Developers,
		Publishers:  g.Publishers,
		Genres:      g.Genres,
		Themes:      g.Themes,
		Keywords:    g.Keywords,
		SimilarIDs:  g.SimilarIDs,
	}
	if g.ReleaseYear > 0 {
		y := int16(g.ReleaseYear)
		m.ReleaseYear = &y
	}
	if g.CoverURL != "" {
		m.CoverURL = &g.CoverURL
	}
	if g.SteamAppID != "" {
		m.SteamAppID = &g.SteamAppID
	}
	if g.Rating > 0 {
		r := float32(g.Rating)
		m.IGDBRating = &r
	}
	return m
}

// UnresolvedMeta records an ambiguous match with the top candidates.
func UnresolvedMeta(id uuid.UUID, title string, candidates []catalog.Game) model.GameMeta {
	ranked := catalog.Rank(catalog.LinkQuery(title), candidates)
	options := make([]model.MatchCandidate, 0, maxCandidates)
	for _, c := range ranked[:min(maxCandidates, len(ranked))] {
		options = append(options, model.MatchCandidate{IGDBID: c.IGDBID, Title: c.Title, Year: c.ReleaseYear, CoverURL: c.CoverURL})
	}
	return model.GameMeta{VideoGameID: id, MatchMethod: model.MatchUnresolved, Candidates: options}
}

func hasStore(listings []model.StoreListing, store string) bool {
	for _, l := range listings {
		if l.Store == store {
			return true
		}
	}
	return false
}

func intSlice(xs []int32) []int {
	out := make([]int, len(xs))
	for i, x := range xs {
		out[i] = int(x)
	}
	return out
}

func int32Slice(xs []int) []int32 {
	out := make([]int32, len(xs))
	for i, x := range xs {
		out[i] = int32(x)
	}
	return out
}
