import { rpc } from './client'

const SVC = 'trove.v1.gaming.GamingService'

// Mirrors trove/v1/gaming/gaming.proto. Connect's JSON drops empty fields and
// sends int64 as strings, hence the optional fields and string amounts.

export interface CatalogGame {
  igdbId: string
  title: string
  year?: number
  coverUrl?: string
  platforms?: string[]
}

export interface PriceRun {
  from: string
  until: string
  baseMinor: string
  finalMinor: string
  discountPct?: number
  saleEndsAt?: string
}

export type StoreName = 'steam' | 'eshop' | 'psn'

export interface Listing {
  id: string
  store: StoreName
  edition?: string
  title: string
  url: string
  currency: string
  isSearchLink?: boolean
  history?: PriceRun[]
  lowestMinor?: string
}

export type MatchStatus = 'none' | 'picked' | 'auto' | 'unresolved'

export type Device = 'pc' | 'ps5' | 'switch2'
export type DeviceRatingName = 'great' | 'good' | 'playable' | 'avoid'

export interface DeviceRating {
  device: Device
  rating: DeviceRatingName
  expect?: string
  reasons?: string[]
}

/** Which of your devices to play a game on. */
export interface DeviceVerdict {
  summary?: string
  devices?: DeviceRating[] // best first
  unavailable?: Device[]
  sources?: { label?: string; url: string }[]
  complete?: boolean // false: some evidence was missing; it's redone later
  createdAt: string
}

export interface GameInsights {
  matchStatus: MatchStatus
  match?: CatalogGame
  candidates?: CatalogGame[]
  listings?: Listing[]
  usdToInr?: number
  listingsCheckedAt?: string
  verdict?: DeviceVerdict
  verdictsEnabled?: boolean // false: no verdict is coming (no API key)
}

export const gamingApi = {
  insights: (videoGameId: string) =>
    rpc<GameInsights>(SVC, 'GetGameInsights', { videoGameId }),

  search: (query: string) =>
    rpc<{ games?: CatalogGame[] }>(SVC, 'SearchCatalog', { query }).then(r => r.games ?? []),

  link: (videoGameId: string, igdbId: string) =>
    rpc<Record<string, never>>(SVC, 'LinkGame', { videoGameId, igdbId }),

  refresh: (videoGameId: string) =>
    rpc<Record<string, never>>(SVC, 'RefreshGame', { videoGameId }),

  setPsnLink: (videoGameId: string, url: string) =>
    rpc<Record<string, never>>(SVC, 'SetPsnLink', { videoGameId, url }),

  pushConfig: () =>
    rpc<{ enabled?: boolean; publicKey?: string }>(SVC, 'GetPushConfig', {}),

  subscribePush: (sub: { endpoint: string; p256dh: string; auth: string }) =>
    rpc<Record<string, never>>(SVC, 'SubscribePush', sub),

  unsubscribePush: (endpoint: string) =>
    rpc<Record<string, never>>(SVC, 'UnsubscribePush', { endpoint }),

  sendTestPush: () =>
    rpc<{ delivered?: number }>(SVC, 'SendTestPush', {}).then(r => r.delivered ?? 0),
}
