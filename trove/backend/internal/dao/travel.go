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

type TravelStore struct {
	db *pgxpool.Pool
}

func (s *TravelStore) List(ctx context.Context, search string) ([]model.TravelLocation, error) {
	args := []any{}
	where := " WHERE l.deleted = false"
	if search != "" {
		where += " AND (l.city ILIKE $1 OR l.country ILIKE $1)"
		args = append(args, "%"+search+"%")
	}

	sql := "SELECT l.id, l.city, l.country, l.visited_from, l.visited_to, l.photo_collection_url, l.deleted, l.created_at, l.updated_at FROM travel_locations l" + where + " ORDER BY l.country, l.city"
	rows, err := s.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("list locations: %w", err)
	}
	locs, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.TravelLocation])
	if err != nil {
		return nil, fmt.Errorf("scan locations: %w", err)
	}

	for i := range locs {
		locs[i].TouristSpots, err = s.listSpots(ctx, locs[i].ID)
		if err != nil {
			return nil, err
		}
	}

	if locs == nil {
		locs = []model.TravelLocation{}
	}
	return locs, nil
}

func (s *TravelStore) Get(ctx context.Context, id uuid.UUID) (*model.TravelLocation, error) {
	rows, err := s.db.Query(ctx,
		"SELECT id, city, country, visited_from, visited_to, photo_collection_url, deleted, created_at, updated_at FROM travel_locations WHERE id = $1 AND deleted = false",
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("get location: %w", err)
	}
	loc, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.TravelLocation])
	if err != nil {
		return nil, fmt.Errorf("get location: %w", err)
	}

	loc.TouristSpots, err = s.listSpots(ctx, loc.ID)
	if err != nil {
		return nil, err
	}
	return &loc, nil
}

type CreateLocationParams struct {
	City               string
	Country            string
	VisitedFrom        *time.Time
	VisitedTo          *time.Time
	PhotoCollectionURL *string
}

func (s *TravelStore) Create(ctx context.Context, p CreateLocationParams) (*model.TravelLocation, error) {
	rows, err := s.db.Query(ctx,
		"INSERT INTO travel_locations (city, country, visited_from, visited_to, photo_collection_url) VALUES ($1, $2, $3, $4, $5) RETURNING id, city, country, visited_from, visited_to, photo_collection_url, deleted, created_at, updated_at",
		p.City, p.Country, p.VisitedFrom, p.VisitedTo, p.PhotoCollectionURL,
	)
	if err != nil {
		return nil, fmt.Errorf("insert location: %w", err)
	}
	loc, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.TravelLocation])
	if err != nil {
		return nil, fmt.Errorf("scan location: %w", err)
	}
	loc.TouristSpots = []model.TouristSpot{}
	return &loc, nil
}

type UpdateLocationParams struct {
	ID                 uuid.UUID
	City               string
	Country            string
	VisitedFrom        *time.Time
	VisitedTo          *time.Time
	PhotoCollectionURL *string
}

func (s *TravelStore) Update(ctx context.Context, p UpdateLocationParams) (*model.TravelLocation, error) {
	rows, err := s.db.Query(ctx,
		"UPDATE travel_locations SET city=$2, country=$3, visited_from=$4, visited_to=$5, photo_collection_url=$6, updated_at=now() WHERE id=$1 RETURNING id, city, country, visited_from, visited_to, photo_collection_url, deleted, created_at, updated_at",
		p.ID, p.City, p.Country, p.VisitedFrom, p.VisitedTo, p.PhotoCollectionURL,
	)
	if err != nil {
		return nil, fmt.Errorf("update location: %w", err)
	}
	loc, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.TravelLocation])
	if err != nil {
		return nil, fmt.Errorf("scan location: %w", err)
	}

	loc.TouristSpots, err = s.listSpots(ctx, loc.ID)
	if err != nil {
		return nil, err
	}
	return &loc, nil
}

func (s *TravelStore) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, "UPDATE travel_locations SET deleted = true, updated_at = now() WHERE id = $1", id)
	return err
}

type CreateSpotParams struct {
	LocationID  uuid.UUID
	Name        string
	Description *string
}

func (s *TravelStore) CreateSpot(ctx context.Context, p CreateSpotParams) (*model.TouristSpot, error) {
	rows, err := s.db.Query(ctx,
		"INSERT INTO tourist_spots (location_id, name, description) VALUES ($1, $2, $3) RETURNING id, location_id, name, description, deleted, updated_at",
		p.LocationID, p.Name, p.Description,
	)
	if err != nil {
		return nil, fmt.Errorf("insert spot: %w", err)
	}
	spot, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.TouristSpot])
	if err != nil {
		return nil, fmt.Errorf("scan spot: %w", err)
	}
	return &spot, nil
}

type UpdateSpotParams struct {
	ID          uuid.UUID
	Name        string
	Description *string
}

func (s *TravelStore) UpdateSpot(ctx context.Context, p UpdateSpotParams) (*model.TouristSpot, error) {
	rows, err := s.db.Query(ctx,
		"UPDATE tourist_spots SET name=$2, description=$3, updated_at=now() WHERE id=$1 RETURNING id, location_id, name, description, deleted, updated_at",
		p.ID, p.Name, p.Description,
	)
	if err != nil {
		return nil, fmt.Errorf("update spot: %w", err)
	}
	spot, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.TouristSpot])
	if err != nil {
		return nil, fmt.Errorf("scan spot: %w", err)
	}
	return &spot, nil
}

func (s *TravelStore) DeleteSpot(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, "UPDATE tourist_spots SET deleted = true, updated_at = now() WHERE id = $1", id)
	return err
}

func (s *TravelStore) listSpots(ctx context.Context, locationID uuid.UUID) ([]model.TouristSpot, error) {
	rows, err := s.db.Query(ctx,
		"SELECT id, location_id, name, description, deleted, updated_at FROM tourist_spots WHERE location_id = $1 AND deleted = false ORDER BY name",
		locationID,
	)
	if err != nil {
		return nil, err
	}
	spots, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.TouristSpot])
	if err != nil {
		return nil, err
	}
	if spots == nil {
		spots = []model.TouristSpot{}
	}
	return spots, nil
}
