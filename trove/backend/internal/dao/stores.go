package dao

import "github.com/jackc/pgx/v5/pgxpool"

type Stores struct {
	DB *pgxpool.Pool
}

func NewStores(db *pgxpool.Pool) *Stores {
	return &Stores{DB: db}
}
