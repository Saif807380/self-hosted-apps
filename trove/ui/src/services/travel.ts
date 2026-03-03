import type { TravelLocation, TravelLocationFormData, TouristSpot } from '@/types/api'
import { rpc } from './client'

const SVC = 'trove.v1.travel.TravelService'

export const travelApi = {
  list: (params: { search?: string } = {}): Promise<TravelLocation[]> => {
    const body: Record<string, unknown> = {}
    if (params.search) body.search = params.search
    return rpc<{ locations?: TravelLocation[] }>(SVC, 'ListLocations', body).then(r => r.locations ?? [])
  },

  create: (data: TravelLocationFormData): Promise<TravelLocation> =>
    rpc<{ location: TravelLocation }>(SVC, 'CreateLocation', data).then(r => r.location),

  update: (id: string, data: TravelLocationFormData): Promise<TravelLocation> =>
    rpc<{ location: TravelLocation }>(SVC, 'UpdateLocation', { id, ...data }).then(r => r.location),

  delete: (id: string): Promise<void> =>
    rpc<Record<string, never>>(SVC, 'DeleteLocation', { id }).then(() => undefined),

  createSpot: (locationId: string, name: string, description?: string): Promise<TouristSpot> =>
    rpc<{ spot: TouristSpot }>(SVC, 'CreateSpot', { locationId, name, description }).then(r => r.spot),

  updateSpot: (id: string, name: string, description?: string): Promise<TouristSpot> =>
    rpc<{ spot: TouristSpot }>(SVC, 'UpdateSpot', { id, name, description }).then(r => r.spot),

  deleteSpot: (id: string): Promise<void> =>
    rpc<Record<string, never>>(SVC, 'DeleteSpot', { id }).then(() => undefined),
}
