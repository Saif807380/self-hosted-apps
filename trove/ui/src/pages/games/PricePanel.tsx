import { useCallback, useEffect, useState, type ReactNode } from 'react'
import { Box, Button, Flex, Input, Spinner, Stack, Text } from '@chakra-ui/react'
import type { VideoGame } from '@/types/api'
import { gamingApi, type CatalogGame, type GameInsights, type Listing } from '@/services/gaming'
import { useDebounce } from '@/hooks/useDebounce'
import PriceChart from './PriceChart'
import { useIsDark } from '@/hooks/useIsDark'
import {
  buildChart, currentRun, formatDay, formatPrice, isPriced, listingLabel, platPricesUrl, seriesColor,
} from './prices'

// While the worker is still looking a game up, re-fetch on this interval,
// giving up after POLL_LIMIT tries (about two minutes).
const POLL_MS = 8000
const POLL_LIMIT = 15

/** Prices, price history and the IGDB match for a wishlisted game. */
export default function PricePanel({ game }: { game: VideoGame }) {
  const [insights, setInsights] = useState<GameInsights | null>(null)
  const [error, setError] = useState('')
  const [polls, setPolls] = useState(0)

  const load = useCallback(async () => {
    try {
      setInsights(await gamingApi.insights(game.id))
      setError('')
    } catch (e) {
      // A synced-but-not-yet-pushed game isn't on the server yet either.
      setError(e instanceof TypeError || !navigator.onLine ? 'offline' : 'unavailable')
    }
  }, [game.id])

  useEffect(() => { setPolls(0); load() }, [load])

  const pending = !insights || insights.matchStatus === 'none' ||
    (insights.matchStatus !== 'unresolved' && !insights.listingsCheckedAt)
  useEffect(() => {
    if (!pending || polls >= POLL_LIMIT) return
    const timer = setTimeout(() => { setPolls(n => n + 1); load() }, POLL_MS)
    return () => clearTimeout(timer)
  }, [pending, polls, load])

  const restartPolling = async () => { setPolls(0); await load() }

  if (error === 'offline' && !insights) {
    return <Note>Prices need a connection.</Note>
  }
  if (!insights) {
    return error ? <Note>Prices aren't available yet; they appear once the game has synced.</Note> : <Loading />
  }

  return (
    <Stack gap={4} w="full" pt={3} borderTopWidth="1px" borderColor="border.default">
      <MatchSection game={game} insights={insights} onLinked={restartPolling} />
      {insights.matchStatus !== 'unresolved' && (
        <PriceSection game={game} insights={insights} pending={pending && polls < POLL_LIMIT} onChanged={restartPolling} />
      )}
    </Stack>
  )
}

function PriceSection({ game, insights, pending, onChanged }: {
  game: VideoGame; insights: GameInsights; pending: boolean; onChanged: () => Promise<void>
}) {
  const listings = insights.listings ?? []
  const usdToInr = insights.usdToInr ?? 0
  const { series, rows } = buildChart(listings, usdToInr)
  const [refreshing, setRefreshing] = useState(false)

  const handleRefresh = async () => {
    setRefreshing(true)
    try {
      await gamingApi.refresh(game.id)
      await onChanged()
    } finally {
      setRefreshing(false)
    }
  }

  if (listings.length === 0) {
    return pending ? <Loading label="Finding store listings…" /> : <Note>No store listings found for this game.</Note>
  }

  return (
    <Stack gap={3}>
      <SectionTitle>Prices</SectionTitle>
      <Stack gap={2}>
        {listings.map(l => <ListingRow key={l.id} listing={l} usdToInr={usdToInr} gameId={game.id} onChanged={onChanged} />)}
      </Stack>
      {rows.length > 0 && <PriceChart series={series} rows={rows} targetRupees={game.targetPriceInr} />}
      <Flex justify="space-between" align="center">
        <Text fontSize="xs" color="text.muted">
          {pending ? 'Checking prices…' : priceAge(listings)}
        </Text>
        <Button size="xs" variant="ghost" onClick={handleRefresh} loading={refreshing}>Refresh</Button>
      </Flex>
    </Stack>
  )
}

