import { useState, useCallback } from 'react'
import { useLiveQuery } from 'dexie-react-hooks'
import { db } from '@/lib/db'
import {
  createWorkoutType,
  deleteWorkoutType,
  createExercise as createExerciseOp,
  deleteExercise as deleteExerciseOp,
  upsertWorkoutLog,
  deleteWorkoutLog,
} from '@/lib/operations'
import type { WorkoutType, Exercise, WorkoutLog } from '@/types/api'

export function useWorkouts() {
  const [logs, setLogs] = useState<WorkoutLog[]>([])

  const rawTypes = useLiveQuery(async () => {
    const allTypes = await db.workoutTypes.filter(t => !t.deleted).toArray()
    const allExercises = await db.exercises.filter(e => !e.deleted).toArray()

    return allTypes.map(t => ({
      id: t.id,
      name: t.name,
      sortOrder: t.sortOrder,
      createdAt: t.createdAt,
      exercises: allExercises
        .filter(e => e.workoutTypeId === t.id)
        .sort((a, b) => a.sortOrder - b.sortOrder)
        .map(e => ({ id: e.id, workoutTypeId: e.workoutTypeId, name: e.name, sortOrder: e.sortOrder })),
    }))
  })

  const types: WorkoutType[] = rawTypes ?? []

  const fetchLogsForType = useCallback(
    async (typeId: string, currentTypes: WorkoutType[]): Promise<WorkoutLog[]> => {
      const type = currentTypes.find(t => t.id === typeId)
      if (!type?.exercises?.length) {
        setLogs([])
        return []
      }
      const exerciseIds = new Set(type.exercises.map(e => e.id))
      const allLogs = await db.workoutLogs.filter(l => !l.deleted && exerciseIds.has(l.exerciseId)).toArray()
      const mapped: WorkoutLog[] = allLogs.map(l => ({
        id: l.id, exerciseId: l.exerciseId, weekNumber: l.weekNumber,
        sets: l.sets, reps: l.reps, weightKg: l.weightKg, loggedAt: l.loggedAt,
      }))
      setLogs(mapped)
      return mapped
    },
    [],
  )

  const createType = useCallback(async (name: string): Promise<WorkoutType> => {
    const rec = await createWorkoutType(name)
    return { ...rec, exercises: [] }
  }, [])

  const deleteType = useCallback(async (id: string): Promise<void> => {
    await deleteWorkoutType(id)
    setLogs([])
  }, [])

  const createExercise = useCallback(
    async (typeId: string, name: string): Promise<Exercise> => {
      const sortOrder = types.find(t => t.id === typeId)?.exercises?.length ?? 0
      const rec = await createExerciseOp(typeId, name, sortOrder)
      return { id: rec.id, workoutTypeId: rec.workoutTypeId, name: rec.name, sortOrder: rec.sortOrder }
    },
    [types],
  )

  const deleteExercise = useCallback(
    async (_typeId: string, id: string): Promise<void> => {
      await deleteExerciseOp(id)
      setLogs(prev => prev.filter(l => l.exerciseId !== id))
    },
    [],
  )

  const upsertLog = useCallback(
    async (
      exerciseId: string, weekNumber: number,
      sets?: number, reps?: string, weightKg?: number,
    ): Promise<WorkoutLog> => {
      const rec = await upsertWorkoutLog(exerciseId, weekNumber, sets, reps, weightKg)
      const log: WorkoutLog = {
        id: rec.id, exerciseId: rec.exerciseId, weekNumber: rec.weekNumber,
        sets: rec.sets, reps: rec.reps, weightKg: rec.weightKg, loggedAt: rec.loggedAt,
      }
      setLogs(prev => {
        const idx = prev.findIndex(l => l.exerciseId === exerciseId && l.weekNumber === weekNumber)
        if (idx >= 0) return prev.map((l, i) => (i === idx ? log : l))
        return [...prev, log]
      })
      return log
    },
    [],
  )

  const deleteLog = useCallback(async (id: string): Promise<void> => {
    await deleteWorkoutLog(id)
    setLogs(prev => prev.filter(l => l.id !== id))
  }, [])

  const deleteWeekLogs = useCallback(
    async (weekNumber: number): Promise<void> => {
      const weekLogs = logs.filter(l => l.weekNumber === weekNumber)
      await Promise.all(weekLogs.map(l => deleteWorkoutLog(l.id)))
      setLogs(prev => prev.filter(l => l.weekNumber !== weekNumber))
    },
    [logs],
  )

  const refetch = useCallback(async () => {}, [])

  return {
    types, logs, loading: rawTypes === undefined, error: null, refetch,
    fetchLogsForType, createType, deleteType, createExercise, deleteExercise,
    upsertLog, deleteLog, deleteWeekLogs,
  }
}
