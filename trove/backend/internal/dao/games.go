package dao

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/saifkazi/trove/backend/internal/model"
)

type GameStore struct {
	db *pgxpool.Pool
}

type GameFilter struct {
	Search     string
	YearPlayed int32
	Status     string // empty means every status
}

func (s *GameStore) List(ctx context.Context, f GameFilter) ([]model.VideoGame, error) {
	args := []any{}
	idx := 1

	where := " WHERE g.deleted = false"
	if f.Search != "" {
		where += fmt.Sprintf(" AND (g.title ILIKE $%d OR g.studio ILIKE $%d)", idx, idx+1)
		args = append(args, "%"+f.Search+"%", "%"+f.Search+"%")
		idx += 2
	}
	if f.YearPlayed != 0 {
		where += fmt.Sprintf(" AND EXISTS (SELECT 1 FROM game_years_played gyp WHERE gyp.game_id = g.id AND gyp.year = $%d)", idx)
		args = append(args, f.YearPlayed)
		idx++
	}
	if f.Status != "" {
		where += fmt.Sprintf(" AND g.status = $%d", idx)
		args = append(args, f.Status)
	}

	sql := "SELECT " + model.VideoGameColumns + " FROM video_games g" + where + " ORDER BY g.title"
	rows, err := s.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("list games: %w", err)
	}
	games, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.VideoGame])
	if err != nil {
		return nil, fmt.Errorf("scan games: %w", err)
	}

	for i := range games {
		games[i].YearsPlayed, err = s.getYears(ctx, games[i].ID)
		if err != nil {
			return nil, err
		}
	}

	if games == nil {
		games = []model.VideoGame{}
	}
	return games, nil
}

func (s *GameStore) Get(ctx context.Context, id uuid.UUID) (*model.VideoGame, error) {
	rows, err := s.db.Query(ctx, "SELECT "+model.VideoGameColumns+" FROM video_games WHERE id = $1 AND deleted = false", id)
	if err != nil {
		return nil, fmt.Errorf("get game: %w", err)
	}
	game, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.VideoGame])
	if err != nil {
		return nil, fmt.Errorf("get game: %w", err)
	}

	game.YearsPlayed, err = s.getYears(ctx, game.ID)
	if err != nil {
		return nil, err
	}
	return &game, nil
}

type CreateGameParams struct {
	Title          string
	Studio         *string
	Rating         *int16
	Review         *string
	CoverImage     *string
	YearsPlayed    []int16
	Status         string // required; the service defaults it
	Platform       *string
	TargetPriceINR *int32
}

func (s *GameStore) Create(ctx context.Context, p CreateGameParams) (*model.VideoGame, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	rows, err := tx.Query(ctx,
		"INSERT INTO video_games (title, studio, rating, review, cover_image, status, platform, target_price_inr) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING "+model.VideoGameColumns,
		p.Title, p.Studio, p.Rating, p.Review, p.CoverImage, p.Status, p.Platform, p.TargetPriceINR,
	)
	if err != nil {
		return nil, fmt.Errorf("insert game: %w", err)
	}
	game, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.VideoGame])
	if err != nil {
		return nil, fmt.Errorf("scan game: %w", err)
	}

	if err := replaceGameYears(ctx, tx, game.ID, p.YearsPlayed); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	game.YearsPlayed = p.YearsPlayed
	return &game, nil
}

type UpdateGameParams struct {
	ID          uuid.UUID
	Title       string
	Studio      *string
	Rating      *int16
	Review      *string
	CoverImage  *string
	YearsPlayed []int16
	// Lifecycle is nil for clients that don't send a status; the stored
	// status, platform and target price are then left alone.
	Lifecycle *GameLifecycle
}

// GameLifecycle is the set of fields that only lifecycle-aware clients send.
type GameLifecycle struct {
	Status         string
	Platform       *string
	TargetPriceINR *int32
}

func (s *GameStore) Update(ctx context.Context, p UpdateGameParams) (*model.VideoGame, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var status, platform, target any
	if p.Lifecycle != nil {
		status, platform, target = p.Lifecycle.Status, p.Lifecycle.Platform, p.Lifecycle.TargetPriceINR
	}
	// $7 IS NULL marks a client without lifecycle fields: keep what's stored.
	rows, err := tx.Query(ctx,
		`UPDATE video_games SET title=$2, studio=$3, rating=$4, review=$5, cover_image=$6,
			status = COALESCE($7::text, status),
			platform = CASE WHEN $7::text IS NULL THEN platform ELSE $8 END,
			target_price_inr = CASE WHEN $7::text IS NULL THEN target_price_inr ELSE $9 END,
			updated_at=now()
		WHERE id=$1 RETURNING `+model.VideoGameColumns,
		p.ID, p.Title, p.Studio, p.Rating, p.Review, p.CoverImage, status, platform, target,
	)
	if err != nil {
		return nil, fmt.Errorf("update game: %w", err)
	}
	game, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.VideoGame])
	if err != nil {
		return nil, fmt.Errorf("scan game: %w", err)
	}

	if err := replaceGameYears(ctx, tx, game.ID, p.YearsPlayed); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	game.YearsPlayed = p.YearsPlayed
	return &game, nil
}

func (s *GameStore) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, "UPDATE video_games SET deleted = true, updated_at = now() WHERE id = $1", id)
	return err
}

func (s *GameStore) getYears(ctx context.Context, id uuid.UUID) ([]int16, error) {
	rows, err := s.db.Query(ctx, "SELECT year FROM game_years_played WHERE game_id = $1 ORDER BY year", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var years []int16
	for rows.Next() {
		var y int16
		if err := rows.Scan(&y); err != nil {
			return nil, err
		}
		years = append(years, y)
	}
	if years == nil {
		years = []int16{}
	}
	return years, rows.Err()
}

func replaceGameYears(ctx context.Context, tx pgx.Tx, id uuid.UUID, years []int16) error {
	if _, err := tx.Exec(ctx, "DELETE FROM game_years_played WHERE game_id = $1", id); err != nil {
		return fmt.Errorf("delete years: %w", err)
	}
	for _, y := range years {
		if _, err := tx.Exec(ctx, "INSERT INTO game_years_played (game_id, year) VALUES ($1, $2)", id, y); err != nil {
			return fmt.Errorf("insert year: %w", err)
		}
	}
	return nil
}
