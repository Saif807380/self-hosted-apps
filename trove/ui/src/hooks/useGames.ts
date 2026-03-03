import { useState, useEffect, useCallback } from 'react'
import { gamesApi } from '@/services/games'
import type { VideoGame, VideoGameFormData } from '@/types/api'

export function useGames({
  search = '',
  yearPlayed,
}: {
  search?: string
  yearPlayed?: number
} = {}) {
  const [games, setGames] = useState<VideoGame[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const refetch = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await gamesApi.list({ search: search || undefined, yearPlayed })
      setGames(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load games')
    } finally {
      setLoading(false)
    }
  }, [search, yearPlayed])

  useEffect(() => {
    refetch()
  }, [refetch])

  const createGame = useCallback(
    async (data: VideoGameFormData): Promise<VideoGame> => {
      const game = await gamesApi.create(data)
      await refetch()
      return game
    },
    [refetch],
  )

  const updateGame = useCallback(
    async (id: string, data: VideoGameFormData): Promise<VideoGame> => {
      const game = await gamesApi.update(id, data)
      await refetch()
      return game
    },
    [refetch],
  )

  const deleteGame = useCallback(
    async (id: string): Promise<void> => {
      await gamesApi.delete(id)
      setGames(prev => prev.filter(g => g.id !== id))
    },
    [],
  )

  return { games, loading, error, refetch, createGame, updateGame, deleteGame }
}
