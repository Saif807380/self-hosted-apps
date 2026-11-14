package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/saifkazi/trove/backend/internal/fit"
)

const (
	// verdictMaxAge is how long a verdict stands: patches, upgrades and new
	// editions change the answer, but not often.
	verdictMaxAge = 90 * 24 * time.Hour
	// verdictRetry is how often an incomplete verdict's missing evidence is
	// looked for again.
	verdictRetry = 6 * time.Hour
)

// VerdictTarget is a wishlisted, linked game due a device verdict.
type VerdictTarget struct {
	ID          uuid.UUID `db:"id"`
	Title       string    `db:"title"`
	ReleaseYear *int16    `db:"release_year"`
	Genres      []string  `db:"genres"`
	Platforms   []int32   `db:"platforms"`
	SteamAppID  *string   `db:"steam_appid"`
	// HasIncomplete: the game already has a verdict made without all the
	// evidence, which stands until the evidence is complete.
	HasIncomplete bool `db:"has_incomplete"`
}

// Game is the target as fit sees it.
func (t VerdictTarget) Game() fit.Game {
	g := fit.Game{Title: t.Title, Genres: t.Genres, Platforms: t.Platforms}
	if t.ReleaseYear != nil {
		g.Year = int(*t.ReleaseYear)
	}
	if t.SteamAppID != nil {
		g.SteamAppID = *t.SteamAppID
	}
	return g
}

// GamesNeedingVerdict returns wishlisted, linked games with no verdict, an
// old one, or an incomplete one due a retry; newest wishlist entries first.
// With id set, only that game is considered and an incomplete verdict is
// retried straight away.
func (s *GamingStore) GamesNeedingVerdict(ctx context.Context, id *uuid.UUID, limit int) ([]VerdictTarget, error) {
	now := time.Now()
	rows, err := s.db.Query(ctx, `
		SELECT g.id, m.title, m.release_year, m.genres, m.platforms, m.steam_appid,
			COALESCE(NOT v.complete, false) AS has_incomplete
		FROM video_games g
		JOIN game_meta m ON m.video_game_id = g.id
		LEFT JOIN device_verdicts v ON v.video_game_id = g.id
		WHERE g.deleted = false AND g.status = 'wishlist' AND m.match_method <> 'unresolved'
			AND ($1::uuid IS NULL OR g.id = $1)
			AND (v.video_game_id IS NULL OR v.created_at < $2
				OR (NOT v.complete AND ($1::uuid IS NOT NULL OR v.checked_at < $3)))
		ORDER BY g.created_at DESC
		LIMIT $4`,
		id, now.Add(-verdictMaxAge), now.Add(-verdictRetry), limit)
	if err != nil {
		return nil, fmt.Errorf("games needing verdict: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[VerdictTarget])
}

// StoredVerdict is a game's device verdict as saved.
type StoredVerdict struct {
	Verdict   fit.Verdict  `db:"verdict"`
	Sources   []fit.Source `db:"sources"`
	Model     string       `db:"model"`
	Complete  bool         `db:"complete"`
	CreatedAt time.Time    `db:"created_at"`
}

// SaveVerdict replaces a game's verdict.
func (s *GamingStore) SaveVerdict(ctx context.Context, gameID uuid.UUID, v StoredVerdict) error {
	sources := v.Sources
	if sources == nil {
		sources = []fit.Source{}
	}
	_, err := s.db.Exec(ctx, `
		INSERT INTO device_verdicts (video_game_id, verdict, sources, model, complete, created_at, checked_at)
		VALUES ($1, $2, $3, $4, $5, now(), now())
		ON CONFLICT (video_game_id) DO UPDATE SET
			verdict = EXCLUDED.verdict, sources = EXCLUDED.sources, model = EXCLUDED.model,
			complete = EXCLUDED.complete, created_at = now(), checked_at = now()`,
		gameID, v.Verdict, sources, v.Model, v.Complete)
	if err != nil {
		return fmt.Errorf("save verdict: %w", err)
	}
	return nil
}

// TouchVerdict records that an incomplete verdict's evidence was looked for
// again and is still missing.
func (s *GamingStore) TouchVerdict(ctx context.Context, gameID uuid.UUID) error {
	if _, err := s.db.Exec(ctx, "UPDATE device_verdicts SET checked_at = now() WHERE video_game_id = $1", gameID); err != nil {
		return fmt.Errorf("touch verdict: %w", err)
	}
	return nil
}

// GetVerdict returns a game's verdict, or pgx.ErrNoRows.
func (s *GamingStore) GetVerdict(ctx context.Context, gameID uuid.UUID) (*StoredVerdict, error) {
	rows, err := s.db.Query(ctx,
		"SELECT verdict, sources, model, complete, created_at FROM device_verdicts WHERE video_game_id = $1", gameID)
	if err != nil {
		return nil, fmt.Errorf("get verdict: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[StoredVerdict])
}
