package dao

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/saifkazi/trove/backend/internal/model"
)

type WorkoutStore struct {
	db *pgxpool.Pool
}

func (s *WorkoutStore) ListTypes(ctx context.Context) ([]model.WorkoutType, error) {
	rows, err := s.db.Query(ctx, "SELECT id, name, sort_order, deleted, created_at, updated_at FROM workout_types WHERE deleted = false ORDER BY sort_order, name")
	if err != nil {
		return nil, fmt.Errorf("list workout types: %w", err)
	}
	types, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.WorkoutType])
	if err != nil {
		return nil, fmt.Errorf("scan workout types: %w", err)
	}

	for i := range types {
		types[i].Exercises, err = s.listExercises(ctx, types[i].ID)
		if err != nil {
			return nil, err
		}
	}

	if types == nil {
		types = []model.WorkoutType{}
	}
	return types, nil
}

func (s *WorkoutStore) CreateType(ctx context.Context, name string, sortOrder int16) (*model.WorkoutType, error) {
	rows, err := s.db.Query(ctx,
		"INSERT INTO workout_types (name, sort_order) VALUES ($1, $2) RETURNING id, name, sort_order, deleted, created_at, updated_at",
		name, sortOrder,
	)
	if err != nil {
		return nil, fmt.Errorf("insert workout type: %w", err)
	}
	wt, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.WorkoutType])
	if err != nil {
		return nil, fmt.Errorf("scan workout type: %w", err)
	}
	wt.Exercises = []model.Exercise{}
	return &wt, nil
}

func (s *WorkoutStore) DeleteType(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, "UPDATE workout_types SET deleted = true, updated_at = now() WHERE id = $1", id)
	return err
}

type CreateExerciseParams struct {
	WorkoutTypeID uuid.UUID
	Name          string
	SortOrder     int16
}

func (s *WorkoutStore) CreateExercise(ctx context.Context, p CreateExerciseParams) (*model.Exercise, error) {
	rows, err := s.db.Query(ctx,
		"INSERT INTO exercises (workout_type_id, name, sort_order) VALUES ($1, $2, $3) RETURNING id, workout_type_id, name, sort_order, deleted, updated_at",
		p.WorkoutTypeID, p.Name, p.SortOrder,
	)
	if err != nil {
		return nil, fmt.Errorf("insert exercise: %w", err)
	}
	ex, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Exercise])
	if err != nil {
		return nil, fmt.Errorf("scan exercise: %w", err)
	}
	return &ex, nil
}

type UpdateExerciseParams struct {
	ID        uuid.UUID
	Name      string
	SortOrder int16
}

func (s *WorkoutStore) UpdateExercise(ctx context.Context, p UpdateExerciseParams) (*model.Exercise, error) {
	rows, err := s.db.Query(ctx,
		"UPDATE exercises SET name=$2, sort_order=$3, updated_at=now() WHERE id=$1 RETURNING id, workout_type_id, name, sort_order, deleted, updated_at",
		p.ID, p.Name, p.SortOrder,
	)
	if err != nil {
		return nil, fmt.Errorf("update exercise: %w", err)
	}
	ex, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Exercise])
	if err != nil {
		return nil, fmt.Errorf("scan exercise: %w", err)
	}
	return &ex, nil
}

func (s *WorkoutStore) DeleteExercise(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, "UPDATE exercises SET deleted = true, updated_at = now() WHERE id = $1", id)
	return err
}

func (s *WorkoutStore) ListLogs(ctx context.Context, exerciseID uuid.UUID, weekNumber *int32) ([]model.WorkoutLog, error) {
	var (
		rows pgx.Rows
		err  error
	)
	if weekNumber != nil {
		rows, err = s.db.Query(ctx,
			"SELECT id, exercise_id, week_number, sets, reps, weight_kg, deleted, logged_at, updated_at FROM workout_logs WHERE exercise_id = $1 AND week_number = $2 AND deleted = false ORDER BY logged_at DESC",
			exerciseID, *weekNumber,
		)
	} else {
		rows, err = s.db.Query(ctx,
			"SELECT id, exercise_id, week_number, sets, reps, weight_kg, deleted, logged_at, updated_at FROM workout_logs WHERE exercise_id = $1 AND deleted = false ORDER BY week_number DESC, logged_at DESC",
			exerciseID,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("list logs: %w", err)
	}
	logs, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.WorkoutLog])
	if err != nil {
		return nil, fmt.Errorf("scan logs: %w", err)
	}
	if logs == nil {
		logs = []model.WorkoutLog{}
	}
	return logs, nil
}

type UpsertLogParams struct {
	ExerciseID uuid.UUID
	WeekNumber int16
	Sets       *int16
	Reps       *string
	WeightKg   *float64
}

func (s *WorkoutStore) UpsertLog(ctx context.Context, p UpsertLogParams) (*model.WorkoutLog, error) {
	rows, err := s.db.Query(ctx,
		`INSERT INTO workout_logs (exercise_id, week_number, sets, reps, weight_kg)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (exercise_id, week_number)
		 DO UPDATE SET sets = EXCLUDED.sets, reps = EXCLUDED.reps, weight_kg = EXCLUDED.weight_kg, logged_at = now(), updated_at = now()
		 RETURNING id, exercise_id, week_number, sets, reps, weight_kg, deleted, logged_at, updated_at`,
		p.ExerciseID, p.WeekNumber, p.Sets, p.Reps, p.WeightKg,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert log: %w", err)
	}
	log, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.WorkoutLog])
	if err != nil {
		return nil, fmt.Errorf("scan log: %w", err)
	}
	return &log, nil
}

func (s *WorkoutStore) DeleteLog(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, "UPDATE workout_logs SET deleted = true, updated_at = now() WHERE id = $1", id)
	return err
}

func (s *WorkoutStore) listExercises(ctx context.Context, workoutTypeID uuid.UUID) ([]model.Exercise, error) {
	rows, err := s.db.Query(ctx,
		"SELECT id, workout_type_id, name, sort_order, deleted, updated_at FROM exercises WHERE workout_type_id = $1 AND deleted = false ORDER BY sort_order, name",
		workoutTypeID,
	)
	if err != nil {
		return nil, err
	}
	exs, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.Exercise])
	if err != nil {
		return nil, err
	}
	if exs == nil {
		exs = []model.Exercise{}
	}
	return exs, nil
}
