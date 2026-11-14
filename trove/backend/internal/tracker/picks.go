package tracker

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/saifkazi/trove/backend/internal/catalog"
	"github.com/saifkazi/trove/backend/internal/dao"
	"github.com/saifkazi/trove/backend/internal/picks"
	"github.com/saifkazi/trove/backend/internal/push"
	"github.com/saifkazi/trove/backend/internal/stores"
)

const (
	picksPerWeek  = 5
	shortlistSize = 15 // the model chooses the week's picks from these
	// How many of each store's current sales to consider, most popular
	// first. Every eShop one costs an IGDB search, so fewer of those.
	steamCandidates = 300
	eshopCandidates = 120
	// minRatings keeps out games too obscure to judge.
	minRatings = 10
)

// ist is India Standard Time, fixed (the container has no time zone data).
var ist = time.FixedZone("IST", 5*3600+30*60)

// PickWeek returns the day the picks week containing t started: picks are
// made on Sundays at 09:00 IST.
func PickWeek(t time.Time) time.Time {
	local := t.In(ist)
	start := time.Date(local.Year(), local.Month(), local.Day(), 9, 0, 0, 0, ist).
		AddDate(0, 0, -int(local.Weekday()))
	if start.After(local) {
		start = start.AddDate(0, 0, -7)
	}
	return time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
}

// MakePicksIfDue makes this week's picks if they haven't been made yet.
func (t *Tracker) MakePicksIfDue(ctx context.Context) error {
	week := PickWeek(time.Now())
	last, err := t.Store.LastPickRun(ctx)
	if err != nil || !last.Before(week) {
		return err
	}
	return t.MakePicks(ctx, week)
}

// MakePicks chooses the week's picks: games on sale now, scored against
// your taste, the best few chosen and pitched by the model. Then it tells
// your devices. A run that finds nothing is still recorded, so it isn't
// retried every hour; a run that can't reach the stores is.
func (t *Tracker) MakePicks(ctx context.Context, week time.Time) error {
	library, err := t.Store.Library(ctx)
	if err != nil {
		return err
	}
	dismissed, err := t.Store.DismissedPickTags(ctx)
	if err != nil {
		return err
	}
	profile := picks.Build(library, dismissed, time.Now().In(ist).Year())
	if !profile.Ready() {
		t.Log.Info("picks skipped: not enough liked games yet")
		return t.Store.SavePicks(ctx, week, nil)
	}
	excluded, err := t.Store.ExcludedFromPicks(ctx)
	if err != nil {
		return err
	}
	candidates, err := t.pickCandidates(ctx, excluded)
	if err != nil {
		return err
	}
	shortlist := picks.Shortlist(profile, candidates, shortlistSize)

	var chosen []picks.Pick
	if t.LLM != nil {
		if chosen, err = picks.Choose(ctx, t.LLM, profile, shortlist, picksPerWeek); err != nil {
			t.Log.Warn("model couldn't choose picks; using the top scores", "err", err)
		}
	}
	if chosen == nil {
		chosen = picks.Fallback(profile, shortlist, picksPerWeek)
	}

	rows := make([]dao.Pick, len(chosen))
	for i, p := range chosen {
		rows[i] = pickRow(i+1, p)
	}
	if err := t.Store.SavePicks(ctx, week, rows); err != nil {
		return err
	}
	t.Log.Info("picks made", "week", week.Format(time.DateOnly), "candidates", len(candidates), "picks", len(rows))
	t.announcePicks(ctx, rows)
	return nil
}

// pickCandidates gathers games on sale in both stores, matched to IGDB and
// not excluded. One store failing is tolerated; both is an error.
func (t *Tracker) pickCandidates(ctx context.Context, excluded map[int64]bool) ([]picks.Candidate, error) {
	steam, steamErr := t.steamCandidates(ctx)
	if steamErr != nil {
		t.Log.Warn("steam candidates for picks", "err", steamErr)
	}
	eshop, eshopErr := t.eshopCandidates(ctx)
	if eshopErr != nil {
		t.Log.Warn("eshop candidates for picks", "err", eshopErr)
	}
	if steamErr != nil && eshopErr != nil {
		return nil, errors.Join(steamErr, eshopErr)
	}
	// One candidate per game: the deeper discount when both stores have it.
	best := map[int64]picks.Candidate{}
	var order []int64
	for _, c := range append(steam, eshop...) {
		g := c.Game
		if excluded[g.IGDBID] || len(g.Genres) == 0 || g.RatingCount < minRatings {
			continue
		}
		prev, seen := best[g.IGDBID]
		if !seen {
			order = append(order, g.IGDBID)
		}
		if !seen || c.Quote.DiscountPct > prev.Quote.DiscountPct {
			best[g.IGDBID] = c
		}
	}
	out := make([]picks.Candidate, len(order))
	for i, id := range order {
		out[i] = best[id]
	}
	return out, nil
}

