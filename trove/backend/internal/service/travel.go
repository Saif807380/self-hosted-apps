package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saifkazi/trove/backend/internal/dao"
	"github.com/saifkazi/trove/backend/internal/model"
)

type TravelService struct {
	store *dao.TravelStore
}

func (s *TravelService) ListLocations(ctx context.Context, search string) ([]model.TravelLocation, error) {
	return s.store.List(ctx, search)
}

func (s *TravelService) GetLocation(ctx context.Context, id string) (*model.TravelLocation, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", err)
	}
	return s.store.Get(ctx, uid)
}

func (s *TravelService) CreateLocation(ctx context.Context, p dao.CreateLocationParams) (*model.TravelLocation, error) {
	return s.store.Create(ctx, p)
}

func (s *TravelService) UpdateLocation(ctx context.Context, p dao.UpdateLocationParams) (*model.TravelLocation, error) {
	return s.store.Update(ctx, p)
}

func (s *TravelService) DeleteLocation(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid id: %w", err)
	}
	return s.store.Delete(ctx, uid)
}

func (s *TravelService) CreateSpot(ctx context.Context, p dao.CreateSpotParams) (*model.TouristSpot, error) {
	return s.store.CreateSpot(ctx, p)
}

func (s *TravelService) UpdateSpot(ctx context.Context, p dao.UpdateSpotParams) (*model.TouristSpot, error) {
	return s.store.UpdateSpot(ctx, p)
}

func (s *TravelService) DeleteSpot(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid id: %w", err)
	}
	return s.store.DeleteSpot(ctx, uid)
}

