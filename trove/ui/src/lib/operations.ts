import { db } from './db'
import type {
  BookRecord, TagRecord, CollectionRecord, VideoGameRecord,
  TravelLocationRecord, TouristSpotRecord, WorkoutTypeRecord,
  ExerciseRecord, WorkoutLogRecord,
} from './db'

const now = () => new Date().toISOString()
const uuid = () => crypto.randomUUID()

function enqueue(table: string, entityId: string) {
  return db.syncOutbox.add({ table, entityId })
}

// ── Books ──

export async function createBook(data: {
  title: string; author: string; rating?: number; review?: string;
  coverImage?: string; yearsRead: number[]; tagIds: string[]
}): Promise<BookRecord> {
  const id = uuid()
  const ts = now()
  const book: BookRecord = {
    id, title: data.title, author: data.author,
    rating: data.rating, review: data.review, coverImage: data.coverImage,
    createdAt: ts, updatedAt: ts, deleted: false,
  }
  await db.transaction('rw', [db.books, db.bookYearsRead, db.bookTags, db.syncOutbox], async () => {
    await db.books.put(book)
    await db.bookYearsRead.bulkPut(data.yearsRead.map(year => ({ bookId: id, year })))
    await db.bookTags.bulkPut(data.tagIds.map(tagId => ({ bookId: id, tagId })))
    await enqueue('books', id)
  })
  return book
}

export async function updateBook(id: string, data: {
  title: string; author: string; rating?: number; review?: string;
  coverImage?: string; yearsRead: number[]; tagIds: string[]
}): Promise<BookRecord> {
  const existing = await db.books.get(id)
  if (!existing) throw new Error('Book not found')
  const book: BookRecord = {
    ...existing, title: data.title, author: data.author,
    rating: data.rating, review: data.review, coverImage: data.coverImage,
    updatedAt: now(), deleted: false,
  }
  await db.transaction('rw', [db.books, db.bookYearsRead, db.bookTags, db.syncOutbox], async () => {
    await db.books.put(book)
    await db.bookYearsRead.where('bookId').equals(id).delete()
    await db.bookYearsRead.bulkPut(data.yearsRead.map(year => ({ bookId: id, year })))
    await db.bookTags.where('bookId').equals(id).delete()
    await db.bookTags.bulkPut(data.tagIds.map(tagId => ({ bookId: id, tagId })))
    await enqueue('books', id)
  })
  return book
}

export async function deleteBook(id: string): Promise<void> {
  await db.transaction('rw', [db.books, db.syncOutbox], async () => {
    await db.books.update(id, { deleted: true, updatedAt: now() })
    await enqueue('books', id)
  })
}

// ── Tags ──

export async function createTag(name: string): Promise<TagRecord> {
  const tag: TagRecord = { id: uuid(), name, updatedAt: now(), deleted: false }
  await db.transaction('rw', [db.tags, db.syncOutbox], async () => {
    await db.tags.put(tag)
    await enqueue('tags', tag.id)
  })
  return tag
}

// ── Collections ──

export async function createCollection(name: string): Promise<CollectionRecord> {
  const col: CollectionRecord = { id: uuid(), name, updatedAt: now(), deleted: false }
  await db.transaction('rw', [db.collections, db.syncOutbox], async () => {
    await db.collections.put(col)
    await enqueue('collections', col.id)
  })
  return col
}

export async function addToCollection(collectionId: string, bookId: string): Promise<void> {
  await db.transaction('rw', [db.collectionBooks, db.collections, db.syncOutbox], async () => {
    await db.collectionBooks.put({ collectionId, bookId })
    await db.collections.update(collectionId, { updatedAt: now() })
    await enqueue('collections', collectionId)
  })
}

export async function removeFromCollection(collectionId: string, bookId: string): Promise<void> {
  await db.transaction('rw', [db.collectionBooks, db.collections, db.syncOutbox], async () => {
    await db.collectionBooks.where('[collectionId+bookId]').equals([collectionId, bookId]).delete()
    await db.collections.update(collectionId, { updatedAt: now() })
    await enqueue('collections', collectionId)
  })
}

