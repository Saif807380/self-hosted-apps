package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/saifkazi/trove/backend/internal/model"
)

type SyncStore struct {
	db *pgxpool.Pool
}

// GetChangesSince returns all entity and junction table rows modified after `since`.
// Includes soft-deleted records so clients learn about deletions.
func (s *SyncStore) GetChangesSince(ctx context.Context, since time.Time) (*model.ChangeSet, error) {
	cs := &model.ChangeSet{}
	var err error

	// --- Entity tables (include deleted rows) ---

	cs.Books, err = collectAll[model.Book](ctx, s.db,
		"SELECT id, title, author, rating, review, cover_image, deleted, created_at, updated_at FROM books WHERE updated_at > $1", since)
	if err != nil {
		return nil, fmt.Errorf("sync books: %w", err)
	}
	// Load years + tags for each book
	for i := range cs.Books {
		cs.Books[i].YearsRead, err = collectYears(ctx, s.db, "SELECT year FROM book_years_read WHERE book_id = $1 ORDER BY year", cs.Books[i].ID)
		if err != nil {
			return nil, fmt.Errorf("sync book years: %w", err)
		}
		cs.Books[i].Tags, err = collectAll[model.Tag](ctx, s.db,
			"SELECT t.id, t.name, t.deleted, t.updated_at FROM tags t JOIN book_tags bt ON bt.tag_id = t.id WHERE bt.book_id = $1", cs.Books[i].ID)
		if err != nil {
			return nil, fmt.Errorf("sync book tags: %w", err)
		}
	}

	cs.Tags, err = collectAll[model.Tag](ctx, s.db,
		"SELECT id, name, deleted, updated_at FROM tags WHERE updated_at > $1", since)
	if err != nil {
		return nil, fmt.Errorf("sync tags: %w", err)
	}

	cs.Collections, err = collectAll[model.Collection](ctx, s.db,
		"SELECT id, name, deleted, updated_at FROM collections WHERE updated_at > $1", since)
	if err != nil {
		return nil, fmt.Errorf("sync collections: %w", err)
	}

	cs.VideoGames, err = collectAll[model.VideoGame](ctx, s.db,
		"SELECT id, title, studio, rating, review, cover_image, deleted, created_at, updated_at FROM video_games WHERE updated_at > $1", since)
	if err != nil {
		return nil, fmt.Errorf("sync video_games: %w", err)
	}
	for i := range cs.VideoGames {
		cs.VideoGames[i].YearsPlayed, err = collectYears(ctx, s.db, "SELECT year FROM game_years_played WHERE game_id = $1 ORDER BY year", cs.VideoGames[i].ID)
		if err != nil {
			return nil, fmt.Errorf("sync game years: %w", err)
		}
	}

	cs.TravelLocations, err = collectAll[model.TravelLocation](ctx, s.db,
		"SELECT id, city, country, visited_from, visited_to, photo_collection_url, deleted, created_at, updated_at FROM travel_locations WHERE updated_at > $1", since)
	if err != nil {
		return nil, fmt.Errorf("sync travel_locations: %w", err)
	}

	cs.TouristSpots, err = collectAll[model.TouristSpot](ctx, s.db,
		"SELECT id, location_id, name, description, deleted, updated_at FROM tourist_spots WHERE updated_at > $1", since)
	if err != nil {
		return nil, fmt.Errorf("sync tourist_spots: %w", err)
	}

	cs.WorkoutTypes, err = collectAll[model.WorkoutType](ctx, s.db,
		"SELECT id, name, sort_order, deleted, created_at, updated_at FROM workout_types WHERE updated_at > $1", since)
	if err != nil {
		return nil, fmt.Errorf("sync workout_types: %w", err)
	}

	cs.Exercises, err = collectAll[model.Exercise](ctx, s.db,
		"SELECT id, workout_type_id, name, sort_order, deleted, updated_at FROM exercises WHERE updated_at > $1", since)
	if err != nil {
		return nil, fmt.Errorf("sync exercises: %w", err)
	}

	cs.WorkoutLogs, err = collectAll[model.WorkoutLog](ctx, s.db,
		"SELECT id, exercise_id, week_number, sets, reps, weight_kg, deleted, logged_at, updated_at FROM workout_logs WHERE updated_at > $1", since)
	if err != nil {
		return nil, fmt.Errorf("sync workout_logs: %w", err)
	}

	// --- Junction tables (for changed parent entities) ---

	cs.BookYearsRead, err = collectJunction[model.BookYearRead](ctx, s.db, since,
		"SELECT byr.book_id, byr.year FROM book_years_read byr JOIN books b ON b.id = byr.book_id WHERE b.updated_at > $1")
	if err != nil {
		return nil, fmt.Errorf("sync book_years_read: %w", err)
	}

	cs.BookTags, err = collectJunction[model.BookTagEntry](ctx, s.db, since,
		"SELECT bt.book_id, bt.tag_id FROM book_tags bt JOIN books b ON b.id = bt.book_id WHERE b.updated_at > $1")
	if err != nil {
		return nil, fmt.Errorf("sync book_tags: %w", err)
	}

	cs.CollectionBooks, err = collectJunction[model.CollectionBookEntry](ctx, s.db, since,
		"SELECT cb.collection_id, cb.book_id FROM collection_books cb JOIN collections c ON c.id = cb.collection_id WHERE c.updated_at > $1")
	if err != nil {
		return nil, fmt.Errorf("sync collection_books: %w", err)
	}

	cs.GameYearsPlayed, err = collectJunction[model.GameYearPlayed](ctx, s.db, since,
		"SELECT gyp.game_id, gyp.year FROM game_years_played gyp JOIN video_games g ON g.id = gyp.game_id WHERE g.updated_at > $1")
	if err != nil {
		return nil, fmt.Errorf("sync game_years_played: %w", err)
	}

	return cs, nil
}

