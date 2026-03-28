import { sync } from './sync-engine'

export type SyncStatus = 'idle' | 'syncing' | 'error' | 'offline'

export interface SyncState {
  status: SyncStatus
  lastSyncTime: string | null
  pendingChanges: number
  error: string | null
}

type Listener = (state: SyncState) => void

const SYNC_INTERVAL = 10_000
const MAX_BACKOFF = 60_000
const HEALTH_URL = '/api/health'
const HEALTH_TIMEOUT = 5_000

export class SyncScheduler {
  private state: SyncState = {
    status: 'idle',
    lastSyncTime: null,
    pendingChanges: 0,
    error: null,
  }
  private listeners = new Set<Listener>()
  private timer: ReturnType<typeof setInterval> | null = null
  private backoff = SYNC_INTERVAL
  private channel: BroadcastChannel | null = null
  private isLeader = false
  private destroyed = false

  constructor() {
    this.channel = new BroadcastChannel('trove-sync')
    this.channel.onmessage = (e) => {
      if (e.data?.type === 'leader-ping') {
        // Another tab is leader — stand down
        this.isLeader = false
      }
    }
    // Try to become leader
    this.electLeader()
  }

  private electLeader() {
    // Simple leader election: first tab to claim wins
    // Broadcast a claim; if no response after 200ms, become leader
    this.channel?.postMessage({ type: 'leader-ping' })
    setTimeout(() => {
      if (!this.destroyed) {
        this.isLeader = true
        console.debug('[sync-scheduler] became leader, starting')
        this.start()
      }
    }, 200)
  }

  private emit() {
    for (const fn of this.listeners) fn({ ...this.state })
  }

  subscribe(fn: Listener): () => void {
    this.listeners.add(fn)
    fn({ ...this.state })
    return () => this.listeners.delete(fn)
  }

  getState(): SyncState {
    return { ...this.state }
  }

  private start() {
    if (this.timer) return
    this.tick()
    this.timer = setInterval(() => this.tick(), this.backoff)
  }

  private async tick() {
    if (this.state.status === 'syncing') return
    if (!this.isLeader) return

    // Check connectivity
    const online = await this.checkOnline()
    console.debug('[sync-scheduler] tick: online =', online)
    if (!online) {
      this.state = { ...this.state, status: 'offline', error: null }
      this.emit()
      return
    }

    this.state = { ...this.state, status: 'syncing', error: null }
    this.emit()

    try {
      const { serverTime } = await sync()
      console.debug('[sync] ok', serverTime)
      this.state = {
        status: 'idle',
        lastSyncTime: serverTime,
        pendingChanges: 0,
        error: null,
      }
      this.backoff = SYNC_INTERVAL
      this.resetTimer()
    } catch (err) {
      console.warn('[sync] error', err)
      this.state = {
        ...this.state,
        status: 'error',
        error: err instanceof Error ? err.message : 'Sync failed',
      }
      this.backoff = Math.min(this.backoff * 2, MAX_BACKOFF)
      this.resetTimer()
    }

    this.emit()
  }

  private resetTimer() {
    if (this.timer) clearInterval(this.timer)
    this.timer = setInterval(() => this.tick(), this.backoff)
  }

  private async checkOnline(): Promise<boolean> {
    if (!navigator.onLine) return false
    try {
      const controller = new AbortController()
      const timer = setTimeout(() => controller.abort(), HEALTH_TIMEOUT)
      const res = await fetch(HEALTH_URL, {
        method: 'GET',
        cache: 'no-store',
        signal: controller.signal,
      })
      clearTimeout(timer)
      return res.ok
    } catch {
      return false
    }
  }

  async syncNow(): Promise<void> {
    this.backoff = SYNC_INTERVAL
    await this.tick()
  }

  destroy() {
    this.destroyed = true
    if (this.timer) clearInterval(this.timer)
    this.channel?.close()
    this.listeners.clear()
  }
}