// ── Video Games ──

export async function createGame(data: {
  title: string; studio?: string; rating?: number; review?: string;
  coverImage?: string; yearsPlayed: number[]
}): Promise<VideoGameRecord> {
  const id = uuid()
  const ts = now()
  const game: VideoGameRecord = {
    id, title: data.title, studio: data.studio,
    rating: data.rating, review: data.review, coverImage: data.coverImage,
    yearsPlayed: data.yearsPlayed, createdAt: ts, updatedAt: ts, deleted: false,
  }
  await db.transaction('rw', [db.videoGames, db.gameYearsPlayed, db.syncOutbox], async () => {
    await db.videoGames.put(game)
    await db.gameYearsPlayed.bulkPut(data.yearsPlayed.map(year => ({ gameId: id, year })))
    await enqueue('videoGames', id)
  })
  return game
}

export async function updateGame(id: string, data: {
  title: string; studio?: string; rating?: number; review?: string;
  coverImage?: string; yearsPlayed: number[]
}): Promise<VideoGameRecord> {
  const existing = await db.videoGames.get(id)
  if (!existing) throw new Error('Game not found')
  const game: VideoGameRecord = {
    ...existing, title: data.title, studio: data.studio,
    rating: data.rating, review: data.review, coverImage: data.coverImage,
    yearsPlayed: data.yearsPlayed, updatedAt: now(), deleted: false,
  }
  await db.transaction('rw', [db.videoGames, db.gameYearsPlayed, db.syncOutbox], async () => {
    await db.videoGames.put(game)
    await db.gameYearsPlayed.where('gameId').equals(id).delete()
    await db.gameYearsPlayed.bulkPut(data.yearsPlayed.map(year => ({ gameId: id, year })))
    await enqueue('videoGames', id)
  })
  return game
}

export async function deleteGame(id: string): Promise<void> {
  await db.transaction('rw', [db.videoGames, db.syncOutbox], async () => {
    await db.videoGames.update(id, { deleted: true, updatedAt: now() })
    await enqueue('videoGames', id)
  })
}

// ── Travel Locations ──

export async function createLocation(data: {
  city: string; country: string; visitedFrom?: string; visitedTo?: string;
  photoCollectionUrl?: string
}): Promise<TravelLocationRecord> {
  const ts = now()
  const loc: TravelLocationRecord = {
    id: uuid(), city: data.city, country: data.country,
    visitedFrom: data.visitedFrom, visitedTo: data.visitedTo,
    photoCollectionUrl: data.photoCollectionUrl,
    createdAt: ts, updatedAt: ts, deleted: false,
  }
  await db.transaction('rw', [db.travelLocations, db.syncOutbox], async () => {
    await db.travelLocations.put(loc)
    await enqueue('travelLocations', loc.id)
  })
  return loc
}

export async function updateLocation(id: string, data: {
  city: string; country: string; visitedFrom?: string; visitedTo?: string;
  photoCollectionUrl?: string
}): Promise<TravelLocationRecord> {
  const existing = await db.travelLocations.get(id)
  if (!existing) throw new Error('Location not found')
  const loc: TravelLocationRecord = {
    ...existing, city: data.city, country: data.country,
    visitedFrom: data.visitedFrom, visitedTo: data.visitedTo,
    photoCollectionUrl: data.photoCollectionUrl,
    updatedAt: now(), deleted: false,
  }
  await db.transaction('rw', [db.travelLocations, db.syncOutbox], async () => {
    await db.travelLocations.put(loc)
    await enqueue('travelLocations', id)
  })
  return loc
}

export async function deleteLocation(id: string): Promise<void> {
  await db.transaction('rw', [db.travelLocations, db.syncOutbox], async () => {
    await db.travelLocations.update(id, { deleted: true, updatedAt: now() })
    await enqueue('travelLocations', id)
  })
}

