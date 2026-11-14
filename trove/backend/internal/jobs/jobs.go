// Package jobs is the Redis queue the API uses to ask the worker for work
// that shouldn't wait for its next scheduled run.
package jobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// refreshQueue holds ids of games whose IGDB link, store listings and prices
// should be refreshed now (just wishlisted, relinked, or asked for).
const refreshQueue = "trove:gaming:refresh"

// EnqueueRefresh asks the worker to refresh the given games.
func EnqueueRefresh(ctx context.Context, rdb *redis.Client, ids ...uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	values := make([]any, len(ids))
	for i, id := range ids {
		values[i] = id.String()
	}
	if err := rdb.RPush(ctx, refreshQueue, values...).Err(); err != nil {
		return fmt.Errorf("enqueue refresh: %w", err)
	}
	return nil
}

// NextRefresh blocks up to wait for a game to refresh. ok is false when the
// wait ended without one.
func NextRefresh(ctx context.Context, rdb *redis.Client, wait time.Duration) (id uuid.UUID, ok bool, err error) {
	res, err := rdb.BLPop(ctx, wait, refreshQueue).Result()
	if errors.Is(err, redis.Nil) {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("next refresh: %w", err)
	}
	id, err = uuid.Parse(res[1])
	if err != nil {
		// A malformed entry can't be retried meaningfully; drop it.
		return uuid.Nil, false, nil
	}
	return id, true, nil
}
