import { useState, useEffect, useCallback } from 'react'
import { workoutsApi } from '@/services/workouts'
import type { WorkoutType, Exercise, WorkoutLog } from '@/types/api'

export function useWorkouts() {
  const [types, setTypes] = useState<WorkoutType[]>([])
  const [logs, setLogs] = useState<WorkoutLog[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const refetch = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await workoutsApi.listTypes()
      setTypes(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load workouts')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { refetch() }, [refetch])

  const fetchLogsForType = useCallback(async (typeId: string, currentTypes: WorkoutType[]): Promise<WorkoutLog[]> => {
    const type = currentTypes.find(t => t.id === typeId)
    if (!type?.exercises.length) { setLogs([]); return [] }
    const results = await Promise.all(type.exercises.map(e => workoutsApi.listLogs(e.id)))
    const flat = results.flat()
    setLogs(flat)
    return flat
  }, [])

  const createType = useCallback(async (name: string): Promise<WorkoutType> => {
    const type = await workoutsApi.createType(name)
    const newType = { ...type, exercises: [] }
    setTypes(prev => [...prev, newType])
    return newType
  }, [])

  const deleteType = useCallback(async (id: string): Promise<void> => {
    await workoutsApi.deleteType(id)
    setTypes(prev => prev.filter(t => t.id !== id))
    setLogs([])
  }, [])

  const createExercise = useCallback(async (typeId: string, name: string): Promise<Exercise> => {
    const sortOrder = (types.find(t => t.id === typeId)?.exercises.length ?? 0)
    const exercise = await workoutsApi.createExercise(typeId, name, sortOrder)
    setTypes(prev => prev.map(t =>
      t.id === typeId ? { ...t, exercises: [...t.exercises, exercise] } : t
    ))
    return exercise
  }, [types])

  const deleteExercise = useCallback(async (typeId: string, id: string): Promise<void> => {
    await workoutsApi.deleteExercise(id)
    setTypes(prev => prev.map(t =>
      t.id === typeId ? { ...t, exercises: t.exercises.filter(e => e.id !== id) } : t
    ))
    setLogs(prev => prev.filter(l => l.exerciseId !== id))
  }, [])

  const upsertLog = useCallback(async (
    exerciseId: string,
    weekNumber: number,
    sets?: number,
    reps?: string,
    weightKg?: number,
  ): Promise<WorkoutLog> => {
    const log = await workoutsApi.upsertLog(exerciseId, weekNumber, sets, reps, weightKg)
    setLogs(prev => {
      const idx = prev.findIndex(l => l.exerciseId === exerciseId && l.weekNumber === weekNumber)
      if (idx >= 0) return prev.map((l, i) => i === idx ? log : l)
      return [...prev, log]
    })
    return log
  }, [])

  const deleteLog = useCallback(async (id: string): Promise<void> => {
    await workoutsApi.deleteLog(id)
    setLogs(prev => prev.filter(l => l.id !== id))
  }, [])

  const deleteWeekLogs = useCallback(async (weekNumber: number): Promise<void> => {
    const weekLogs = logs.filter(l => l.weekNumber === weekNumber)
    await Promise.all(weekLogs.map(l => workoutsApi.deleteLog(l.id)))
    setLogs(prev => prev.filter(l => l.weekNumber !== weekNumber))
  }, [logs])

  return {
    types, logs, loading, error,
    refetch, fetchLogsForType,
    createType, deleteType,
    createExercise, deleteExercise,
    upsertLog, deleteLog, deleteWeekLogs,
  }
}