function ListingRow({ listing, usdToInr, gameId, onChanged }: {
  listing: Listing; usdToInr: number; gameId: string; onChanged: () => Promise<void>
}) {
  const isDark = useIsDark()
  const run = currentRun(listing)
  const onSale = run && Number(run.finalMinor) < Number(run.baseMinor)
  const isLowest = run && listing.lowestMinor && run.finalMinor === listing.lowestMinor && (listing.history?.length ?? 0) > 1

  return (
    <Box p={2.5} borderRadius="8px" bg="bg.subtle">
      <Flex justify="space-between" align="center" gap={2}>
        <Flex align="center" gap={2} minW={0}>
          {isPriced(listing) && (
            // Same colour as this store's line in the chart below.
            <Box w="8px" h="8px" borderRadius="full" flexShrink={0} style={{ background: seriesColor(listing, isDark) }} />
          )}
          <Text fontSize="sm" fontWeight="600" color="text.primary" truncate>{listingLabel(listing)}</Text>
        </Flex>
        <Flex gap={1.5} flexShrink={0}>
          {listing.store === 'psn' && <LinkButton href={platPricesUrl(listing.title)}>PlatPrices</LinkButton>}
          <LinkButton href={listing.url}>{listing.store === 'psn' ? 'PS Store' : 'Open'}</LinkButton>
        </Flex>
      </Flex>

      {isPriced(listing) && (run ? (
        <Box mt={1}>
          <Flex gap={2} align="baseline" wrap="wrap">
            <Text fontSize="sm" color="text.primary" fontWeight="600">
              {formatPrice(run.finalMinor, listing.currency, usdToInr)}
            </Text>
            {onSale && (
              <>
                <Text fontSize="xs" color="text.muted" textDecoration="line-through">
                  {formatPrice(run.baseMinor, listing.currency, 0)}
                </Text>
                <Text fontSize="xs" fontWeight="700" color="text.primary">−{run.discountPct}%</Text>
              </>
            )}
          </Flex>
          <Text fontSize="xs" color="text.secondary">
            {[
              isLowest && 'Lowest seen',
              !isLowest && listing.lowestMinor && `Lowest ${formatPrice(listing.lowestMinor, listing.currency, 0)}`,
              onSale && run.saleEndsAt && `Sale ends ${formatDay(run.saleEndsAt)}`,
            ].filter(Boolean).join(' · ')}
          </Text>
        </Box>
      ) : (
        <Text mt={1} fontSize="xs" color="text.muted">Not purchasable right now</Text>
      ))}

      {listing.store === 'psn' && <PSNLinkEditor listing={listing} gameId={gameId} onChanged={onChanged} />}
    </Box>
  )
}

function PSNLinkEditor({ listing, gameId, onChanged }: {
  listing: Listing; gameId: string; onChanged: () => Promise<void>
}) {
  const [editing, setEditing] = useState(false)
  const [url, setUrl] = useState('')
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)

  const save = async (value: string) => {
    setSaving(true)
    try {
      await gamingApi.setPsnLink(gameId, value)
      setEditing(false)
      setUrl('')
      setError('')
      await onChanged()
    } catch {
      setError('That isn\'t a PS Store game page link.')
    } finally {
      setSaving(false)
    }
  }

  if (!editing) {
    return (
      <Text mt={1} fontSize="xs" color="text.secondary">
        {listing.isSearchLink ? 'PS Store opens a search. ' : 'PS Store opens the game page. '}
        PlatPrices shows its prices and history.{' '}
        <button type="button" onClick={() => setEditing(true)} style={linkStyle}>
          {listing.isSearchLink ? 'Paste game page link' : 'Change'}
        </button>
        {!listing.isSearchLink && (
          <> · <button type="button" onClick={() => save('')} style={linkStyle}>Use search</button></>
        )}
      </Text>
    )
  }
  return (
    <Box mt={2}>
      <Flex gap={2}>
        <Input
          size="xs" value={url} onChange={e => setUrl(e.target.value)} autoFocus
          placeholder="https://store.playstation.com/…/product/…" bg="bg.surface" borderRadius="6px"
        />
        <Button size="xs" onClick={() => save(url.trim())} disabled={!url.trim()} loading={saving}>Save</Button>
        <Button size="xs" variant="ghost" onClick={() => setEditing(false)}>Cancel</Button>
      </Flex>
      {error && <Text fontSize="xs" color="red.500" mt={1}>{error}</Text>}
    </Box>
  )
}

