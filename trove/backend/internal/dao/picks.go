package dao

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/saifkazi/trove/backend/internal/picks"
)

// Pick states.
const (
	PickNew        = "new"
	PickWishlisted = "wishlisted"
	PickOwned      = "owned"
	PickDismissed  = "dismissed"
)

// recentPickWeeks is how long a pick you ignored stays out of later picks.
const recentPickWeeks = 8

// Pick is one of a week's picks as stored.
type Pick struct {
	ID          uuid.UUID  `db:"id"`
	Week        time.Time  `db:"week"`
	Rank        int16      `db:"rank"`
	IGDBID      int64      `db:"igdb_id"`
	Title       string     `db:"title"`
	Year        *int16     `db:"year"`
	CoverURL    *string    `db:"cover_url"`
	Pitch       string     `db:"pitch"`
	Tags        []string   `db:"tags"`
	Store       string     `db:"store"`
	StoreRef    string     `db:"store_ref"`
	Edition     string     `db:"edition"`
	URL         string     `db:"url"`
	Currency    string     `db:"currency"`
	BaseMinor   int64      `db:"base_minor"`
	FinalMinor  int64      `db:"final_minor"`
	DiscountPct int16      `db:"discount_pct"`
	SaleEndsAt  *time.Time `db:"sale_ends_at"`
	State       string     `db:"state"`
	VideoGameID *uuid.UUID `db:"video_game_id"`
}

const pickColumns = `id, week, rank, igdb_id, title, year, cover_url, pitch, tags, store, store_ref, edition,
	url, currency, base_minor, final_minor, discount_pct, sale_ends_at, state, video_game_id`

// Library returns your linked games as the taste profile sees them.
func (s *GamingStore) Library(ctx context.Context) ([]picks.LibraryGame, error) {
	rows, err := s.db.Query(ctx, `
		SELECT g.title, g.status, g.rating, m.igdb_id, m.genres, m.themes, m.keywords, m.similar_ids,
			COALESCE((SELECT max(y.year) FROM game_years_played y WHERE y.game_id = g.id), 0) AS last_played
		FROM video_games g JOIN game_meta m ON m.video_game_id = g.id
		WHERE g.deleted = false AND m.match_method <> 'unresolved' AND m.igdb_id IS NOT NULL`)
	if err != nil {
		return nil, fmt.Errorf("library: %w", err)
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (picks.LibraryGame, error) {
		var g picks.LibraryGame
		var lastPlayed int16
		err := row.Scan(&g.Title, &g.Status, &g.Rating, &g.IGDBID, &g.Genres, &g.Themes, &g.Keywords, &g.SimilarIDs, &lastPlayed)
		g.LastPlayed = int(lastPlayed)
		return g, err
	})
}

