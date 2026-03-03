import { useState, useEffect, useCallback } from "react";
import { travelApi } from "@/services/travel";
import type {
  TravelLocation,
  TravelLocationFormData,
  TouristSpot,
} from "@/types/api";

export function useTravel() {
  const [locations, setLocations] = useState<TravelLocation[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refetch = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await travelApi.list();
      setLocations(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load locations");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refetch();
  }, [refetch]);

  const createLocation = useCallback(
    async (data: TravelLocationFormData): Promise<TravelLocation> => {
      const location = await travelApi.create(data);
      await refetch();
      return location;
    },
    [refetch],
  );

  const updateLocation = useCallback(
    async (
      id: string,
      data: TravelLocationFormData,
    ): Promise<TravelLocation> => {
      const location = await travelApi.update(id, data);
      await refetch();
      return location;
    },
    [refetch],
  );

  const deleteLocation = useCallback(async (id: string): Promise<void> => {
    await travelApi.delete(id);
    setLocations((prev) => prev.filter((l) => l.id !== id));
  }, []);

  const createSpot = useCallback(
    async (
      locationId: string,
      name: string,
      description?: string,
    ): Promise<TouristSpot> => {
      const spot = await travelApi.createSpot(locationId, name, description);
      setLocations((prev) =>
        prev.map((l) =>
          l.id === locationId
            ? { ...l, touristSpots: [...l?.touristSpots, spot] }
            : l,
        ),
      );
      return spot;
    },
    [],
  );

  const updateSpot = useCallback(
    async (
      locationId: string,
      id: string,
      name: string,
      description?: string,
    ): Promise<TouristSpot> => {
      const spot = await travelApi.updateSpot(id, name, description);
      setLocations((prev) =>
        prev.map((l) =>
          l.id === locationId
            ? {
                ...l,
                touristSpots: l.touristSpots.map((s) =>
                  s.id === id ? spot : s,
                ),
              }
            : l,
        ),
      );
      return spot;
    },
    [],
  );

  const deleteSpot = useCallback(
    async (locationId: string, id: string): Promise<void> => {
      await travelApi.deleteSpot(id);
      setLocations((prev) =>
        prev.map((l) =>
          l.id === locationId
            ? { ...l, touristSpots: l.touristSpots.filter((s) => s.id !== id) }
            : l,
        ),
      );
    },
    [],
  );

  return {
    locations,
    loading,
    error,
    refetch,
    createLocation,
    updateLocation,
    deleteLocation,
    createSpot,
    updateSpot,
    deleteSpot,
  };
}
