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

/**
 * The colour of the chart line a listing is drawn as: its own, or the
 * shared line's when it has the same prices as another store.
 */
export function lineColor(l: Listing, series: ChartSeries[], isDark: boolean): string {
  const line = series.find(s => s.listingIds.includes(l.id))
  return seriesColor(line?.listing ?? l, isDark)
}

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
  listing: Listing // gives the line its colour
  listingIds: string[] // every listing drawn as this line (see shareIdenticalLines)
}

export type ChartRow = { t: number } & Record<string, number | null>

/**
 * Builds one row per moment any price changed or was last seen, with every
 * series' price (in rupees) at that moment, so a shared tooltip can show all
 * stores at once. A series is null before its first observation.
 */
export function buildChart(listings: Listing[], usdToInr: number): { series: ChartSeries[]; rows: ChartRow[] } {
  const priced = listings.filter(l => isPriced(l) && l.history?.length)
  const series = priced.map(l => ({ key: l.id, label: listingLabel(l), listing: l, listingIds: [l.id] }))

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
  return { series: shareIdenticalLines(series, rows), rows }
}

/**
 * Stores priced the same all along (a game's Switch and Switch 2 editions
 * often cost the same) would draw one line exactly over the other and hide
 * it, so they share one line labelled with both: "eShop · Switch & Switch 2".
 * They split again once their prices differ.
 */
function shareIdenticalLines(series: ChartSeries[], rows: ChartRow[]): ChartSeries[] {
  const groups: ChartSeries[][] = []
  for (const s of series) {
    const same = groups.find(g => rows.every(r => r[g[0].key] === r[s.key]))
    if (same) same.push(s)
    else groups.push([s])
  }
  return groups.map(g => (g.length === 1 ? g[0] : {
    ...g[0],
    label: sharedLabel(g.map(s => s.listing)),
    listingIds: g.flatMap(s => s.listingIds),
  }))
}

function sharedLabel(listings: Listing[]): string {
  if (listings.every(l => l.store === 'eshop' && l.edition)) {
    return `eShop · ${listings.map(l => l.edition).join(' & ')}`
  }
  return listings.map(listingLabel).join(' & ')
}