// ApplyChanges upserts all entities in a single transaction using topological ordering.
// LWW: only updates if incoming updated_at > existing updated_at.
func (s *SyncStore) ApplyChanges(ctx context.Context, cs *model.ChangeSet) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, "SET CONSTRAINTS ALL DEFERRED"); err != nil {
		return fmt.Errorf("defer constraints: %w", err)
	}

	// Layer 1: no FK deps
	for _, t := range cs.Tags {
		if err := upsertTag(ctx, tx, t); err != nil {
			return err
		}
	}
	for _, c := range cs.Collections {
		if err := upsertCollection(ctx, tx, c); err != nil {
			return err
		}
	}
	for _, wt := range cs.WorkoutTypes {
		if err := upsertWorkoutType(ctx, tx, wt); err != nil {
			return err
		}
	}

	// Layer 2: FK to layer 1 or standalone
	for _, b := range cs.Books {
		if err := upsertBook(ctx, tx, b); err != nil {
			return err
		}
	}
	for _, g := range cs.VideoGames {
		if err := upsertVideoGame(ctx, tx, g); err != nil {
			return err
		}
	}
	for _, l := range cs.TravelLocations {
		if err := upsertTravelLocation(ctx, tx, l); err != nil {
			return err
		}
	}

	// Layer 3: FK to layer 2
	for _, e := range cs.Exercises {
		if err := upsertExercise(ctx, tx, e); err != nil {
			return err
		}
	}
	for _, s := range cs.TouristSpots {
		if err := upsertTouristSpot(ctx, tx, s); err != nil {
			return err
		}
	}

	// Layer 4: FK to layer 3
	for _, wl := range cs.WorkoutLogs {
		if err := upsertWorkoutLog(ctx, tx, wl); err != nil {
			return err
		}
	}

	// Junction tables: delete-all-for-parent + re-insert
	if err := replaceJunctionBookYears(ctx, tx, cs.BookYearsRead); err != nil {
		return err
	}
	if err := replaceJunctionBookTags(ctx, tx, cs.BookTags); err != nil {
		return err
	}
	if err := replaceJunctionCollectionBooks(ctx, tx, cs.CollectionBooks); err != nil {
		return err
	}
	if err := replaceJunctionGameYears(ctx, tx, cs.GameYearsPlayed); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// --- Upsert helpers (LWW: only update if incoming is newer) ---

func upsertTag(ctx context.Context, tx pgx.Tx, t model.Tag) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO tags (id, name, deleted, updated_at) VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, deleted = EXCLUDED.deleted, updated_at = EXCLUDED.updated_at
		WHERE EXCLUDED.updated_at > tags.updated_at`,
		t.ID, t.Name, t.Deleted, t.UpdatedAt)
	return err
}

func upsertCollection(ctx context.Context, tx pgx.Tx, c model.Collection) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO collections (id, name, deleted, updated_at) VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, deleted = EXCLUDED.deleted, updated_at = EXCLUDED.updated_at
		WHERE EXCLUDED.updated_at > collections.updated_at`,
		c.ID, c.Name, c.Deleted, c.UpdatedAt)
	return err
}

