export interface Tag {
  id: string
  name: string
}

export interface Collection {
  id: string
  name: string
}

export interface Book {
  id: string
  title: string
  author: string
  rating?: number
  review?: string
  coverImage?: string
  yearsRead: number[]
  tags: Tag[]
  createdAt: string
  updatedAt: string
}

export interface BookFormData {
  title: string
  author: string
  rating?: number
  review?: string
  coverImage?: string
  yearsRead: number[]
  tagIds: string[]
}

export interface VideoGame {
  id: string
  title: string
  studio?: string
  rating?: number
  review?: string
  coverImage?: string
  yearsPlayed: number[]
  createdAt: string
  updatedAt: string
}

export interface TouristSpot {
  id: string
  locationId: string
  name: string
  description?: string
}

export interface TravelLocation {
  id: string
  city: string
  country: string
  visitedFrom?: string
  visitedTo?: string
  photoCollectionUrl?: string
  touristSpots: TouristSpot[]
  createdAt: string
  updatedAt: string
}

export interface Exercise {
  id: string
  workoutTypeId: string
  name: string
  sortOrder: number
}

export interface WorkoutType {
  id: string
  name: string
  sortOrder: number
  exercises: Exercise[]
  createdAt: string
}

export interface WorkoutLog {
  id: string
  exerciseId: string
  weekNumber: number
  sets?: number
  reps?: string
  weightKg?: number
  loggedAt: string
}
