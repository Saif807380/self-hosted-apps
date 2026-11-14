package dao

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/saifkazi/trove/backend/internal/model"
	"github.com/saifkazi/trove/backend/internal/stores"
)

// GamingStore holds the worker-written price-tracking data: IGDB metadata,
// store listings, price history, sale events and exchange rates.
type GamingStore struct {
	db *pgxpool.Pool
}

// listingsRefresh is how long store listings are trusted before being looked
// up again (a Switch 2 Edition can appear months after launch).
const listingsRefresh = 30 * 24 * time.Hour

const metaColumns = `video_game_id, match_method, igdb_id, title, release_year, cover_url, steam_appid,
	platforms, developers, publishers, genres, themes, keywords, igdb_rating, similar_ids, candidates,
	listings_at, updated_at`

const listingColumns = "id, video_game_id, store, store_ref, edition, title, url, region, currency, created_at"

// GameRef is the little the worker needs to know about a Trove game.
type GameRef struct {
	ID     uuid.UUID `db:"id"`
	Title  string    `db:"title"`
	Studio *string   `db:"studio"`
}

// GamesToLink returns games that have never been looked up in IGDB.
func (s *GamingStore) GamesToLink(ctx context.Context, limit int) ([]GameRef, error) {
	rows, err := s.db.Query(ctx, `
		SELECT g.id, g.title, g.studio FROM video_games g
		WHERE g.deleted = false AND NOT EXISTS (SELECT 1 FROM game_meta m WHERE m.video_game_id = g.id)
		ORDER BY (g.status = 'wishlist') DESC, g.created_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("games to link: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[GameRef])
}

// GetGameRef returns one game's reference, or pgx.ErrNoRows.
func (s *GamingStore) GetGameRef(ctx context.Context, id uuid.UUID) (GameRef, error) {
	rows, err := s.db.Query(ctx, "SELECT id, title, studio FROM video_games WHERE id = $1 AND deleted = false", id)
	if err != nil {
		return GameRef{}, fmt.Errorf("get game ref: %w", err)
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[GameRef])
}

// GetMeta returns a game's IGDB metadata, or pgx.ErrNoRows if never linked.
func (s *GamingStore) GetMeta(ctx context.Context, id uuid.UUID) (*model.GameMeta, error) {
	rows, err := s.db.Query(ctx, "SELECT "+metaColumns+" FROM game_meta WHERE video_game_id = $1", id)
	if err != nil {
		return nil, fmt.Errorf("get meta: %w", err)
	}
	m, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.GameMeta])
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// SaveMeta stores a game's IGDB link. Linking to a different IGDB game drops
// the old store listings and their price history, since they belong to the
// other game.
func (s *GamingStore) SaveMeta(ctx context.Context, m model.GameMeta) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var previous *int64
	err = tx.QueryRow(ctx, "SELECT igdb_id FROM game_meta WHERE video_game_id = $1 FOR UPDATE", m.VideoGameID).Scan(&previous)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("read meta: %w", err)
	}
	relinked := previous != nil && (m.IGDBID == nil || *previous != *m.IGDBID)
	if relinked {
		// Both were about the old game.
		if _, err := tx.Exec(ctx, "DELETE FROM store_listings WHERE video_game_id = $1", m.VideoGameID); err != nil {
			return fmt.Errorf("drop old listings: %w", err)
		}
		if _, err := tx.Exec(ctx, "DELETE FROM device_verdicts WHERE video_game_id = $1", m.VideoGameID); err != nil {
			return fmt.Errorf("drop old verdict: %w", err)
		}
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO game_meta (video_game_id, match_method, igdb_id, title, release_year, cover_url, steam_appid,
			platforms, genres, themes, keywords, igdb_rating, similar_ids, candidates, listings_at, updated_at,
			developers, publishers)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NULL, now(), $16, $17)
		ON CONFLICT (video_game_id) DO UPDATE SET
			match_method = EXCLUDED.match_method, igdb_id = EXCLUDED.igdb_id, title = EXCLUDED.title,
			release_year = EXCLUDED.release_year, cover_url = EXCLUDED.cover_url, steam_appid = EXCLUDED.steam_appid,
			platforms = EXCLUDED.platforms, genres = EXCLUDED.genres, themes = EXCLUDED.themes,
			keywords = EXCLUDED.keywords, igdb_rating = EXCLUDED.igdb_rating, similar_ids = EXCLUDED.similar_ids,
			candidates = EXCLUDED.candidates, developers = EXCLUDED.developers, publishers = EXCLUDED.publishers,
			listings_at = CASE WHEN $15 THEN NULL ELSE game_meta.listings_at END,
			updated_at = now()`,
		m.VideoGameID, m.MatchMethod, m.IGDBID, m.Title, m.ReleaseYear, m.CoverURL, m.SteamAppID,
		nonNil(m.Platforms), nonNil(m.Genres), nonNil(m.Themes), nonNil(m.Keywords), m.IGDBRating,
		nonNil(m.SimilarIDs), candidatesOrNull(m.Candidates), relinked, nonNil(m.Developers), nonNil(m.Publishers))
	if err != nil {
		return fmt.Errorf("save meta: %w", err)
	}
	return tx.Commit(ctx)
}