func upsertWorkoutType(ctx context.Context, tx pgx.Tx, wt model.WorkoutType) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO workout_types (id, name, sort_order, deleted, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, sort_order = EXCLUDED.sort_order, deleted = EXCLUDED.deleted, updated_at = EXCLUDED.updated_at
		WHERE EXCLUDED.updated_at > workout_types.updated_at`,
		wt.ID, wt.Name, wt.SortOrder, wt.Deleted, wt.CreatedAt, wt.UpdatedAt)
	return err
}

func upsertBook(ctx context.Context, tx pgx.Tx, b model.Book) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO books (id, title, author, rating, review, cover_image, deleted, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET title = EXCLUDED.title, author = EXCLUDED.author, rating = EXCLUDED.rating,
			review = EXCLUDED.review, cover_image = EXCLUDED.cover_image, deleted = EXCLUDED.deleted, updated_at = EXCLUDED.updated_at
		WHERE EXCLUDED.updated_at > books.updated_at`,
		b.ID, b.Title, b.Author, b.Rating, b.Review, b.CoverImage, b.Deleted, b.CreatedAt, b.UpdatedAt)
	return err
}

func upsertVideoGame(ctx context.Context, tx pgx.Tx, g model.VideoGame) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO video_games (id, title, studio, rating, review, cover_image, deleted, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET title = EXCLUDED.title, studio = EXCLUDED.studio, rating = EXCLUDED.rating,
			review = EXCLUDED.review, cover_image = EXCLUDED.cover_image, deleted = EXCLUDED.deleted, updated_at = EXCLUDED.updated_at
		WHERE EXCLUDED.updated_at > video_games.updated_at`,
		g.ID, g.Title, g.Studio, g.Rating, g.Review, g.CoverImage, g.Deleted, g.CreatedAt, g.UpdatedAt)
	return err
}

func upsertTravelLocation(ctx context.Context, tx pgx.Tx, l model.TravelLocation) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO travel_locations (id, city, country, visited_from, visited_to, photo_collection_url, deleted, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET city = EXCLUDED.city, country = EXCLUDED.country, visited_from = EXCLUDED.visited_from,
			visited_to = EXCLUDED.visited_to, photo_collection_url = EXCLUDED.photo_collection_url, deleted = EXCLUDED.deleted, updated_at = EXCLUDED.updated_at
		WHERE EXCLUDED.updated_at > travel_locations.updated_at`,
		l.ID, l.City, l.Country, l.VisitedFrom, l.VisitedTo, l.PhotoCollectionURL, l.Deleted, l.CreatedAt, l.UpdatedAt)
	return err
}

func upsertExercise(ctx context.Context, tx pgx.Tx, e model.Exercise) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO exercises (id, workout_type_id, name, sort_order, deleted, updated_at) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET workout_type_id = EXCLUDED.workout_type_id, name = EXCLUDED.name,
			sort_order = EXCLUDED.sort_order, deleted = EXCLUDED.deleted, updated_at = EXCLUDED.updated_at
		WHERE EXCLUDED.updated_at > exercises.updated_at`,
		e.ID, e.WorkoutTypeID, e.Name, e.SortOrder, e.Deleted, e.UpdatedAt)
	return err
}

func upsertTouristSpot(ctx context.Context, tx pgx.Tx, s model.TouristSpot) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO tourist_spots (id, location_id, name, description, deleted, updated_at) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET location_id = EXCLUDED.location_id, name = EXCLUDED.name,
			description = EXCLUDED.description, deleted = EXCLUDED.deleted, updated_at = EXCLUDED.updated_at
		WHERE EXCLUDED.updated_at > tourist_spots.updated_at`,
		s.ID, s.LocationID, s.Name, s.Description, s.Deleted, s.UpdatedAt)
	return err
}

