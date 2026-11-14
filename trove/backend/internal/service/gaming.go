package service

import (
	"context"
	"errors"
	"fmt"
	"time"

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

// ErrInvalidPickAction is returned for an UpdatePick action it doesn't know.
var ErrInvalidPickAction = errors.New("action must be wishlist, own or dismiss")

// ErrInvalidLink is returned for a PS Store URL that isn't a product page.
var ErrInvalidLink = errors.New("not a PS Store product or concept page")

// GamingService serves the price-tracking data the worker gathers, and lets
// you correct what it got wrong (the IGDB match, the PS Store link).
type GamingService struct {
	store *dao.GamingStore
	games *dao.GameStore // to add a game you pick from the weekly picks
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

// ListDeals returns the wishlisted games' discounted listings and the
// USD->INR rate (0 if never fetched) to compare eShop prices with.
func (s *GamingService) ListDeals(ctx context.Context) ([]dao.Deal, float64, error) {
	deals, err := s.store.ListDeals(ctx)
	if err != nil {
		return nil, 0, err
	}
	rate, _, err := s.store.FXRate(ctx, "USD", "INR")
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, 0, err
	}
	return deals, rate, nil
}

// ListPicks returns this week's picks you haven't acted on, and the week
// they're for (zero before the first run).
func (s *GamingService) ListPicks(ctx context.Context) ([]dao.Pick, time.Time, error) {
	picks, err := s.store.CurrentPicks(ctx)
	if err != nil {
		return nil, time.Time{}, err
	}
	week, err := s.store.LastPickRun(ctx)
	return picks, week, err
}

// UpdatePick acts on a pick: "dismiss" turns it down; "wishlist" and "own"
// add the game to Trove with that status, linked to the picked IGDB game,
// and return its id. Each pick can be acted on once (dao.ErrPickTaken).
func (s *GamingService) UpdatePick(ctx context.Context, id uuid.UUID, action string) (*uuid.UUID, error) {
	status, state := "", ""
	switch action {
	case "dismiss":
		return nil, s.store.ClaimPick(ctx, id, dao.PickDismissed)
	case "wishlist":
		status, state = model.GameWishlist, dao.PickWishlisted
	case "own":
		status, state = model.GameOwned, dao.PickOwned
	default:
		return nil, ErrInvalidPickAction
	}
	pick, err := s.store.GetPick(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.store.ClaimPick(ctx, id, state); err != nil {
		return nil, err
	}
	gameID, err := s.addPickedGame(ctx, pick, status)
	if err != nil {
		if undo := s.store.UnclaimPick(ctx, id); undo != nil {
			err = errors.Join(err, undo)
		}
		return nil, err
	}
	return &gameID, s.store.SetPickGame(ctx, id, gameID)
}

// addPickedGame adds a pick to your library and links it to its IGDB game,
// which names it, fills in its cover and starts tracking it.
func (s *GamingService) addPickedGame(ctx context.Context, pick *dao.Pick, status string) (uuid.UUID, error) {
	game, err := s.games.Create(ctx, dao.CreateGameParams{Title: pick.Title, Status: status, YearsPlayed: []int16{}})
	if err != nil {
		return uuid.Nil, err
	}
	if err := s.LinkGame(ctx, game.ID, pick.IGDBID); err != nil {
		return uuid.Nil, fmt.Errorf("link picked game: %w", err)
	}
	return game.ID, nil
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

// enqueueRefresh asks the worker to look at the games in the batch, so a
// game added or changed on the phone gets its match, prices and verdict
// within a minute rather than at the next hourly run. A failure here only
// delays that until the run, so it's ignored.
func enqueueRefresh(ctx context.Context, rdb *redis.Client, games []model.VideoGame) {
	var ids []uuid.UUID
	for _, g := range games {
		if !g.Deleted {
			ids = append(ids, g.ID)
		}
	}
	jobs.EnqueueRefresh(ctx, rdb, ids...) //nolint:errcheck
}
