import { db } from './db'
import type {
  BookRecord, TagRecord, CollectionRecord, VideoGameRecord,
  TravelLocationRecord, TouristSpotRecord, WorkoutTypeRecord,
  ExerciseRecord, WorkoutLogRecord,
} from './db'
import { rpc } from '@/services/client'
import { pushImages, pullImages } from './image-sync'

const SVC = 'trove.v1.sync.SyncService'
const EPOCH = '2000-01-01T00:00:00Z'

// ── Pull: server → IndexedDB ──

interface PullResponse {
  changes: {
    books?: PullBook[]
    tags?: PullTag[]
    collections?: PullCollection[]
    videoGames?: PullVideoGame[]
    travelLocations?: PullTravelLocation[]
    touristSpots?: PullTouristSpot[]
    workoutTypes?: PullWorkoutType[]
    exercises?: PullExercise[]
    workoutLogs?: PullWorkoutLog[]
    bookYearsRead?: { bookId: string; year: number }[]
    bookTags?: { bookId: string; tagId: string }[]
    collectionBooks?: { collectionId: string; bookId: string }[]
    gameYearsPlayed?: { gameId: string; year: number }[]
  }
  serverTime: string
}

interface PullBook {
  id: string; title: string; author: string; rating?: number; review?: string
  coverImage?: string; tags?: { id: string; name: string }[]
  yearsRead?: number[]; createdAt: string; updatedAt: string; deleted: boolean
}
interface PullTag { id: string; name: string; updatedAt: string; deleted: boolean }
interface PullCollection { id: string; name: string; updatedAt: string; deleted: boolean }
interface PullVideoGame {
  id: string; title: string; studio?: string; rating?: number; review?: string
  coverImage?: string; yearsPlayed?: number[]; createdAt: string; updatedAt: string; deleted: boolean
}
interface PullTravelLocation {
  id: string; city: string; country: string; visitedFrom?: string; visitedTo?: string
  photoCollectionUrl?: string; touristSpots?: PullTouristSpot[]
  createdAt: string; updatedAt: string; deleted: boolean
}
interface PullTouristSpot {
  id: string; locationId: string; name: string; description?: string
  updatedAt: string; deleted: boolean
}
interface PullWorkoutType {
  id: string; name: string; sortOrder: number; exercises?: PullExercise[]
  createdAt: string; updatedAt: string; deleted: boolean
}
interface PullExercise {
  id: string; workoutTypeId: string; name: string; sortOrder: number
  updatedAt: string; deleted: boolean
}
interface PullWorkoutLog {
  id: string; exerciseId: string; weekNumber: number; sets?: number
  reps?: string; weightKg?: number; loggedAt: string; updatedAt: string; deleted: boolean
}

export async function pull(): Promise<string> {
  const meta = await db.syncMeta.get('lastSync')
  const since = meta?.value ?? EPOCH
  console.debug('[sync-engine] pull since =', since)

  const res = await rpc<PullResponse>(SVC, 'PullChanges', { since })
  const c = res.changes ?? {}
  console.debug('[sync-engine] pull got:', {
    books: c.books?.length ?? 0,
    tags: c.tags?.length ?? 0,
    videoGames: c.videoGames?.length ?? 0,
    bookYearsRead: c.bookYearsRead?.length ?? 0,
    bookTags: c.bookTags?.length ?? 0,
    serverTime: res.serverTime,
  })

  await db.transaction('rw',
    [db.books, db.tags, db.collections, db.videoGames, db.travelLocations,
     db.touristSpots, db.workoutTypes, db.exercises, db.workoutLogs,
     db.bookYearsRead, db.bookTags, db.collectionBooks, db.gameYearsPlayed,
     db.syncMeta],
    async () => {
      // Entity tables — bulkPut (upsert)
      if (c.books?.length) {
        await db.books.bulkPut(c.books.map(b => ({
          id: b.id, title: b.title, author: b.author, rating: b.rating,
          review: b.review, coverImage: b.coverImage,
          createdAt: b.createdAt, updatedAt: b.updatedAt, deleted: b.deleted,
        })))
      }
      if (c.tags?.length) await db.tags.bulkPut(c.tags)
      if (c.collections?.length) await db.collections.bulkPut(c.collections)
      if (c.videoGames?.length) {
        await db.videoGames.bulkPut(c.videoGames.map(g => ({
          id: g.id, title: g.title, studio: g.studio, rating: g.rating,
          review: g.review, coverImage: g.coverImage,
          yearsPlayed: g.yearsPlayed ?? [],
          createdAt: g.createdAt, updatedAt: g.updatedAt, deleted: g.deleted,
        })))
      }
      if (c.travelLocations?.length) {
        await db.travelLocations.bulkPut(c.travelLocations.map(l => ({
          id: l.id, city: l.city, country: l.country,
          visitedFrom: l.visitedFrom, visitedTo: l.visitedTo,
          photoCollectionUrl: l.photoCollectionUrl,
          createdAt: l.createdAt, updatedAt: l.updatedAt, deleted: l.deleted,
        })))
      }
      if (c.touristSpots?.length) await db.touristSpots.bulkPut(c.touristSpots)
      if (c.workoutTypes?.length) {
        await db.workoutTypes.bulkPut(c.workoutTypes.map(wt => ({
          id: wt.id, name: wt.name, sortOrder: wt.sortOrder,
          createdAt: wt.createdAt, updatedAt: wt.updatedAt, deleted: wt.deleted,
        })))
      }
      if (c.exercises?.length) await db.exercises.bulkPut(c.exercises)
      if (c.workoutLogs?.length) await db.workoutLogs.bulkPut(c.workoutLogs)

      // Junction tables — delete-for-parent + re-insert
      if (c.bookYearsRead?.length) {
        const bookIds = [...new Set(c.bookYearsRead.map(r => r.bookId))]
        for (const bid of bookIds) await db.bookYearsRead.where('bookId').equals(bid).delete()
        await db.bookYearsRead.bulkPut(c.bookYearsRead)
      }
      if (c.bookTags?.length) {
        const bookIds = [...new Set(c.bookTags.map(r => r.bookId))]
        for (const bid of bookIds) await db.bookTags.where('bookId').equals(bid).delete()
        await db.bookTags.bulkPut(c.bookTags)
      }
      if (c.collectionBooks?.length) {
        const colIds = [...new Set(c.collectionBooks.map(r => r.collectionId))]
        for (const cid of colIds) await db.collectionBooks.where('collectionId').equals(cid).delete()
        await db.collectionBooks.bulkPut(c.collectionBooks)
      }
      if (c.gameYearsPlayed?.length) {
        const gameIds = [...new Set(c.gameYearsPlayed.map(r => r.gameId))]
        for (const gid of gameIds) await db.gameYearsPlayed.where('gameId').equals(gid).delete()
        await db.gameYearsPlayed.bulkPut(c.gameYearsPlayed)
      }

      await db.syncMeta.put({ key: 'lastSync', value: res.serverTime })
    },
  )

  return res.serverTime
}