function MatchSection({ game, insights, onLinked }: {
  game: VideoGame; insights: GameInsights; onLinked: () => Promise<void>
}) {
  const [picking, setPicking] = useState(false)
  const unresolved = insights.matchStatus === 'unresolved'

  if (insights.matchStatus === 'none') return <Loading label="Looking the game up…" />

  if (!unresolved && !picking) {
    const m = insights.match
    return (
      <Text fontSize="xs" color="text.muted">
        Matched to {m?.title}{m?.year ? ` (${m.year})` : ''}.{' '}
        <button type="button" onClick={() => setPicking(true)} style={linkStyle}>Wrong game?</button>
      </Text>
    )
  }
  return (
    <LinkPicker
      game={game}
      candidates={unresolved ? insights.candidates ?? [] : []}
      onCancel={unresolved ? undefined : () => setPicking(false)}
      onLinked={async () => { setPicking(false); await onLinked() }}
    />
  )
}

function LinkPicker({ game, candidates, onCancel, onLinked }: {
  game: VideoGame; candidates: CatalogGame[]; onCancel?: () => void; onLinked: () => Promise<void>
}) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<CatalogGame[] | null>(null)
  const [linking, setLinking] = useState('')
  const debounced = useDebounce(query, 350)

  useEffect(() => {
    if (debounced.trim().length < 2) { setResults(null); return }
    let cancelled = false
    gamingApi.search(debounced.trim()).then(r => { if (!cancelled) setResults(r) }).catch(() => {})
    return () => { cancelled = true }
  }, [debounced])

  const pick = async (g: CatalogGame) => {
    setLinking(g.igdbId)
    try {
      await gamingApi.link(game.id, g.igdbId)
      await onLinked()
    } finally {
      setLinking('')
    }
  }

  const options = results ?? candidates
  return (
    <Stack gap={2}>
      <SectionTitle>Which game is this?</SectionTitle>
      {options.map(g => (
        <Button
          key={g.igdbId} variant="outline" size="sm" justifyContent="flex-start" h="auto" py={1.5}
          onClick={() => pick(g)} loading={linking === g.igdbId} borderRadius="8px"
        >
          {g.coverUrl && <img src={g.coverUrl} alt="" style={{ width: 24, height: 32, objectFit: 'cover', borderRadius: 3 }} />}
          <Text as="span" truncate>{g.title}</Text>
          {g.year ? <Text as="span" color="text.muted" fontSize="xs">{g.year}</Text> : null}
        </Button>
      ))}
      {results?.length === 0 && <Text fontSize="xs" color="text.muted">No games found.</Text>}
      <Input
        size="sm" value={query} onChange={e => setQuery(e.target.value)}
        placeholder={options.length ? 'Not listed? Search IGDB…' : 'Search IGDB…'} bg="bg.surface" borderRadius="7px"
      />
      {onCancel && <Button size="xs" variant="ghost" alignSelf="flex-start" onClick={onCancel}>Cancel</Button>}
    </Stack>
  )
}

function LinkButton({ href, children }: { href: string; children: ReactNode }) {
  return (
    <Button
      asChild size="xs" variant="outline" borderRadius="6px"
      bg="bg.surface" borderColor="text.muted" _hover={{ borderColor: 'text.secondary' }}
    >
      <a href={href} target="_blank" rel="noreferrer">{children}</a>
    </Button>
  )
}

function priceAge(listings: Listing[]): string {
  const last = listings.flatMap(l => l.history ?? []).map(r => Date.parse(r.until)).reduce((a, b) => Math.max(a, b), 0)
  return last ? `Checked ${formatDay(last)}, every 6 hours` : 'Not checked yet'
}

const linkStyle = {
  background: 'none', border: 'none', padding: 0, cursor: 'pointer',
  color: 'var(--chakra-colors-accent)', fontSize: 'inherit', textDecoration: 'underline',
} as const

function SectionTitle({ children }: { children: ReactNode }) {
  return (
    <Text fontSize="xs" color="text.muted" fontWeight="500" textTransform="uppercase" letterSpacing="0.04em">
      {children}
    </Text>
  )
}

function Note({ children }: { children: ReactNode }) {
  return <Text fontSize="xs" color="text.muted" textAlign="center" pt={2}>{children}</Text>
}

function Loading({ label = 'Loading prices…' }: { label?: string }) {
  return (
    <Flex align="center" justify="center" gap={2} pt={2}>
      <Spinner size="xs" color="accent" />
      <Text fontSize="xs" color="text.muted">{label}</Text>
    </Flex>
  )
}
