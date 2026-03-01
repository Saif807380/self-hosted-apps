package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/saifkazi/trove/backend/internal/dao"
	"github.com/saifkazi/trove/backend/internal/model"
)

const bookCacheTTL = 5 * time.Minute

type BookService struct {
	store *dao.BookStore
	redis *redis.Client
}

func (s *BookService) ListBooks(ctx context.Context, f dao.BookFilter) ([]model.Book, error) {
	cacheKey := fmt.Sprintf("trove:books:list:%s:%s:%d", f.Search, f.TagID, f.YearRead)

	if cached, err := s.redis.Get(ctx, cacheKey).Bytes(); err == nil {
		var books []model.Book
		if json.Unmarshal(cached, &books) == nil {
			return books, nil
		}
	}

	books, err := s.store.List(ctx, f)
	if err != nil {
		return nil, err
	}

	if data, err := json.Marshal(books); err == nil {
		s.redis.Set(ctx, cacheKey, data, bookCacheTTL) //nolint:errcheck
	}

	return books, nil
}

func (s *BookService) GetBook(ctx context.Context, id string) (*model.Book, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", err)
	}
	return s.store.Get(ctx, uid)
}

func (s *BookService) CreateBook(ctx context.Context, p dao.CreateBookParams, tagNames []string) (*model.Book, error) {
	tagIDs, err := s.upsertTags(ctx, tagNames)
	if err != nil {
		return nil, err
	}
	p.TagIDs = tagIDs

	book, err := s.store.Create(ctx, p)
	if err != nil {
		return nil, err
	}
	s.invalidateCache(ctx)
	return book, nil
}

func (s *BookService) CreateBookWithTagIDs(ctx context.Context, p dao.CreateBookParams) (*model.Book, error) {
	book, err := s.store.Create(ctx, p)
	if err != nil {
		return nil, err
	}
	s.invalidateCache(ctx)
	return book, nil
}

func (s *BookService) UpdateBook(ctx context.Context, p dao.UpdateBookParams) (*model.Book, error) {
	book, err := s.store.Update(ctx, p)
	if err != nil {
		return nil, err
	}
	s.invalidateCache(ctx)
	return book, nil
}

func (s *BookService) DeleteBook(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid id: %w", err)
	}
	if err := s.store.Delete(ctx, uid); err != nil {
		return err
	}
	s.invalidateCache(ctx)
	return nil
}

func (s *BookService) ListTags(ctx context.Context) ([]model.Tag, error) {
	return s.store.ListTags(ctx)
}

func (s *BookService) CreateTag(ctx context.Context, name string) (*model.Tag, error) {
	return s.store.UpsertTag(ctx, name)
}

func (s *BookService) ListCollections(ctx context.Context) ([]model.Collection, error) {
	return s.store.ListCollections(ctx)
}

func (s *BookService) CreateCollection(ctx context.Context, name string) (*model.Collection, error) {
	return s.store.CreateCollection(ctx, name)
}

func (s *BookService) AddBookToCollection(ctx context.Context, collectionID, bookID string) error {
	cid, err := uuid.Parse(collectionID)
	if err != nil {
		return fmt.Errorf("invalid collection_id: %w", err)
	}
	bid, err := uuid.Parse(bookID)
	if err != nil {
		return fmt.Errorf("invalid book_id: %w", err)
	}
	return s.store.AddBookToCollection(ctx, cid, bid)
}

func (s *BookService) RemoveBookFromCollection(ctx context.Context, collectionID, bookID string) error {
	cid, err := uuid.Parse(collectionID)
	if err != nil {
		return fmt.Errorf("invalid collection_id: %w", err)
	}
	bid, err := uuid.Parse(bookID)
	if err != nil {
		return fmt.Errorf("invalid book_id: %w", err)
	}
	return s.store.RemoveBookFromCollection(ctx, cid, bid)
}

func (s *BookService) upsertTags(ctx context.Context, names []string) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, len(names))
	for _, name := range names {
		tag, err := s.store.UpsertTag(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("upsert tag %q: %w", name, err)
		}
		ids = append(ids, tag.ID)
	}
	return ids, nil
}

func (s *BookService) invalidateCache(ctx context.Context) {
	// TODO: use SCAN cursor for larger keyspaces
	keys, err := s.redis.Keys(ctx, "trove:books:list:*").Result()
	if err != nil || len(keys) == 0 {
		return
	}
	s.redis.Del(ctx, keys...) //nolint:errcheck
}