// ── Tourist Spots ──

export async function createSpot(locationId: string, name: string, description?: string): Promise<TouristSpotRecord> {
  const spot: TouristSpotRecord = {
    id: uuid(), locationId, name, description, updatedAt: now(), deleted: false,
  }
  await db.transaction('rw', [db.touristSpots, db.syncOutbox], async () => {
    await db.touristSpots.put(spot)
    await enqueue('touristSpots', spot.id)
  })
  return spot
}

export async function updateSpot(id: string, name: string, description?: string): Promise<TouristSpotRecord> {
  const existing = await db.touristSpots.get(id)
  if (!existing) throw new Error('Spot not found')
  const spot: TouristSpotRecord = { ...existing, name, description, updatedAt: now() }
  await db.transaction('rw', [db.touristSpots, db.syncOutbox], async () => {
    await db.touristSpots.put(spot)
    await enqueue('touristSpots', id)
  })
  return spot
}

export async function deleteSpot(id: string): Promise<void> {
  await db.transaction('rw', [db.touristSpots, db.syncOutbox], async () => {
    await db.touristSpots.update(id, { deleted: true, updatedAt: now() })
    await enqueue('touristSpots', id)
  })
}

// ── Workout Types ──

export async function createWorkoutType(name: string): Promise<WorkoutTypeRecord> {
  const ts = now()
  const wt: WorkoutTypeRecord = {
    id: uuid(), name, sortOrder: 0, createdAt: ts, updatedAt: ts, deleted: false,
  }
  await db.transaction('rw', [db.workoutTypes, db.syncOutbox], async () => {
    await db.workoutTypes.put(wt)
    await enqueue('workoutTypes', wt.id)
  })
  return wt
}

export async function deleteWorkoutType(id: string): Promise<void> {
  await db.transaction('rw', [db.workoutTypes, db.syncOutbox], async () => {
    await db.workoutTypes.update(id, { deleted: true, updatedAt: now() })
    await enqueue('workoutTypes', id)
  })
}

// ── Exercises ──

export async function createExercise(workoutTypeId: string, name: string, sortOrder: number): Promise<ExerciseRecord> {
  const ex: ExerciseRecord = {
    id: uuid(), workoutTypeId, name, sortOrder, updatedAt: now(), deleted: false,
  }
  await db.transaction('rw', [db.exercises, db.syncOutbox], async () => {
    await db.exercises.put(ex)
    await enqueue('exercises', ex.id)
  })
  return ex
}

export async function deleteExercise(id: string): Promise<void> {
  await db.transaction('rw', [db.exercises, db.syncOutbox], async () => {
    await db.exercises.update(id, { deleted: true, updatedAt: now() })
    await enqueue('exercises', id)
  })
}

// ── Workout Logs ──

export async function upsertWorkoutLog(
  exerciseId: string, weekNumber: number,
  sets?: number, reps?: string, weightKg?: number,
): Promise<WorkoutLogRecord> {
  const ts = now()
  // Check if a log already exists for this exercise+week
  const existing = await db.workoutLogs
    .filter(l => l.exerciseId === exerciseId && l.weekNumber === weekNumber && !l.deleted)
    .first()

  const log: WorkoutLogRecord = {
    id: existing?.id ?? uuid(),
    exerciseId, weekNumber, sets, reps, weightKg,
    loggedAt: existing?.loggedAt ?? ts,
    updatedAt: ts, deleted: false,
  }
  await db.transaction('rw', [db.workoutLogs, db.syncOutbox], async () => {
    await db.workoutLogs.put(log)
    await enqueue('workoutLogs', log.id)
  })
  return log
}

export async function deleteWorkoutLog(id: string): Promise<void> {
  await db.transaction('rw', [db.workoutLogs, db.syncOutbox], async () => {
    await db.workoutLogs.update(id, { deleted: true, updatedAt: now() })
    await enqueue('workoutLogs', id)
  })
}