func (t *Tracker) steamCandidates(ctx context.Context) ([]picks.Candidate, error) {
	listings, err := t.Steam.Specials(ctx, steamCandidates)
	if err != nil {
		return nil, err
	}
	byApp := map[string]stores.Listing{}
	appids := make([]string, len(listings))
	for i, l := range listings {
		byApp[l.Ref], appids[i] = l, l.Ref
	}
	quotes, err := t.Steam.Quotes(ctx, appids)
	if err != nil {
		return nil, err
	}
	var onSale []string
	for id, q := range quotes {
		if q.OnSale() {
			onSale = append(onSale, id)
		}
	}
	games, err := t.IGDB.BySteamAppIDs(ctx, onSale)
	if err != nil {
		return nil, err
	}
	var out []picks.Candidate
	for _, g := range games {
		if q, ok := quotes[g.SteamAppID]; ok && q.OnSale() {
			out = append(out, picks.Candidate{Game: g, Listing: byApp[g.SteamAppID], Quote: q})
		}
	}
	return out, nil
}

func (t *Tracker) eshopCandidates(ctx context.Context) ([]picks.Candidate, error) {
	listings, err := t.Eshop.Deals(ctx, eshopCandidates)
	if err != nil {
		return nil, err
	}
	nsuids := make([]string, len(listings))
	for i, l := range listings {
		nsuids[i] = l.Ref
	}
	quotes, err := t.Eshop.Quotes(ctx, nsuids)
	if err != nil {
		return nil, err
	}
	var out []picks.Candidate
	for _, l := range listings {
		q, ok := quotes[l.Ref]
		if !ok || !q.OnSale() {
			continue
		}
		found, err := t.IGDB.Search(ctx, catalog.LinkQuery(l.Title), catalog.LinkPool)
		if err != nil {
			return nil, err
		}
		if g, ok := catalog.ChooseMatch(l.Title, "", onYourDevices(found)); ok {
			out = append(out, picks.Candidate{Game: g, Listing: l, Quote: q})
		}
	}
	return out, nil
}

func pickRow(rank int, p picks.Pick) dao.Pick {
	g, l, q := p.Game, p.Listing, p.Quote
	row := dao.Pick{
		Rank: int16(rank), IGDBID: g.IGDBID, Title: g.Title, Pitch: p.Pitch, Tags: picks.Tags(g),
		Store: l.Store, StoreRef: l.Ref, Edition: l.Edition, URL: l.URL, Currency: q.Currency,
		BaseMinor: q.BaseMinor, FinalMinor: q.FinalMinor, DiscountPct: int16(q.DiscountPct), SaleEndsAt: q.SaleEndsAt,
	}
	if g.ReleaseYear > 0 {
		year := int16(g.ReleaseYear)
		row.Year = &year
	}
	if g.CoverURL != "" {
		row.CoverURL = &g.CoverURL
	}
	return row
}

// announcePicks sends one notification for the week's picks.
func (t *Tracker) announcePicks(ctx context.Context, rows []dao.Pick) {
	if len(rows) == 0 || t.Notify == nil || !t.Notify.Enabled() {
		return
	}
	names := make([]string, 0, 3)
	for _, r := range rows[:min(3, len(rows))] {
		names = append(names, fmt.Sprintf("%s −%d%%", r.Title, r.DiscountPct))
	}
	body := strings.Join(names, " · ")
	if len(rows) > 3 {
		body += fmt.Sprintf(" and %d more", len(rows)-3)
	}
	icon := "/icon-192.png"
	if rows[0].CoverURL != nil {
		icon = *rows[0].CoverURL
	}
	delivered, err := t.Notify.Send(ctx, push.Notification{
		Title: fmt.Sprintf("This week's picks: %d games on sale", len(rows)),
		Body:  body, URL: "/games?tab=deals", Icon: icon, Tag: "trove-picks",
	})
	if err != nil {
		t.Log.Warn("picks notification failed", "err", err)
		return
	}
	t.Log.Info("picks notification sent", "devices", delivered)
}
