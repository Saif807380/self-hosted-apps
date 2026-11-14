import type { GamePlatform, GameStatus } from '@/types/api'

export const STATUS_TABS: { status: GameStatus; label: string; icon: string }[] = [
  { status: 'played', label: 'Played', icon: '✅' },
  { status: 'wishlist', label: 'Wishlist', icon: '⭐' },
  { status: 'owned', label: 'Owned', icon: '📦' },
  { status: 'dropped', label: 'Dropped', icon: '💤' },
]

export const STATUS_LABELS = Object.fromEntries(
  STATUS_TABS.map(t => [t.status, t.label]),
) as Record<GameStatus, string>

export const PLATFORM_LABELS: Record<GamePlatform, string> = {
  pc: 'PC',
  ps5: 'PS5',
  switch2: 'Switch 2',
}

export const isGameStatus = (v: string | null): v is GameStatus =>
  STATUS_TABS.some(t => t.status === v)

export const formatInr = (rupees: number) => `₹${rupees.toLocaleString('en-IN')}`
