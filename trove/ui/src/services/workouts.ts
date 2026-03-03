import type { WorkoutType, Exercise, WorkoutLog } from '@/types/api'
import { rpc } from './client'

const SVC = 'trove.v1.workouts.WorkoutService'

export const workoutsApi = {
  listTypes: (): Promise<WorkoutType[]> =>
    rpc<{ workoutTypes?: WorkoutType[] }>(SVC, 'ListWorkoutTypes', {})
      .then(r => r.workoutTypes ?? []),

  createType: (name: string, sortOrder = 0): Promise<WorkoutType> =>
    rpc<{ workoutType: WorkoutType }>(SVC, 'CreateWorkoutType', { name, sortOrder })
      .then(r => r.workoutType),

  deleteType: (id: string): Promise<void> =>
    rpc<Record<string, never>>(SVC, 'DeleteWorkoutType', { id }).then(() => undefined),

  createExercise: (workoutTypeId: string, name: string, sortOrder = 0): Promise<Exercise> =>
    rpc<{ exercise: Exercise }>(SVC, 'CreateExercise', { workoutTypeId, name, sortOrder })
      .then(r => r.exercise),

  deleteExercise: (id: string): Promise<void> =>
    rpc<Record<string, never>>(SVC, 'DeleteExercise', { id }).then(() => undefined),

  listLogs: (exerciseId: string): Promise<WorkoutLog[]> =>
    rpc<{ logs?: WorkoutLog[] }>(SVC, 'ListLogs', { exerciseId })
      .then(r => r.logs ?? []),

  upsertLog: (
    exerciseId: string,
    weekNumber: number,
    sets?: number,
    reps?: string,
    weightKg?: number,
  ): Promise<WorkoutLog> =>
    rpc<{ log: WorkoutLog }>(SVC, 'UpsertLog', { exerciseId, weekNumber, sets, reps, weightKg })
      .then(r => r.log),

  deleteLog: (id: string): Promise<void> =>
    rpc<Record<string, never>>(SVC, 'DeleteLog', { id }).then(() => undefined),
}
