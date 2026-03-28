import { useCallback } from 'react'
import { useLiveQuery } from 'dexie-react-hooks'
import { db } from '@/lib/db'
import {
  createLocation as createLocationOp,
  updateLocation as updateLocationOp,
  deleteLocation as deleteLocationOp,
  createSpot as createSpotOp,
  updateSpot as updateSpotOp,
  deleteSpot as deleteSpotOp,
} from '@/lib/operations'
import type { TravelLocation, TravelLocationFormData, TouristSpot } from '@/types/api'

export function useTravel() {
  const raw = useLiveQuery(async () => {
    const allLocations = await db.travelLocations.filter(l => !l.deleted).toArray()
    const allSpots = await db.touristSpots.filter(s => !s.deleted).toArray()

    return allLocations.map(l => ({
      id: l.id,
      city: l.city,
      country: l.country,
      visitedFrom: l.visitedFrom,
      visitedTo: l.visitedTo,
      photoCollectionUrl: l.photoCollectionUrl,
      createdAt: l.createdAt,
      updatedAt: l.updatedAt,
      touristSpots: allSpots
        .filter(s => s.locationId === l.id)
        .map(s => ({ id: s.id, locationId: s.locationId, name: s.name, description: s.description })),
    }))
  })

  const locations: TravelLocation[] = raw ?? []

  const createLocation = useCallback(async (data: TravelLocationFormData): Promise<TravelLocation> => {
    const rec = await createLocationOp(data)
    return { ...rec, touristSpots: [] }
  }, [])

  const updateLocation = useCallback(async (id: string, data: TravelLocationFormData): Promise<TravelLocation> => {
    const rec = await updateLocationOp(id, data)
    const spots = await db.touristSpots.filter(s => s.locationId === id && !s.deleted).toArray()
    return { ...rec, touristSpots: spots }
  }, [])

  const deleteLocation = useCallback(async (id: string): Promise<void> => {
    await deleteLocationOp(id)
  }, [])

  const createSpot = useCallback(async (locationId: string, name: string, description?: string): Promise<TouristSpot> => {
    const rec = await createSpotOp(locationId, name, description)
    return { id: rec.id, locationId: rec.locationId, name: rec.name, description: rec.description }
  }, [])

  const updateSpot = useCallback(async (_locationId: string, id: string, name: string, description?: string): Promise<TouristSpot> => {
    const rec = await updateSpotOp(id, name, description)
    return { id: rec.id, locationId: rec.locationId, name: rec.name, description: rec.description }
  }, [])

  const deleteSpot = useCallback(async (_locationId: string, id: string): Promise<void> => {
    await deleteSpotOp(id)
  }, [])

  const refetch = useCallback(async () => {}, [])

  return {
    locations, loading: raw === undefined, error: null, refetch,
    createLocation, updateLocation, deleteLocation,
    createSpot, updateSpot, deleteSpot,
  }
}
