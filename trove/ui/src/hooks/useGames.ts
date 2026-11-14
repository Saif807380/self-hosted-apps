import { useCallback } from 'react'
import { useLiveQuery } from 'dexie-react-hooks'
import { db } from '@/lib/db'
import {
  createGame as createGameOp,
  updateGame as updateGameOp,
  deleteGame as deleteGameOp,
  setGameStatus as setGameStatusOp,
} from '@/lib/operations'
import type { GameStatus, VideoGame, VideoGameFormData } from '@/types/api'

const NO_COUNTS: Record<GameStatus, number> = { wishlist: 0, owned: 0, played: 0, dropped: 0 }

export function useGames({
  search = '',
  yearPlayed,
  status,
}: {
  search?: string
  yearPlayed?: number
  status?: GameStatus
} = {}) {
  const raw = useLiveQuery(async () => {
    const allGames = await db.videoGames.filter(g => !g.deleted).toArray()
    const allYears = await db.gameYearsPlayed.toArray()

    return allGames.map((g): VideoGame => ({
      id: g.id,
      title: g.title,
      studio: g.studio,
      rating: g.rating,
      review: g.review,
      coverImage: g.coverImage,
      status: g.status ?? 'played',
      platform: g.platform,
      targetPriceInr: g.targetPriceInr,
      createdAt: g.createdAt,
      updatedAt: g.updatedAt,
      yearsPlayed: allYears.filter(y => y.gameId === g.id).map(y => y.year),
    }))
  })

  const all: VideoGame[] = raw ?? []
  const counts = all.reduce((acc, g) => ({ ...acc, [g.status]: acc[g.status] + 1 }), NO_COUNTS)

  let games = status ? all.filter(g => g.status === status) : all
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

  const setStatus = useCallback(async (id: string, next: GameStatus): Promise<void> => {
    await setGameStatusOp(id, next)
  }, [])

  const refetch = useCallback(async () => {}, [])

  return {
    games, counts, loading: raw === undefined, error: null,
    refetch, createGame, updateGame, deleteGame, setStatus,
  }
}
