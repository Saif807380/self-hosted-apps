package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// dealMaxAge is how recently a sale price must have been seen to be shown:
// prices are checked every 6 hours, so an older one may have ended.
const dealMaxAge = 2 * 24 * time.Hour

// Deal is a wishlisted game's listing that's discounted right now.
type Deal struct {
	VideoGameID uuid.UUID  `db:"video_game_id"`
	Store       string     `db:"store"`
	Edition     string     `db:"edition"`
	URL         string     `db:"url"`
	Currency    string     `db:"currency"`
	BaseMinor   int64      `db:"base_minor"`
	FinalMinor  int64      `db:"final_minor"`
	DiscountPct int16      `db:"discount_pct"`
	SaleEndsAt  *time.Time `db:"sale_ends_at"`
	// Lowest: no lower price has been recorded for this listing, which has
	// more than one price on record (a first price isn't a record low).
	Lowest bool `db:"lowest"`
}

// ListDeals returns the discounted listings of wishlisted games: each
// listing's latest price, when it's below the base price, was seen within
// dealMaxAge and its sale hasn't ended.
func (s *GamingStore) ListDeals(ctx context.Context) ([]Deal, error) {
	rows, err := s.db.Query(ctx, `
		SELECT video_game_id, store, edition, url, currency, base_minor, final_minor, discount_pct,
			sale_ends_at, lowest
		FROM (
			SELECT DISTINCT ON (l.id) l.video_game_id, l.store, l.edition, l.url, l.currency,
				q.base_minor, q.final_minor, q.discount_pct, q.sale_ends_at, q.last_seen_at,
				(SELECT count(*) > 1 AND q.final_minor <= min(p.final_minor)
					FROM price_quotes p WHERE p.listing_id = l.id) AS lowest
			FROM store_listings l
			JOIN video_games g ON g.id = l.video_game_id
			JOIN price_quotes q ON q.listing_id = l.id
			WHERE g.deleted = false AND g.status = 'wishlist'
			ORDER BY l.id, q.id DESC -- runs are appended, so the newest is the current price
		) latest
		WHERE final_minor < base_minor AND last_seen_at > $1
			AND (sale_ends_at IS NULL OR sale_ends_at > now())
		ORDER BY discount_pct DESC`, time.Now().Add(-dealMaxAge))
	if err != nil {
		return nil, fmt.Errorf("list deals: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[Deal])
}
