package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saifkazi/trove/backend/internal/dao"
	"github.com/saifkazi/trove/backend/internal/model"
)

type WorkoutService struct {
	store *dao.WorkoutStore
}

func (s *WorkoutService) ListWorkoutTypes(ctx context.Context) ([]model.WorkoutType, error) {
	return s.store.ListTypes(ctx)
}

func (s *WorkoutService) CreateWorkoutType(ctx context.Context, name string, sortOrder int16) (*model.WorkoutType, error) {
	return s.store.CreateType(ctx, name, sortOrder)
}

func (s *WorkoutService) DeleteWorkoutType(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid id: %w", err)
	}
	return s.store.DeleteType(ctx, uid)
}

func (s *WorkoutService) CreateExercise(ctx context.Context, p dao.CreateExerciseParams) (*model.Exercise, error) {
	return s.store.CreateExercise(ctx, p)
}

func (s *WorkoutService) UpdateExercise(ctx context.Context, p dao.UpdateExerciseParams) (*model.Exercise, error) {
	return s.store.UpdateExercise(ctx, p)
}

func (s *WorkoutService) DeleteExercise(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid id: %w", err)
	}
	return s.store.DeleteExercise(ctx, uid)
}

func (s *WorkoutService) ListLogs(ctx context.Context, exerciseID string, weekNumber *int32) ([]model.WorkoutLog, error) {
	uid, err := uuid.Parse(exerciseID)
	if err != nil {
		return nil, fmt.Errorf("invalid exercise_id: %w", err)
	}
	return s.store.ListLogs(ctx, uid, weekNumber)
}

func (s *WorkoutService) UpsertLog(ctx context.Context, p dao.UpsertLogParams) (*model.WorkoutLog, error) {
	return s.store.UpsertLog(ctx, p)
}

func (s *WorkoutService) DeleteLog(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid id: %w", err)
	}
	return s.store.DeleteLog(ctx, uid)
}