// DismissedPickTags returns the tags of every pick you turned down.
func (s *GamingStore) DismissedPickTags(ctx context.Context) ([][]string, error) {
	rows, err := s.db.Query(ctx, "SELECT tags FROM picks WHERE state = 'dismissed'")
	if err != nil {
		return nil, fmt.Errorf("dismissed pick tags: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowTo[[]string])
}

// ExcludedFromPicks returns IGDB ids not to pick: games in your library,
// picks you acted on, and picks from the last few weeks.
func (s *GamingStore) ExcludedFromPicks(ctx context.Context) (map[int64]bool, error) {
	rows, err := s.db.Query(ctx, `
		SELECT m.igdb_id FROM game_meta m JOIN video_games g ON g.id = m.video_game_id
		WHERE g.deleted = false AND m.igdb_id IS NOT NULL
		UNION
		SELECT igdb_id FROM picks WHERE state <> 'new' OR week > $1`,
		time.Now().AddDate(0, 0, -7*recentPickWeeks))
	if err != nil {
		return nil, fmt.Errorf("excluded from picks: %w", err)
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return nil, err
	}
	out := make(map[int64]bool, len(ids))
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}

// LastPickRun returns the week of the latest picks run, or the zero time.
func (s *GamingStore) LastPickRun(ctx context.Context) (time.Time, error) {
	var week *time.Time
	if err := s.db.QueryRow(ctx, "SELECT max(week) FROM pick_runs").Scan(&week); err != nil {
		return time.Time{}, fmt.Errorf("last pick run: %w", err)
	}
	if week == nil {
		return time.Time{}, nil
	}
	return *week, nil
}

// SavePicks stores a week's picks and records the run.
func (s *GamingStore) SavePicks(ctx context.Context, week time.Time, ps []Pick) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	for _, p := range ps {
		_, err := tx.Exec(ctx, `
			INSERT INTO picks (week, rank, igdb_id, title, year, cover_url, pitch, tags, store, store_ref, edition,
				url, currency, base_minor, final_minor, discount_pct, sale_ends_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
			ON CONFLICT (week, igdb_id) DO NOTHING`,
			week, p.Rank, p.IGDBID, p.Title, p.Year, p.CoverURL, p.Pitch, nonNil(p.Tags), p.Store, p.StoreRef,
			p.Edition, p.URL, p.Currency, p.BaseMinor, p.FinalMinor, p.DiscountPct, p.SaleEndsAt)
		if err != nil {
			return fmt.Errorf("save pick: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO pick_runs (week, picks) VALUES ($1, $2)
		ON CONFLICT (week) DO UPDATE SET picks = EXCLUDED.picks, ran_at = now()`, week, len(ps)); err != nil {
		return fmt.Errorf("record pick run: %w", err)
	}
	return tx.Commit(ctx)
}

// CurrentPicks returns the latest week's picks you haven't acted on, whose
// sale hasn't ended, best first.
func (s *GamingStore) CurrentPicks(ctx context.Context) ([]Pick, error) {
	rows, err := s.db.Query(ctx, `SELECT `+pickColumns+` FROM picks
		WHERE week = (SELECT max(week) FROM pick_runs) AND state = 'new'
			AND (sale_ends_at IS NULL OR sale_ends_at > now())
		ORDER BY rank`)
	if err != nil {
		return nil, fmt.Errorf("current picks: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[Pick])
}

// PicksForWeek returns every pick made for a week, in rank order.
func (s *GamingStore) PicksForWeek(ctx context.Context, week time.Time) ([]Pick, error) {
	rows, err := s.db.Query(ctx, `SELECT `+pickColumns+` FROM picks WHERE week = $1 ORDER BY rank`, week)
	if err != nil {
		return nil, fmt.Errorf("picks for week: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[Pick])
}

// GetPick returns one pick, or pgx.ErrNoRows.
func (s *GamingStore) GetPick(ctx context.Context, id uuid.UUID) (*Pick, error) {
	rows, err := s.db.Query(ctx, `SELECT `+pickColumns+` FROM picks WHERE id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("get pick: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[Pick])
}

// ErrPickTaken means the pick was already acted on.
var ErrPickTaken = errors.New("pick already acted on")

// ClaimPick records what you did with a new pick, once: a pick that's no
// longer new returns ErrPickTaken, so a double tap can't add a game twice.
func (s *GamingStore) ClaimPick(ctx context.Context, id uuid.UUID, state string) error {
	tag, err := s.db.Exec(ctx, "UPDATE picks SET state = $2 WHERE id = $1 AND state = 'new'", id, state)
	if err != nil {
		return fmt.Errorf("claim pick: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrPickTaken
	}
	return nil
}

// UnclaimPick makes a pick new again, when adding its game failed.
func (s *GamingStore) UnclaimPick(ctx context.Context, id uuid.UUID) error {
	if _, err := s.db.Exec(ctx, "UPDATE picks SET state = 'new', video_game_id = NULL WHERE id = $1", id); err != nil {
		return fmt.Errorf("unclaim pick: %w", err)
	}
	return nil
}

// SetPickGame records the game made from a pick.
func (s *GamingStore) SetPickGame(ctx context.Context, id, gameID uuid.UUID) error {
	if _, err := s.db.Exec(ctx, "UPDATE picks SET video_game_id = $2 WHERE id = $1", id, gameID); err != nil {
		return fmt.Errorf("set pick game: %w", err)
	}
	return nil
}
