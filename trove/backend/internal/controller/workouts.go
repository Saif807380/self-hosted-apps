package controller

import (
	"context"
	"time"

	"connectrpc.com/connect"
	workoutsv1 "github.com/saifkazi/trove/backend/gen/trove/v1/workouts"
	"github.com/saifkazi/trove/backend/gen/trove/v1/workouts/workoutsv1connect"
	"github.com/saifkazi/trove/backend/internal/dao"
	"github.com/saifkazi/trove/backend/internal/model"
	"github.com/saifkazi/trove/backend/internal/service"
)

type workoutHandler struct {
	svc *service.WorkoutService
}

var _ workoutsv1connect.WorkoutServiceHandler = (*workoutHandler)(nil)

func (h *workoutHandler) ListWorkoutTypes(
	ctx context.Context,
	req *connect.Request[workoutsv1.ListWorkoutTypesRequest],
) (*connect.Response[workoutsv1.ListWorkoutTypesResponse], error) {
	types, err := h.svc.ListWorkoutTypes(ctx)
	if err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&workoutsv1.ListWorkoutTypesResponse{
		WorkoutTypes: mapWorkoutTypes(types),
	}), nil
}

func (h *workoutHandler) CreateWorkoutType(
	ctx context.Context,
	req *connect.Request[workoutsv1.CreateWorkoutTypeRequest],
) (*connect.Response[workoutsv1.CreateWorkoutTypeResponse], error) {
	msg := req.Msg
	if msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}

	wt, err := h.svc.CreateWorkoutType(ctx, msg.Name, int16(msg.SortOrder))
	if err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&workoutsv1.CreateWorkoutTypeResponse{
		WorkoutType: mapWorkoutType(wt),
	}), nil
}

func (h *workoutHandler) DeleteWorkoutType(
	ctx context.Context,
	req *connect.Request[workoutsv1.DeleteWorkoutTypeRequest],
) (*connect.Response[workoutsv1.DeleteWorkoutTypeResponse], error) {
	if err := h.svc.DeleteWorkoutType(ctx, req.Msg.Id); err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&workoutsv1.DeleteWorkoutTypeResponse{}), nil
}

func (h *workoutHandler) CreateExercise(
	ctx context.Context,
	req *connect.Request[workoutsv1.CreateExerciseRequest],
) (*connect.Response[workoutsv1.CreateExerciseResponse], error) {
	msg := req.Msg
	if msg.WorkoutTypeId == "" || msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}

	uid, err := parseUUID(msg.WorkoutTypeId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	ex, err := h.svc.CreateExercise(ctx, dao.CreateExerciseParams{
		WorkoutTypeID: uid,
		Name:          msg.Name,
		SortOrder:     int16(msg.SortOrder),
	})
	if err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&workoutsv1.CreateExerciseResponse{
		Exercise: mapExercise(ex),
	}), nil
}

func (h *workoutHandler) UpdateExercise(
	ctx context.Context,
	req *connect.Request[workoutsv1.UpdateExerciseRequest],
) (*connect.Response[workoutsv1.UpdateExerciseResponse], error) {
	msg := req.Msg
	if msg.Id == "" || msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}

	uid, err := parseUUID(msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	ex, err := h.svc.UpdateExercise(ctx, dao.UpdateExerciseParams{
		ID:        uid,
		Name:      msg.Name,
		SortOrder: int16(msg.SortOrder),
	})
	if err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&workoutsv1.UpdateExerciseResponse{
		Exercise: mapExercise(ex),
	}), nil
}

func (h *workoutHandler) DeleteExercise(
	ctx context.Context,
	req *connect.Request[workoutsv1.DeleteExerciseRequest],
) (*connect.Response[workoutsv1.DeleteExerciseResponse], error) {
	if err := h.svc.DeleteExercise(ctx, req.Msg.Id); err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&workoutsv1.DeleteExerciseResponse{}), nil
}

