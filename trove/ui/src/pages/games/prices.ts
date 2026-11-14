import type { Listing, PriceRun } from '@/services/gaming'

/** A listing that has prices (Steam, eShop), as opposed to a PS Store link. */
export const isPriced = (l: Listing) => l.store !== 'psn'

export function listingLabel(l: Listing): string {
  switch (l.store) {
    case 'steam': return 'Steam'
    case 'eshop': return l.edition ? `eShop · ${l.edition}` : 'eShop'
    case 'psn': return 'PS Store'
  }
}

// One fixed colour per store edition, never assigned by order, so a store
// keeps its colour whichever others are listed. Categorical slots 1-3,
// validated for colour-blind separation against bg.surface in both modes.
const SERIES_COLORS = {
  steam: { light: '#2a78d6', dark: '#3987e5' },
  switch2: { light: '#eb6834', dark: '#d95926' },
  switch: { light: '#1baf7a', dark: '#199e70' },
}

/**
 * A PlatPrices search for the game: PS Store prices and history in any
 * browser, without the extension. PlatPrices remembers the region itself.
 */
export const platPricesUrl = (title: string) =>
  `https://platprices.com/search?${new URLSearchParams({ q: title })}`

/** The chart colour for a listing, as hex (Recharts can't use CSS vars). */
export function seriesColor(l: Listing, isDark: boolean): string {
  const key = l.store === 'eshop' ? (l.edition === 'Switch' ? 'switch' : 'switch2') : 'steam'
  return SERIES_COLORS[key][isDark ? 'dark' : 'light']
}

export const currentRun = (l: Listing): PriceRun | undefined => l.history?.[l.history.length - 1]

/** Converts minor units of a listing's currency to whole rupees. */
export function toRupees(minor: string | number, currency: string, usdToInr: number): number | undefined {
  const units = Number(minor) / 100
  if (currency === 'INR') return units
  if (currency === 'USD' && usdToInr > 0) return units * usdToInr
  return undefined
}

const inr = new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR', maximumFractionDigits: 0 })
const usd = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' })

export const formatRupees = (rupees: number) => inr.format(rupees)

/** "₹1,300", or "$29.99 (≈₹2,827)" for a dollar price. */
export function formatPrice(minor: string | number, currency: string, usdToInr: number): string {
  const units = Number(minor) / 100
  if (currency === 'INR') return inr.format(units)
  const native = currency === 'USD' ? usd.format(units) : `${units.toFixed(2)} ${currency}`
  const rupees = toRupees(minor, currency, usdToInr)
  return rupees === undefined ? native : `${native} (≈${inr.format(rupees)})`
}

export const formatDay = (iso: string | number) =>
  new Date(iso).toLocaleDateString('en-IN', { day: 'numeric', month: 'short' })

export interface ChartSeries {
  key: string
  label: string
  listing: Listing
}

export type ChartRow = { t: number } & Record<string, number | null>

/**
 * Builds one row per moment any price changed or was last seen, with every
 * series' price (in rupees) at that moment, so a shared tooltip can show all
 * stores at once. A series is null before its first observation.
 */
export function buildChart(listings: Listing[], usdToInr: number): { series: ChartSeries[]; rows: ChartRow[] } {
  const priced = listings.filter(l => isPriced(l) && l.history?.length)
  const series = priced.map(l => ({ key: l.id, label: listingLabel(l), listing: l }))

  const times = new Set<number>()
  for (const l of priced) {
    for (const run of l.history!) {
      times.add(Date.parse(run.from))
      times.add(Date.parse(run.until))
    }
  }

  const rows = [...times].sort((a, b) => a - b).map(t => {
    const row: ChartRow = { t }
    for (const l of priced) {
      // The price in force at t is the latest run that started by then.
      const run = [...l.history!].reverse().find(r => Date.parse(r.from) <= t)
      row[l.id] = run ? (toRupees(run.finalMinor, l.currency, usdToInr) ?? null) : null
    }
    return row
  })
  return { series, rows }
}