// ── Push: IndexedDB outbox → server ──

export async function push(): Promise<string | null> {
  const entries = await db.syncOutbox.toArray()
  console.debug('[sync-engine] push: outbox entries =', entries.length)
  if (!entries.length) return null

  // Deduplicate by (table, entityId)
  const seen = new Set<string>()
  const unique = entries.filter(e => {
    const key = `${e.table}:${e.entityId}`
    if (seen.has(key)) return false
    seen.add(key)
    return true
  })

  // Group entity IDs by table
  const byTable = new Map<string, string[]>()
  for (const e of unique) {
    const list = byTable.get(e.table) ?? []
    list.push(e.entityId)
    byTable.set(e.table, list)
  }

  // Collect entities
  const collect = async <T>(table: string, store: { bulkGet(ids: string[]): Promise<(T | undefined)[]> }) => {
    const ids = byTable.get(table)
    if (!ids?.length) return []
    const results = await store.bulkGet(ids)
    return results.filter(Boolean) as T[]
  }

  const books = await collect<BookRecord>('books', db.books)
  const tags = await collect<TagRecord>('tags', db.tags)
  const collections = await collect<CollectionRecord>('collections', db.collections)
  const videoGames = await collect<VideoGameRecord>('videoGames', db.videoGames)
  const travelLocations = await collect<TravelLocationRecord>('travelLocations', db.travelLocations)
  const touristSpots = await collect<TouristSpotRecord>('touristSpots', db.touristSpots)
  const workoutTypes = await collect<WorkoutTypeRecord>('workoutTypes', db.workoutTypes)
  const exercises = await collect<ExerciseRecord>('exercises', db.exercises)
  const workoutLogs = await collect<WorkoutLogRecord>('workoutLogs', db.workoutLogs)

  // Collect junction data for parent entities
  const bookYearsRead = (await Promise.all(
    books.map(b => db.bookYearsRead.where('bookId').equals(b.id).toArray())
  )).flat()
  const bookTags = (await Promise.all(
    books.map(b => db.bookTags.where('bookId').equals(b.id).toArray())
  )).flat()
  const collectionBooks = (await Promise.all(
    collections.map(c => db.collectionBooks.where('collectionId').equals(c.id).toArray())
  )).flat()
  const gameYearsPlayed = (await Promise.all(
    videoGames.map(g => db.gameYearsPlayed.where('gameId').equals(g.id).toArray())
  )).flat()

  // Map to proto format and push
  const changes = {
    books: books.map(b => ({
      ...b, yearsRead: [] as number[], tags: [] as never[],
    })),
    tags,
    collections,
    videoGames,
    travelLocations: travelLocations.map(l => ({
      ...l, touristSpots: [] as never[],
    })),
    touristSpots,
    workoutTypes: workoutTypes.map(wt => ({
      ...wt, exercises: [] as never[],
    })),
    exercises,
    workoutLogs,
    bookYearsRead,
    bookTags,
    collectionBooks,
    gameYearsPlayed,
  }

  const res = await rpc<{ serverTime: string }>(SVC, 'PushChanges', { changes })

  // Clear pushed outbox entries
  const ids = entries.map(e => e.id!).filter(Boolean)
  await db.syncOutbox.bulkDelete(ids)

  return res.serverTime
}

// ── Full sync cycle ──

export async function sync(): Promise<{ pushed: boolean; serverTime: string }> {
  const pushTime = await push()
  const serverTime = await pull()
  await pushImages()
  await pullImages()
  return { pushed: pushTime !== null, serverTime }
}
