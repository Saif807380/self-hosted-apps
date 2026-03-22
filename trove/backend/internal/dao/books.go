package dao

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/saifkazi/trove/backend/internal/model"
)

type BookStore struct {
	db *pgxpool.Pool
}

type BookFilter struct {
	Search   string
	TagID    string
	YearRead int32
}

func (s *BookStore) List(ctx context.Context, f BookFilter) ([]model.Book, error) {
	args := []any{}
	idx := 1

	where := " WHERE b.deleted = false"
	if f.Search != "" {
		where += fmt.Sprintf(" AND (b.title ILIKE $%d OR b.author ILIKE $%d)", idx, idx+1)
		args = append(args, "%"+f.Search+"%", "%"+f.Search+"%")
		idx += 2
	}
	if f.YearRead != 0 {
		where += fmt.Sprintf(" AND EXISTS (SELECT 1 FROM book_years_read byr WHERE byr.book_id = b.id AND byr.year = $%d)", idx)
		args = append(args, f.YearRead)
		idx++
	}
	if f.TagID != "" {
		where += fmt.Sprintf(" AND EXISTS (SELECT 1 FROM book_tags bt WHERE bt.book_id = b.id AND bt.tag_id = $%d)", idx)
		args = append(args, f.TagID)
	}

	sql := "SELECT b.id, b.title, b.author, b.rating, b.review, b.cover_image, b.deleted, b.created_at, b.updated_at FROM books b" + where + " ORDER BY b.title"
	rows, err := s.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("list books: %w", err)
	}
	books, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.Book])
	if err != nil {
		return nil, fmt.Errorf("scan books: %w", err)
	}

	for i := range books {
		books[i].YearsRead, err = s.getYears(ctx, books[i].ID)
		if err != nil {
			return nil, err
		}
		books[i].Tags, err = s.getTags(ctx, books[i].ID)
		if err != nil {
			return nil, err
		}
	}

	if books == nil {
		books = []model.Book{}
	}
	return books, nil
}

func (s *BookStore) Get(ctx context.Context, id uuid.UUID) (*model.Book, error) {
	rows, err := s.db.Query(ctx, "SELECT id, title, author, rating, review, cover_image, deleted, created_at, updated_at FROM books WHERE id = $1 AND deleted = false", id)
	if err != nil {
		return nil, fmt.Errorf("get book: %w", err)
	}
	book, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Book])
	if err != nil {
		return nil, fmt.Errorf("get book: %w", err)
	}

	book.YearsRead, err = s.getYears(ctx, book.ID)
	if err != nil {
		return nil, err
	}
	book.Tags, err = s.getTags(ctx, book.ID)
	if err != nil {
		return nil, err
	}
	return &book, nil
}

type CreateBookParams struct {
	Title      string
	Author     string
	Rating     *int16
	Review     *string
	CoverImage *string
	YearsRead  []int16
	TagIDs     []uuid.UUID
}

func (s *BookStore) Create(ctx context.Context, p CreateBookParams) (*model.Book, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	rows, err := tx.Query(ctx,
		"INSERT INTO books (title, author, rating, review, cover_image) VALUES ($1, $2, $3, $4, $5) RETURNING id, title, author, rating, review, cover_image, deleted, created_at, updated_at",
		p.Title, p.Author, p.Rating, p.Review, p.CoverImage,
	)
	if err != nil {
		return nil, fmt.Errorf("insert book: %w", err)
	}
	book, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Book])
	if err != nil {
		return nil, fmt.Errorf("scan book: %w", err)
	}

	if err := replaceBookYears(ctx, tx, book.ID, p.YearsRead); err != nil {
		return nil, err
	}
	if err := replaceBookTags(ctx, tx, book.ID, p.TagIDs); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	book.YearsRead = p.YearsRead
	book.Tags, err = s.getTags(ctx, book.ID)
	if err != nil {
		return nil, err
	}
	return &book, nil
}

type UpdateBookParams struct {
	ID         uuid.UUID
	Title      string
	Author     string
	Rating     *int16
	Review     *string
	CoverImage *string
	YearsRead  []int16
	TagIDs     []uuid.UUID
}

func (s *BookStore) Update(ctx context.Context, p UpdateBookParams) (*model.Book, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	rows, err := tx.Query(ctx,
		"UPDATE books SET title=$2, author=$3, rating=$4, review=$5, cover_image=$6, updated_at=now() WHERE id=$1 RETURNING id, title, author, rating, review, cover_image, deleted, created_at, updated_at",
		p.ID, p.Title, p.Author, p.Rating, p.Review, p.CoverImage,
	)
	if err != nil {
		return nil, fmt.Errorf("update book: %w", err)
	}
	book, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Book])
	if err != nil {
		return nil, fmt.Errorf("scan book: %w", err)
	}

	if err := replaceBookYears(ctx, tx, book.ID, p.YearsRead); err != nil {
		return nil, err
	}
	if err := replaceBookTags(ctx, tx, book.ID, p.TagIDs); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	book.YearsRead = p.YearsRead
	book.Tags, err = s.getTags(ctx, book.ID)
	if err != nil {
		return nil, err
	}
	return &book, nil
}

