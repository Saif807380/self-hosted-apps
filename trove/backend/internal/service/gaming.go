package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"github.com/saifkazi/trove/backend/internal/catalog"
	"github.com/saifkazi/trove/backend/internal/dao"
	"github.com/saifkazi/trove/backend/internal/jobs"
	"github.com/saifkazi/trove/backend/internal/model"
	"github.com/saifkazi/trove/backend/internal/push"
	"github.com/saifkazi/trove/backend/internal/stores"
	"github.com/saifkazi/trove/backend/internal/tracker"
)

const catalogSearchLimit = 8

// ErrInvalidLink is returned for a PS Store URL that isn't a product page.
var ErrInvalidLink = errors.New("not a PS Store product or concept page")

// GamingService serves the price-tracking data the worker gathers, and lets
// you correct what it got wrong (the IGDB match, the PS Store link).
type GamingService struct {
	store *dao.GamingStore
	igdb  *catalog.Client
	redis *redis.Client
	push  *push.Sender
	// verdicts: the worker has an API key, so verdicts are being made.
	verdicts bool
}

// GameInsights is everything the game detail view shows about prices.
type GameInsights struct {
	Meta     *model.GameMeta // nil if the game was never looked up
	Listings []model.StoreListing
	History  []model.PriceRun
	USDToINR float64
	Verdict  *dao.StoredVerdict // nil until the worker has made one
	// VerdictsEnabled is false when no verdict is coming (no API key).
	VerdictsEnabled bool
}

func (s *GamingService) SearchCatalog(ctx context.Context, query string) ([]catalog.Game, error) {
	return s.igdb.Search(ctx, query, catalogSearchLimit)
}

// LinkGame records your choice of IGDB game and queues a refresh so listings
// and prices follow within a minute.
func (s *GamingService) LinkGame(ctx context.Context, gameID uuid.UUID, igdbID int64) error {
	if _, err := s.store.GetGameRef(ctx, gameID); err != nil {
		return err
	}
	games, err := s.igdb.Get(ctx, []int64{igdbID})
	if err != nil {
		return err
	}
	if len(games) == 0 {
		return fmt.Errorf("igdb game %d: %w", igdbID, pgx.ErrNoRows)
	}
	g, err := s.igdb.EnsureSteamAppID(ctx, games[0])
	if err != nil {
		return err
	}
	if err := s.store.SaveMeta(ctx, tracker.MetaFromGame(gameID, g, model.MatchPicked)); err != nil {
		return err
	}
	// Take the chosen game's proper name, as an automatic match does.
	if err := s.store.SetTitle(ctx, gameID, g.Title); err != nil {
		return err
	}
	invalidateGameLists(ctx, s.redis)
	return jobs.EnqueueRefresh(ctx, s.redis, gameID)
}

func (s *GamingService) RefreshGame(ctx context.Context, gameID uuid.UUID) error {
	if _, err := s.store.GetGameRef(ctx, gameID); err != nil {
		return err
	}
	return jobs.EnqueueRefresh(ctx, s.redis, gameID)
}

// SetPSNLink points the game's PS Store button at a pasted product page, or
// back at a title search when url is empty.
func (s *GamingService) SetPSNLink(ctx context.Context, gameID uuid.UUID, url string) error {
	ref, err := s.store.GetGameRef(ctx, gameID)
	if err != nil {
		return err
	}
	title := ref.Title
	if meta, err := s.store.GetMeta(ctx, gameID); err == nil && meta.Title != nil {
		title = *meta.Title
	}
	listing := stores.PSNSearchListing(title)
	if url != "" {
		var ok bool
		if listing, ok = stores.PSNProductListing(url, title); !ok {
			return ErrInvalidLink
		}
	}
	return s.store.SetPSNListing(ctx, gameID, listing)
}

func (s *GamingService) GetGameInsights(ctx context.Context, gameID uuid.UUID) (*GameInsights, error) {
	if _, err := s.store.GetGameRef(ctx, gameID); err != nil {
		return nil, err
	}
	out := &GameInsights{VerdictsEnabled: s.verdicts}
	meta, err := s.store.GetMeta(ctx, gameID)
	switch {
	case err == nil:
		out.Meta = meta
	case !errors.Is(err, pgx.ErrNoRows):
		return nil, err
	}
	if out.Listings, err = s.store.Listings(ctx, gameID); err != nil {
		return nil, err
	}
	if out.History, err = s.store.PriceHistory(ctx, gameID); err != nil {
		return nil, err
	}
	if rate, _, err := s.store.FXRate(ctx, "USD", "INR"); err == nil {
		out.USDToINR = rate
	}
	verdict, err := s.store.GetVerdict(ctx, gameID)
	switch {
	case err == nil:
		out.Verdict = verdict
	case !errors.Is(err, pgx.ErrNoRows):
		return nil, err
	}
	return out, nil
}

// PushConfig returns whether alerts can be turned on, and the public key a
// browser subscribes with.
func (s *GamingService) PushConfig() (enabled bool, publicKey string) {
	return s.push.Enabled(), s.push.PublicKey
}

func (s *GamingService) SubscribePush(ctx context.Context, sub dao.PushSubscription, userAgent string) error {
	return s.store.SavePushSubscription(ctx, sub, userAgent)
}

func (s *GamingService) UnsubscribePush(ctx context.Context, endpoint string) error {
	return s.store.DeletePushSubscription(ctx, endpoint)
}

// SendTestPush checks the whole path to your devices.
func (s *GamingService) SendTestPush(ctx context.Context) (int, error) {
	return s.push.Send(ctx, push.Notification{
		Title: "Sale alerts are on",
		Body:  "Trove will tell you here when a wishlisted game goes on sale.",
		URL:   "/games?tab=wishlist",
		Icon:  "/icon-192.png",
		Tag:   "trove-test",
	})
}

// enqueueWishlisted asks the worker to start tracking any wishlisted game in
// the batch, so a game added on the phone gets prices within a minute rather
// than at the next hourly run. A failure here only delays tracking until that
// run, so it's ignored.
func enqueueWishlisted(ctx context.Context, rdb *redis.Client, games []model.VideoGame) {
	var ids []uuid.UUID
	for _, g := range games {
		if g.Status == model.GameWishlist && !g.Deleted {
			ids = append(ids, g.ID)
		}
	}
	jobs.EnqueueRefresh(ctx, rdb, ids...) //nolint:errcheck
}
