import Dexie, { type Table } from 'dexie'

// Flat DB record types (no embedded relations)

export interface BookRecord {
  id: string
  title: string
  author: string
  rating?: number
  review?: string
  coverImage?: string
  createdAt: string
  updatedAt: string
  deleted: boolean
}

export interface TagRecord {
  id: string
  name: string
  updatedAt: string
  deleted: boolean
}

export interface CollectionRecord {
  id: string
  name: string
  updatedAt: string
  deleted: boolean
}

export interface VideoGameRecord {
  id: string
  title: string
  studio?: string
  rating?: number
  review?: string
  coverImage?: string
  yearsPlayed: number[]
  createdAt: string
  updatedAt: string
  deleted: boolean
}

export interface TravelLocationRecord {
  id: string
  city: string
  country: string
  visitedFrom?: string
  visitedTo?: string
  photoCollectionUrl?: string
  createdAt: string
  updatedAt: string
  deleted: boolean
}

export interface TouristSpotRecord {
  id: string
  locationId: string
  name: string
  description?: string
  updatedAt: string
  deleted: boolean
}

export interface WorkoutTypeRecord {
  id: string
  name: string
  sortOrder: number
  createdAt: string
  updatedAt: string
  deleted: boolean
}

export interface ExerciseRecord {
  id: string
  workoutTypeId: string
  name: string
  sortOrder: number
  updatedAt: string
  deleted: boolean
}

export interface WorkoutLogRecord {
  id: string
  exerciseId: string
  weekNumber: number
  sets?: number
  reps?: string
  weightKg?: number
  loggedAt: string
  updatedAt: string
  deleted: boolean
}

// Junction table records

export interface BookYearReadRecord {
  bookId: string
  year: number
}

export interface BookTagRecord {
  bookId: string
  tagId: string
}

export interface CollectionBookRecord {
  collectionId: string
  bookId: string
}

export interface GameYearPlayedRecord {
  gameId: string
  year: number
}

// Sync records

export interface SyncOutboxEntry {
  id?: number
  table: string
  entityId: string
}

export interface SyncMetaRecord {
  key: string
  value: string
}

// Image blob record — stores cover images locally before/after upload
export interface ImageBlobRecord {
  id: string        // UUID (no extension)
  ext: string       // e.g. '.jpg', '.png', '.webp'
  blob: Blob
  checksum: string  // SHA-256 hex
  uploaded: boolean
}

class TroveDB extends Dexie {
  books!: Table<BookRecord>
  tags!: Table<TagRecord>
  collections!: Table<CollectionRecord>
  videoGames!: Table<VideoGameRecord>
  travelLocations!: Table<TravelLocationRecord>
  touristSpots!: Table<TouristSpotRecord>
  workoutTypes!: Table<WorkoutTypeRecord>
  exercises!: Table<ExerciseRecord>
  workoutLogs!: Table<WorkoutLogRecord>
  bookYearsRead!: Table<BookYearReadRecord>
  bookTags!: Table<BookTagRecord>
  collectionBooks!: Table<CollectionBookRecord>
  gameYearsPlayed!: Table<GameYearPlayedRecord>
  syncOutbox!: Table<SyncOutboxEntry>
  syncMeta!: Table<SyncMetaRecord>
  imageBlobs!: Table<ImageBlobRecord>

  constructor() {
    super('TroveDB')
    this.version(1).stores({
      books: 'id, updatedAt',
      tags: 'id, updatedAt',
      collections: 'id, updatedAt',
      videoGames: 'id, updatedAt',
      travelLocations: 'id, updatedAt',
      touristSpots: 'id, locationId, updatedAt',
      workoutTypes: 'id, updatedAt',
      exercises: 'id, workoutTypeId, updatedAt',
      workoutLogs: 'id, exerciseId, updatedAt',
      bookYearsRead: '[bookId+year], bookId',
      bookTags: '[bookId+tagId], bookId, tagId',
      collectionBooks: '[collectionId+bookId], collectionId, bookId',
      gameYearsPlayed: '[gameId+year], gameId',
      syncOutbox: '++id, table, entityId',
      syncMeta: 'key',
    })
    this.version(2).stores({
      imageBlobs: 'id, uploaded',
    })
  }
}

export const db = new TroveDB()