// ListingTarget is a wishlisted, linked game whose store listings are due a
// lookup.
type ListingTarget struct {
	ID         uuid.UUID `db:"id"`
	Title      string    `db:"title"`
	SteamAppID *string   `db:"steam_appid"`
	Platforms  []int32   `db:"platforms"`
}

// GamesNeedingListings returns wishlisted games that are linked to IGDB but
// whose listings were never looked up, or not recently. With id set, only
// that game is considered and its age is ignored.
func (s *GamingStore) GamesNeedingListings(ctx context.Context, id *uuid.UUID) ([]ListingTarget, error) {
	rows, err := s.db.Query(ctx, `
		SELECT g.id, m.title, m.steam_appid, m.platforms
		FROM video_games g JOIN game_meta m ON m.video_game_id = g.id
		WHERE g.deleted = false AND g.status = 'wishlist' AND m.match_method <> 'unresolved'
			AND ($1::uuid IS NULL OR g.id = $1)
			AND ($1::uuid IS NOT NULL OR m.listings_at IS NULL OR m.listings_at < $2)`,
		id, time.Now().Add(-listingsRefresh))
	if err != nil {
		return nil, fmt.Errorf("games needing listings: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[ListingTarget])
}

// SaveListings records the listings found for a game and marks the lookup
// done. Existing listings are updated, never removed: a listing that stops
// appearing in search keeps its price history.
func (s *GamingStore) SaveListings(ctx context.Context, gameID uuid.UUID, listings []stores.Listing) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	for _, l := range listings {
		if err := upsertListing(ctx, tx, gameID, l); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, "UPDATE game_meta SET listings_at = now() WHERE video_game_id = $1", gameID); err != nil {
		return fmt.Errorf("mark listings: %w", err)
	}
	return tx.Commit(ctx)
}

// SetPSNListing replaces a game's PS Store link (search or product page).
func (s *GamingStore) SetPSNListing(ctx context.Context, gameID uuid.UUID, l stores.Listing) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, "DELETE FROM store_listings WHERE video_game_id = $1 AND store = $2", gameID, stores.PSN); err != nil {
		return fmt.Errorf("drop psn listing: %w", err)
	}
	if err := upsertListing(ctx, tx, gameID, l); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func upsertListing(ctx context.Context, tx pgx.Tx, gameID uuid.UUID, l stores.Listing) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO store_listings (video_game_id, store, store_ref, edition, title, url, region, currency)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (video_game_id, store, store_ref) DO UPDATE SET
			edition = EXCLUDED.edition, title = EXCLUDED.title, url = EXCLUDED.url`,
		gameID, l.Store, l.Ref, l.Edition, l.Title, l.URL, l.Region, l.Currency)
	if err != nil {
		return fmt.Errorf("upsert listing: %w", err)
	}
	return nil
}

// PricedListing is a listing due a price check, with what the alert rules
// need to know about its game.
type PricedListing struct {
	ID          uuid.UUID `db:"id"`
	VideoGameID uuid.UUID `db:"video_game_id"`
	Store       string    `db:"store"`
	StoreRef    string    `db:"store_ref"`
	Currency    string    `db:"currency"`
	TargetINR   *int32    `db:"target_price_inr"`
}

// ListingsToPrice returns the Steam and eShop listings of wishlisted games.
// With id set, only that game's listings.
func (s *GamingStore) ListingsToPrice(ctx context.Context, id *uuid.UUID) ([]PricedListing, error) {
	rows, err := s.db.Query(ctx, `
		SELECT l.id, l.video_game_id, l.store, l.store_ref, l.currency, g.target_price_inr
		FROM store_listings l JOIN video_games g ON g.id = l.video_game_id
		WHERE g.deleted = false AND g.status = 'wishlist' AND l.store IN ('steam', 'eshop') AND l.store_ref <> ''
			AND ($1::uuid IS NULL OR g.id = $1)`, id)
	if err != nil {
		return nil, fmt.Errorf("listings to price: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[PricedListing])
}

// RecordQuote adds a price observation to a listing's history. It returns
// the previous quote (nil on the first observation) and the lowest final
// price recorded before this one (0 if none), for the alert rules.
func (s *GamingStore) RecordQuote(ctx context.Context, listingID uuid.UUID, q stores.Quote) (*stores.Quote, int64, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var lowest *int64
	if err := tx.QueryRow(ctx, "SELECT min(final_minor) FROM price_quotes WHERE listing_id = $1", listingID).Scan(&lowest); err != nil {
		return nil, 0, fmt.Errorf("lowest price: %w", err)
	}

	var prev model.PriceRun
	rows, err := tx.Query(ctx, `
		SELECT id, listing_id, base_minor, final_minor, discount_pct, sale_ends_at, first_seen_at, last_seen_at
		FROM price_quotes WHERE listing_id = $1 ORDER BY first_seen_at DESC LIMIT 1 FOR UPDATE`, listingID)
	if err != nil {
		return nil, 0, fmt.Errorf("latest price: %w", err)
	}
	prev, err = pgx.CollectOneRow(rows, pgx.RowToStructByName[model.PriceRun])
	hasPrev := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, 0, fmt.Errorf("latest price: %w", err)
	}

	if hasPrev && sameRun(prev, q) {
		_, err = tx.Exec(ctx, "UPDATE price_quotes SET last_seen_at = $2 WHERE id = $1", prev.ID, q.CheckedAt)
	} else {
		_, err = tx.Exec(ctx, `
			INSERT INTO price_quotes (listing_id, base_minor, final_minor, discount_pct, sale_ends_at, first_seen_at, last_seen_at)
			VALUES ($1, $2, $3, $4, $5, $6, $6)`,
			listingID, q.BaseMinor, q.FinalMinor, q.DiscountPct, q.SaleEndsAt, q.CheckedAt)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("record price: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, 0, fmt.Errorf("commit: %w", err)
	}

	var lowestBefore int64
	if lowest != nil {
		lowestBefore = *lowest
	}
	if !hasPrev {
		return nil, lowestBefore, nil
	}
	return &stores.Quote{
		Ref: q.Ref, Currency: q.Currency, BaseMinor: prev.BaseMinor, FinalMinor: prev.FinalMinor,
		DiscountPct: int(prev.DiscountPct), SaleEndsAt: prev.SaleEndsAt, CheckedAt: prev.LastSeenAt,
	}, lowestBefore, nil
}

func sameRun(r model.PriceRun, q stores.Quote) bool {
	sameEnd := (r.SaleEndsAt == nil && q.SaleEndsAt == nil) ||
		(r.SaleEndsAt != nil && q.SaleEndsAt != nil && r.SaleEndsAt.Equal(*q.SaleEndsAt))
	return r.BaseMinor == q.BaseMinor && r.FinalMinor == q.FinalMinor && sameEnd
}

// AddSaleEvent records a sale for the push sender. It reports false when the
// same sale (listing + key) was already recorded.
func (s *GamingStore) AddSaleEvent(ctx context.Context, listingID uuid.UUID, saleKey string, reasons []string, q stores.Quote) (bool, error) {
	tag, err := s.db.Exec(ctx, `
		INSERT INTO sale_events (listing_id, sale_key, reasons, final_minor, base_minor, discount_pct)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (listing_id, sale_key) DO NOTHING`, listingID, saleKey, reasons, q.FinalMinor, q.BaseMinor, q.DiscountPct)
	if err != nil {
		return false, fmt.Errorf("add sale event: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// SetFXRate stores the latest rate for base->quote.
func (s *GamingStore) SetFXRate(ctx context.Context, base, quote string, rate float64) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO fx_rates (base, quote, rate, fetched_at) VALUES ($1, $2, $3, now())
		ON CONFLICT (base, quote) DO UPDATE SET rate = EXCLUDED.rate, fetched_at = now()`, base, quote, rate)
	if err != nil {
		return fmt.Errorf("set fx rate: %w", err)
	}
	return nil
}

