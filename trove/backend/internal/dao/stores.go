package dao

import "github.com/jackc/pgx/v5/pgxpool"

type Stores struct {
	DB       *pgxpool.Pool
	Books    *BookStore
	Games    *GameStore
	Travel   *TravelStore
	Workouts *WorkoutStore
	Sync     *SyncStore
	Gaming   *GamingStore
}

func NewStores(db *pgxpool.Pool) *Stores {
	return &Stores{
		DB:       db,
		Books:    &BookStore{db: db},
		Games:    &GameStore{db: db},
		Travel:   &TravelStore{db: db},
		Workouts: &WorkoutStore{db: db},
		Sync:     &SyncStore{db: db},
		Gaming:   &GamingStore{db: db},
	}
}
