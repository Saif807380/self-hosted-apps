package service

import (
	"github.com/redis/go-redis/v9"
	"github.com/saifkazi/trove/backend/internal/dao"
)

type Services struct {
	Books    *BookService
	Games    *GameService
	Travel   *TravelService
	Workouts *WorkoutService
	Sync     *SyncService
}

func NewServices(stores *dao.Stores, redisClient *redis.Client) *Services {
	return &Services{
		Books:    &BookService{store: stores.Books, redis: redisClient},
		Games:    &GameService{store: stores.Games, redis: redisClient},
		Travel:   &TravelService{store: stores.Travel},
		Workouts: &WorkoutService{store: stores.Workouts},
		Sync:     &SyncService{store: stores.Sync, redis: redisClient},
	}
}