// FXRate returns the stored rate for base->quote and when it was fetched, or
// pgx.ErrNoRows if it never was.
func (s *GamingStore) FXRate(ctx context.Context, base, quote string) (float64, time.Time, error) {
	var rate float64
	var fetched time.Time
	err := s.db.QueryRow(ctx, "SELECT rate, fetched_at FROM fx_rates WHERE base = $1 AND quote = $2", base, quote).Scan(&rate, &fetched)
	return rate, fetched, err
}

// Listings returns a game's store listings, Steam first, then eShop, then PS.
func (s *GamingStore) Listings(ctx context.Context, gameID uuid.UUID) ([]model.StoreListing, error) {
	rows, err := s.db.Query(ctx, "SELECT "+listingColumns+` FROM store_listings WHERE video_game_id = $1
		ORDER BY array_position(ARRAY['steam','eshop','psn']::varchar[], store), edition DESC`, gameID)
	if err != nil {
		return nil, fmt.Errorf("listings: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[model.StoreListing])
}

// PriceHistory returns every price run of a game's listings, oldest first.
func (s *GamingStore) PriceHistory(ctx context.Context, gameID uuid.UUID) ([]model.PriceRun, error) {
	rows, err := s.db.Query(ctx, `
		SELECT q.id, q.listing_id, q.base_minor, q.final_minor, q.discount_pct, q.sale_ends_at, q.first_seen_at, q.last_seen_at
		FROM price_quotes q JOIN store_listings l ON l.id = q.listing_id
		WHERE l.video_game_id = $1 ORDER BY q.first_seen_at`, gameID)
	if err != nil {
		return nil, fmt.Errorf("price history: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[model.PriceRun])
}

// nonNil keeps NULL out of NOT NULL array columns.
func nonNil[T any](xs []T) []T {
	if xs == nil {
		return []T{}
	}
	return xs
}

func candidatesOrNull(c []model.MatchCandidate) any {
	if len(c) == 0 {
		return nil
	}
	return c
}

// CoverTarget is a game with no cover of its own whose IGDB match has one.
type CoverTarget struct {
	ID       uuid.UUID `db:"id"`
	CoverURL string    `db:"cover_url"`
}

// GamesMissingCover returns games without a cover whose IGDB match has one.
// With id set, only that game.
func (s *GamingStore) GamesMissingCover(ctx context.Context, id *uuid.UUID) ([]CoverTarget, error) {
	rows, err := s.db.Query(ctx, `
		SELECT g.id, m.cover_url FROM video_games g JOIN game_meta m ON m.video_game_id = g.id
		WHERE g.deleted = false AND g.cover_image IS NULL AND m.cover_url IS NOT NULL
			AND m.match_method <> 'unresolved' AND ($1::uuid IS NULL OR g.id = $1)`, id)
	if err != nil {
		return nil, fmt.Errorf("games missing cover: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[CoverTarget])
}

// SetCoverIfEmpty gives a game a cover only if it still has none, so a cover
// you uploaded meanwhile is never replaced. It bumps updated_at so the cover
// reaches your devices on their next sync. It reports whether it was set.
//
// It and SetStudioIfEmpty are the worker's only writes to video_games: both
// fill a blank field and never overwrite one. (SetTitle is the API's, when
// you pick a match.)
func (s *GamingStore) SetCoverIfEmpty(ctx context.Context, id uuid.UUID, path string) (bool, error) {
	tag, err := s.db.Exec(ctx, `
		UPDATE video_games SET cover_image = $2, updated_at = now()
		WHERE id = $1 AND cover_image IS NULL AND deleted = false`, id, path)
	if err != nil {
		return false, fmt.Errorf("set cover: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// SetTitle renames a game, e.g. to the IGDB title you picked. It bumps
// updated_at so the new name reaches your devices on their next sync.
func (s *GamingStore) SetTitle(ctx context.Context, id uuid.UUID, title string) error {
	_, err := s.db.Exec(ctx, "UPDATE video_games SET title = $2, updated_at = now() WHERE id = $1 AND deleted = false", id, title)
	if err != nil {
		return fmt.Errorf("set title: %w", err)
	}
	return nil
}

// StudioTarget is a game with no studio whose IGDB match names its makers.
type StudioTarget struct {
	ID         uuid.UUID `db:"id"`
	Developers []string  `db:"developers"`
	Publishers []string  `db:"publishers"`
}

// GamesMissingStudio returns games with a blank studio whose IGDB match lists
// a publisher or developer. With id set, only that game.
func (s *GamingStore) GamesMissingStudio(ctx context.Context, id *uuid.UUID) ([]StudioTarget, error) {
	rows, err := s.db.Query(ctx, `
		SELECT g.id, m.developers, m.publishers FROM video_games g JOIN game_meta m ON m.video_game_id = g.id
		WHERE g.deleted = false AND coalesce(btrim(g.studio), '') = '' AND m.match_method <> 'unresolved'
			AND (cardinality(m.publishers) > 0 OR cardinality(m.developers) > 0)
			AND ($1::uuid IS NULL OR g.id = $1)`, id)
	if err != nil {
		return nil, fmt.Errorf("games missing studio: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[StudioTarget])
}

// SetStudioIfEmpty fills a blank studio, like SetCoverIfEmpty does covers.
func (s *GamingStore) SetStudioIfEmpty(ctx context.Context, id uuid.UUID, studio string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE video_games SET studio = $2, updated_at = now()
		WHERE id = $1 AND coalesce(btrim(studio), '') = '' AND deleted = false`, id, studio)
	if err != nil {
		return fmt.Errorf("set studio: %w", err)
	}
	return nil
}

// PushSubscription is a browser that turned on sale alerts.
type PushSubscription struct {
	Endpoint string `db:"endpoint"`
	P256dh   string `db:"p256dh"`
	Auth     string `db:"auth"`
}

// SavePushSubscription stores (or refreshes) a browser's subscription.
func (s *GamingStore) SavePushSubscription(ctx context.Context, sub PushSubscription, userAgent string) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO push_subscriptions (endpoint, p256dh, auth, user_agent) VALUES ($1, $2, $3, NULLIF($4, ''))
		ON CONFLICT (endpoint) DO UPDATE SET p256dh = EXCLUDED.p256dh, auth = EXCLUDED.auth, user_agent = EXCLUDED.user_agent`,
		sub.Endpoint, sub.P256dh, sub.Auth, truncate(userAgent, 500))
	if err != nil {
		return fmt.Errorf("save push subscription: %w", err)
	}
	return nil
}

// DeletePushSubscription forgets a browser's subscription.
func (s *GamingStore) DeletePushSubscription(ctx context.Context, endpoint string) error {
	if _, err := s.db.Exec(ctx, "DELETE FROM push_subscriptions WHERE endpoint = $1", endpoint); err != nil {
		return fmt.Errorf("delete push subscription: %w", err)
	}
	return nil
}

// PushSubscriptions returns every subscribed browser.
func (s *GamingStore) PushSubscriptions(ctx context.Context) ([]PushSubscription, error) {
	rows, err := s.db.Query(ctx, "SELECT endpoint, p256dh, auth FROM push_subscriptions")
	if err != nil {
		return nil, fmt.Errorf("push subscriptions: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[PushSubscription])
}

// MarkPushOK records a successful delivery to a subscription.
func (s *GamingStore) MarkPushOK(ctx context.Context, endpoint string) error {
	_, err := s.db.Exec(ctx, "UPDATE push_subscriptions SET last_ok_at = now() WHERE endpoint = $1", endpoint)
	if err != nil {
		return fmt.Errorf("mark push ok: %w", err)
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// PendingSale is an undelivered sale event with what the notification says.
type PendingSale struct {
	ID          int64     `db:"id"`
	GameID      uuid.UUID `db:"video_game_id"`
	GameTitle   string    `db:"game_title"`
	CoverImage  *string   `db:"cover_image"`
	TargetINR   *int32    `db:"target_price_inr"`
	StillWanted bool      `db:"still_wanted"` // wishlisted and not deleted
	Store       string    `db:"store"`
	Edition     string    `db:"edition"`
	Currency    string    `db:"currency"`
	Reasons     []string  `db:"reasons"`
	FinalMinor  int64     `db:"final_minor"`
	BaseMinor   *int64    `db:"base_minor"`
	DiscountPct *int16    `db:"discount_pct"`
	CreatedAt   time.Time `db:"created_at"`
}

// PendingSales returns sale events not yet delivered, oldest first.
func (s *GamingStore) PendingSales(ctx context.Context) ([]PendingSale, error) {
	rows, err := s.db.Query(ctx, `
		SELECT e.id, l.video_game_id, g.title AS game_title, g.cover_image, g.target_price_inr,
			(g.status = 'wishlist' AND NOT g.deleted) AS still_wanted,
			l.store, l.edition, l.currency, e.reasons, e.final_minor, e.base_minor, e.discount_pct, e.created_at
		FROM sale_events e
		JOIN store_listings l ON l.id = e.listing_id
		JOIN video_games g ON g.id = l.video_game_id
		WHERE e.notified_at IS NULL
		ORDER BY e.created_at`)
	if err != nil {
		return nil, fmt.Errorf("pending sales: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[PendingSale])
}

// MarkSalesNotified closes sale events, delivered or deliberately skipped.
func (s *GamingStore) MarkSalesNotified(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	if _, err := s.db.Exec(ctx, "UPDATE sale_events SET notified_at = now() WHERE id = ANY($1)", ids); err != nil {
		return fmt.Errorf("mark sales notified: %w", err)
	}
	return nil
}
