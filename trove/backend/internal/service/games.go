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

const gameCacheTTL = 5 * time.Minute

type GameService struct {
	store *dao.GameStore
	redis *redis.Client
}

func (s *GameService) ListGames(ctx context.Context, f dao.GameFilter) ([]model.VideoGame, error) {
	cacheKey := fmt.Sprintf("trove:games:list:%s:%d", f.Search, f.YearPlayed)

	if cached, err := s.redis.Get(ctx, cacheKey).Bytes(); err == nil {
		var games []model.VideoGame
		if json.Unmarshal(cached, &games) == nil {
			return games, nil
		}
	}

	games, err := s.store.List(ctx, f)
	if err != nil {
		return nil, err
	}

	if data, err := json.Marshal(games); err == nil {
		s.redis.Set(ctx, cacheKey, data, gameCacheTTL) //nolint:errcheck
	}

	return games, nil
}

func (s *GameService) GetGame(ctx context.Context, id string) (*model.VideoGame, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", err)
	}
	return s.store.Get(ctx, uid)
}

func (s *GameService) CreateGame(ctx context.Context, p dao.CreateGameParams) (*model.VideoGame, error) {
	game, err := s.store.Create(ctx, p)
	if err != nil {
		return nil, err
	}
	s.invalidateCache(ctx)
	return game, nil
}

func (s *GameService) UpdateGame(ctx context.Context, p dao.UpdateGameParams) (*model.VideoGame, error) {
	game, err := s.store.Update(ctx, p)
	if err != nil {
		return nil, err
	}
	s.invalidateCache(ctx)
	return game, nil
}

func (s *GameService) DeleteGame(ctx context.Context, id string) error {
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

func (s *GameService) invalidateCache(ctx context.Context) {
	// TODO: use SCAN cursor for larger keyspaces
	keys, err := s.redis.Keys(ctx, "trove:games:list:*").Result()
	if err != nil || len(keys) == 0 {
		return
	}
	s.redis.Del(ctx, keys...) //nolint:errcheck
}
