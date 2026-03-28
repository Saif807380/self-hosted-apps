import { useCallback } from 'react'
import { useLiveQuery } from 'dexie-react-hooks'
import { db } from '@/lib/db'
import {
  createGame as createGameOp,
  updateGame as updateGameOp,
  deleteGame as deleteGameOp,
} from '@/lib/operations'
import type { VideoGame, VideoGameFormData } from '@/types/api'

export function useGames({
  search = '',
  yearPlayed,
}: {
  search?: string
  yearPlayed?: number
} = {}) {
  const raw = useLiveQuery(async () => {
    const allGames = await db.videoGames.filter(g => !g.deleted).toArray()
    const allYears = await db.gameYearsPlayed.toArray()

    return allGames.map(g => ({
      id: g.id,
      title: g.title,
      studio: g.studio,
      rating: g.rating,
      review: g.review,
      coverImage: g.coverImage,
      createdAt: g.createdAt,
      updatedAt: g.updatedAt,
      yearsPlayed: allYears.filter(y => y.gameId === g.id).map(y => y.year),
    }))
  })

  let games: VideoGame[] = raw ?? []
  if (search) {
    const q = search.toLowerCase()
    games = games.filter(g => g.title.toLowerCase().includes(q))
  }
  if (yearPlayed) {
    games = games.filter(g => g.yearsPlayed.includes(yearPlayed))
  }

  const createGame = useCallback(async (data: VideoGameFormData): Promise<VideoGame> => {
    const rec = await createGameOp(data)
    return { ...rec, yearsPlayed: data.yearsPlayed }
  }, [])

  const updateGame = useCallback(async (id: string, data: VideoGameFormData): Promise<VideoGame> => {
    const rec = await updateGameOp(id, data)
    return { ...rec, yearsPlayed: data.yearsPlayed }
  }, [])

  const deleteGame = useCallback(async (id: string): Promise<void> => {
    await deleteGameOp(id)
  }, [])

  const refetch = useCallback(async () => {}, [])

  return { games, loading: raw === undefined, error: null, refetch, createGame, updateGame, deleteGame }
}
