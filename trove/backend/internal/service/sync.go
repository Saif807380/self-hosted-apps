package service

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/saifkazi/trove/backend/internal/dao"
	"github.com/saifkazi/trove/backend/internal/model"
)

type SyncService struct {
	store *dao.SyncStore
	redis *redis.Client
}

func (s *SyncService) PullChanges(ctx context.Context, since time.Time) (*model.ChangeSet, time.Time, error) {
	cs, err := s.store.GetChangesSince(ctx, since)
	if err != nil {
		return nil, time.Time{}, err
	}
	return cs, time.Now().UTC(), nil
}

func (s *SyncService) PushChanges(ctx context.Context, cs *model.ChangeSet) (time.Time, error) {
	if err := s.store.ApplyChanges(ctx, cs); err != nil {
		return time.Time{}, err
	}
	s.invalidateCaches(ctx)
	return time.Now().UTC(), nil
}

func (s *SyncService) invalidateCaches(ctx context.Context) {
	patterns := []string{"trove:books:list:*", "trove:games:list:*"}
	for _, pattern := range patterns {
		keys, err := s.redis.Keys(ctx, pattern).Result()
		if err != nil || len(keys) == 0 {
			continue
		}
		s.redis.Del(ctx, keys...) //nolint:errcheck
	}
}