func upsertWorkoutLog(ctx context.Context, tx pgx.Tx, wl model.WorkoutLog) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO workout_logs (id, exercise_id, week_number, sets, reps, weight_kg, deleted, logged_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET exercise_id = EXCLUDED.exercise_id, week_number = EXCLUDED.week_number,
			sets = EXCLUDED.sets, reps = EXCLUDED.reps, weight_kg = EXCLUDED.weight_kg,
			deleted = EXCLUDED.deleted, logged_at = EXCLUDED.logged_at, updated_at = EXCLUDED.updated_at
		WHERE EXCLUDED.updated_at > workout_logs.updated_at`,
		wl.ID, wl.ExerciseID, wl.WeekNumber, wl.Sets, wl.Reps, wl.WeightKg, wl.Deleted, wl.LoggedAt, wl.UpdatedAt)
	return err
}

// --- Junction table replacements (delete-all-for-parent + re-insert) ---

func replaceJunctionBookYears(ctx context.Context, tx pgx.Tx, entries []model.BookYearRead) error {
	seen := map[uuid.UUID]bool{}
	for _, e := range entries {
		if !seen[e.BookID] {
			if _, err := tx.Exec(ctx, "DELETE FROM book_years_read WHERE book_id = $1", e.BookID); err != nil {
				return fmt.Errorf("delete book_years_read: %w", err)
			}
			seen[e.BookID] = true
		}
		if _, err := tx.Exec(ctx, "INSERT INTO book_years_read (book_id, year) VALUES ($1, $2) ON CONFLICT DO NOTHING", e.BookID, e.Year); err != nil {
			return fmt.Errorf("insert book_years_read: %w", err)
		}
	}
	return nil
}

func replaceJunctionBookTags(ctx context.Context, tx pgx.Tx, entries []model.BookTagEntry) error {
	seen := map[uuid.UUID]bool{}
	for _, e := range entries {
		if !seen[e.BookID] {
			if _, err := tx.Exec(ctx, "DELETE FROM book_tags WHERE book_id = $1", e.BookID); err != nil {
				return fmt.Errorf("delete book_tags: %w", err)
			}
			seen[e.BookID] = true
		}
		if _, err := tx.Exec(ctx, "INSERT INTO book_tags (book_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", e.BookID, e.TagID); err != nil {
			return fmt.Errorf("insert book_tags: %w", err)
		}
	}
	return nil
}

func replaceJunctionCollectionBooks(ctx context.Context, tx pgx.Tx, entries []model.CollectionBookEntry) error {
	seen := map[uuid.UUID]bool{}
	for _, e := range entries {
		if !seen[e.CollectionID] {
			if _, err := tx.Exec(ctx, "DELETE FROM collection_books WHERE collection_id = $1", e.CollectionID); err != nil {
				return fmt.Errorf("delete collection_books: %w", err)
			}
			seen[e.CollectionID] = true
		}
		if _, err := tx.Exec(ctx, "INSERT INTO collection_books (collection_id, book_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", e.CollectionID, e.BookID); err != nil {
			return fmt.Errorf("insert collection_books: %w", err)
		}
	}
	return nil
}

func replaceJunctionGameYears(ctx context.Context, tx pgx.Tx, entries []model.GameYearPlayed) error {
	seen := map[uuid.UUID]bool{}
	for _, e := range entries {
		if !seen[e.GameID] {
			if _, err := tx.Exec(ctx, "DELETE FROM game_years_played WHERE game_id = $1", e.GameID); err != nil {
				return fmt.Errorf("delete game_years_played: %w", err)
			}
			seen[e.GameID] = true
		}
		if _, err := tx.Exec(ctx, "INSERT INTO game_years_played (game_id, year) VALUES ($1, $2) ON CONFLICT DO NOTHING", e.GameID, e.Year); err != nil {
			return fmt.Errorf("insert game_years_played: %w", err)
		}
	}
	return nil
}

// --- Generic query helpers ---

func collectAll[T any](ctx context.Context, db *pgxpool.Pool, sql string, args ...any) ([]T, error) {
	rows, err := db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[T])
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []T{}
	}
	return items, nil
}

func collectJunction[T any](ctx context.Context, db *pgxpool.Pool, since time.Time, sql string) ([]T, error) {
	rows, err := db.Query(ctx, sql, since)
	if err != nil {
		return nil, err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[T])
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []T{}
	}
	return items, nil
}

func collectYears(ctx context.Context, db *pgxpool.Pool, sql string, id uuid.UUID) ([]int16, error) {
	rows, err := db.Query(ctx, sql, id)
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