func (h *workoutHandler) ListLogs(
	ctx context.Context,
	req *connect.Request[workoutsv1.ListLogsRequest],
) (*connect.Response[workoutsv1.ListLogsResponse], error) {
	msg := req.Msg
	if msg.ExerciseId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}

	logs, err := h.svc.ListLogs(ctx, msg.ExerciseId, msg.WeekNumber)
	if err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&workoutsv1.ListLogsResponse{
		Logs: mapLogs(logs),
	}), nil
}

func (h *workoutHandler) UpsertLog(
	ctx context.Context,
	req *connect.Request[workoutsv1.UpsertLogRequest],
) (*connect.Response[workoutsv1.UpsertLogResponse], error) {
	msg := req.Msg
	if msg.ExerciseId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}

	uid, err := parseUUID(msg.ExerciseId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	var weightKg *float64
	if msg.WeightKg != nil {
		v := msg.GetWeightKg()
		weightKg = &v
	}

	log, err := h.svc.UpsertLog(ctx, dao.UpsertLogParams{
		ExerciseID: uid,
		WeekNumber: int16(msg.WeekNumber),
		Sets:       int16PtrFromInt32(msg.Sets),
		Reps:       msg.Reps,
		WeightKg:   weightKg,
	})
	if err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&workoutsv1.UpsertLogResponse{Log: mapLog(log)}), nil
}

func (h *workoutHandler) DeleteLog(
	ctx context.Context,
	req *connect.Request[workoutsv1.DeleteLogRequest],
) (*connect.Response[workoutsv1.DeleteLogResponse], error) {
	if err := h.svc.DeleteLog(ctx, req.Msg.Id); err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&workoutsv1.DeleteLogResponse{}), nil
}

// --- mapping helpers ---

func mapWorkoutTypes(types []model.WorkoutType) []*workoutsv1.WorkoutType {
	result := make([]*workoutsv1.WorkoutType, len(types))
	for i := range types {
		result[i] = mapWorkoutType(&types[i])
	}
	return result
}

func mapWorkoutType(wt *model.WorkoutType) *workoutsv1.WorkoutType {
	return &workoutsv1.WorkoutType{
		Id:        wt.ID.String(),
		Name:      wt.Name,
		SortOrder: int32(wt.SortOrder),
		Exercises: mapExercises(wt.Exercises),
		CreatedAt: wt.CreatedAt.Format(time.RFC3339),
	}
}

func mapExercises(exs []model.Exercise) []*workoutsv1.Exercise {
	result := make([]*workoutsv1.Exercise, len(exs))
	for i := range exs {
		result[i] = mapExercise(&exs[i])
	}
	return result
}

func mapExercise(ex *model.Exercise) *workoutsv1.Exercise {
	return &workoutsv1.Exercise{
		Id:            ex.ID.String(),
		WorkoutTypeId: ex.WorkoutTypeID.String(),
		Name:          ex.Name,
		SortOrder:     int32(ex.SortOrder),
	}
}

func mapLogs(logs []model.WorkoutLog) []*workoutsv1.WorkoutLog {
	result := make([]*workoutsv1.WorkoutLog, len(logs))
	for i := range logs {
		result[i] = mapLog(&logs[i])
	}
	return result
}

func mapLog(l *model.WorkoutLog) *workoutsv1.WorkoutLog {
	pb := &workoutsv1.WorkoutLog{
		Id:         l.ID.String(),
		ExerciseId: l.ExerciseID.String(),
		WeekNumber: int32(l.WeekNumber),
		LoggedAt:   l.LoggedAt.Format(time.RFC3339),
	}
	if l.Sets != nil {
		v := int32(*l.Sets)
		pb.Sets = &v
	}
	if l.Reps != nil {
		pb.Reps = l.Reps
	}
	if l.WeightKg != nil {
		pb.WeightKg = l.WeightKg
	}
	return pb
}