func (s *BookStore) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, "UPDATE books SET deleted = true, updated_at = now() WHERE id = $1", id)
	return err
}

func (s *BookStore) ListTags(ctx context.Context) ([]model.Tag, error) {
	rows, err := s.db.Query(ctx, "SELECT id, name, deleted, updated_at FROM tags WHERE deleted = false ORDER BY name")
	if err != nil {
		return nil, err
	}
	tags, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.Tag])
	if err != nil {
		return nil, err
	}
	if tags == nil {
		tags = []model.Tag{}
	}
	return tags, nil
}

func (s *BookStore) UpsertTag(ctx context.Context, name string) (*model.Tag, error) {
	rows, err := s.db.Query(ctx,
		"INSERT INTO tags (name) VALUES ($1) ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name, deleted = false, updated_at = now() RETURNING id, name, deleted, updated_at",
		name,
	)
	if err != nil {
		return nil, err
	}
	tag, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Tag])
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

func (s *BookStore) ListCollections(ctx context.Context) ([]model.Collection, error) {
	rows, err := s.db.Query(ctx, "SELECT id, name, deleted, updated_at FROM collections WHERE deleted = false ORDER BY name")
	if err != nil {
		return nil, err
	}
	cols, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.Collection])
	if err != nil {
		return nil, err
	}
	if cols == nil {
		cols = []model.Collection{}
	}
	return cols, nil
}

func (s *BookStore) CreateCollection(ctx context.Context, name string) (*model.Collection, error) {
	rows, err := s.db.Query(ctx,
		"INSERT INTO collections (name) VALUES ($1) RETURNING id, name, deleted, updated_at",
		name,
	)
	if err != nil {
		return nil, err
	}
	col, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Collection])
	if err != nil {
		return nil, err
	}
	return &col, nil
}

func (s *BookStore) AddBookToCollection(ctx context.Context, collectionID, bookID uuid.UUID) error {
	_, err := s.db.Exec(ctx,
		"INSERT INTO collection_books (collection_id, book_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
		collectionID, bookID,
	)
	return err
}

func (s *BookStore) RemoveBookFromCollection(ctx context.Context, collectionID, bookID uuid.UUID) error {
	_, err := s.db.Exec(ctx,
		"DELETE FROM collection_books WHERE collection_id = $1 AND book_id = $2",
		collectionID, bookID,
	)
	return err
}

func (s *BookStore) getYears(ctx context.Context, id uuid.UUID) ([]int16, error) {
	rows, err := s.db.Query(ctx, "SELECT year FROM book_years_read WHERE book_id = $1 ORDER BY year", id)
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

func (s *BookStore) getTags(ctx context.Context, id uuid.UUID) ([]model.Tag, error) {
	rows, err := s.db.Query(ctx,
		"SELECT t.id, t.name, t.deleted, t.updated_at FROM tags t JOIN book_tags bt ON bt.tag_id = t.id WHERE bt.book_id = $1 AND t.deleted = false ORDER BY t.name",
		id,
	)
	if err != nil {
		return nil, err
	}
	tags, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.Tag])
	if err != nil {
		return nil, err
	}
	if tags == nil {
		tags = []model.Tag{}
	}
	return tags, nil
}

func replaceBookYears(ctx context.Context, tx pgx.Tx, id uuid.UUID, years []int16) error {
	if _, err := tx.Exec(ctx, "DELETE FROM book_years_read WHERE book_id = $1", id); err != nil {
		return fmt.Errorf("delete years: %w", err)
	}
	for _, y := range years {
		if _, err := tx.Exec(ctx, "INSERT INTO book_years_read (book_id, year) VALUES ($1, $2)", id, y); err != nil {
			return fmt.Errorf("insert year: %w", err)
		}
	}
	return nil
}

func replaceBookTags(ctx context.Context, tx pgx.Tx, id uuid.UUID, tagIDs []uuid.UUID) error {
	if _, err := tx.Exec(ctx, "DELETE FROM book_tags WHERE book_id = $1", id); err != nil {
		return fmt.Errorf("delete tags: %w", err)
	}
	for _, tid := range tagIDs {
		if _, err := tx.Exec(ctx, "INSERT INTO book_tags (book_id, tag_id) VALUES ($1, $2)", id, tid); err != nil {
			return fmt.Errorf("insert tag: %w", err)
		}
	}
	return nil
}
