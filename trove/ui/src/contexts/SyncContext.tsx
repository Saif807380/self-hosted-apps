import { createContext, useEffect, useState, useCallback, useRef } from 'react'
import { SyncScheduler, type SyncState } from '@/lib/sync-scheduler'
import { pull } from '@/lib/sync-engine'
import { db } from '@/lib/db'

export interface SyncContextValue extends SyncState {
  syncNow: () => Promise<void>
  bootstrapping: boolean
}

export const SyncContext = createContext<SyncContextValue>({
  status: 'idle',
  lastSyncTime: null,
  pendingChanges: 0,
  error: null,
  syncNow: async () => {},
  bootstrapping: true,
})

export function SyncProvider({ children }: { children: React.ReactNode }) {
  const [state, setState] = useState<SyncState>({
    status: 'idle', lastSyncTime: null, pendingChanges: 0, error: null,
  })
  const [bootstrapping, setBootstrapping] = useState(true)
  const schedulerRef = useRef<SyncScheduler | null>(null)

  // Initial hydration: full pull on first load (skip if offline)
  useEffect(() => {
    let cancelled = false
    ;(async () => {
      try {
        const meta = await db.syncMeta.get('lastSync')
        console.debug('[bootstrap] lastSync =', meta?.value ?? '(none)')
        if (!meta && navigator.onLine) {
          console.debug('[bootstrap] first load — doing full pull')
          await pull()
          console.debug('[bootstrap] full pull complete')
        } else if (!meta) {
          console.debug('[bootstrap] first load but offline — skipping pull')
        }
      } catch (err) {
        console.error('[bootstrap] Initial sync failed:', err)
      } finally {
        if (!cancelled) setBootstrapping(false)
      }
    })()
    return () => { cancelled = true }
  }, [])

  // Start scheduler after bootstrap
  useEffect(() => {
    if (bootstrapping) return
    const scheduler = new SyncScheduler()
    schedulerRef.current = scheduler
    const unsub = scheduler.subscribe(setState)
    return () => {
      unsub()
      scheduler.destroy()
      schedulerRef.current = null
    }
  }, [bootstrapping])

  const syncNow = useCallback(async () => {
    await schedulerRef.current?.syncNow()
  }, [])

  return (
    <SyncContext.Provider value={{ ...state, syncNow, bootstrapping }}>
      {children}
    </SyncContext.Provider>
  )
}
