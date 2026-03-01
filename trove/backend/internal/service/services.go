package service

import (
	"github.com/redis/go-redis/v9"
	"github.com/saifkazi/trove/backend/internal/dao"
)

type Services struct {
	Stores *dao.Stores
	Redis  *redis.Client
}

func NewServices(stores *dao.Stores, redis *redis.Client) *Services {
	return &Services{Stores: stores, Redis: redis}
}
