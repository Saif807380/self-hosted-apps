import { useCallback, useEffect, useState } from 'react'
import { gamingApi, type Deal } from '@/services/gaming'
import { bestDeals } from '@/pages/games/prices'

/** loading → ready, or offline / unavailable when the server can't be reached. */
export type DealsStatus = 'loading' | 'ready' | 'offline' | 'unavailable'

/**
 * Wishlisted games on sale right now, best deal per game. Prices live on the
 * server rather than in the offline store, so this needs a connection; it
 * reloads when the device comes back online.
 */
export function useDeals() {
  const [deals, setDeals] = useState<Map<string, Deal>>(new Map())
  const [status, setStatus] = useState<DealsStatus>('loading')

  const load = useCallback(async () => {
    try {
      const res = await gamingApi.deals()
      setDeals(bestDeals(res.deals ?? [], res.usdToInr ?? 0))
      setStatus('ready')
    } catch (e) {
      setStatus(e instanceof TypeError || !navigator.onLine ? 'offline' : 'unavailable')
    }
  }, [])

  useEffect(() => {
    load()
    window.addEventListener('online', load)
    return () => window.removeEventListener('online', load)
  }, [load])

  return { deals, status }
}
