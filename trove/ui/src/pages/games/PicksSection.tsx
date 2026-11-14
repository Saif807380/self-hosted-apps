import { useCallback, useEffect, useState } from 'react'
import { Box, Button, Flex, SimpleGrid, Spinner, Stack, Text } from '@chakra-ui/react'
import { gamingApi, type GamePick, type PickAction } from '@/services/gaming'
import { useSync } from '@/hooks/useSync'
import { toaster } from '@/lib/toaster'
import DealTag from './DealTag'
import { LinkButton, SectionTitle } from './panel'

const DONE: Record<PickAction, string> = {
  wishlist: 'Added to your wishlist',
  own: 'Added to Owned',
  dismiss: "Got it, fewer like that",
}

/**
 * This week's picks: games on sale you'd probably like, chosen every
 * Sunday. Adding one makes it a Trove game; "Not for me" teaches the picks.
 */
export default function PicksSection() {
  const [picks, setPicks] = useState<GamePick[] | null>(null)
  const [week, setWeek] = useState('')
  const [status, setStatus] = useState<'loading' | 'ready' | 'offline' | 'unavailable'>('loading')
  const [busy, setBusy] = useState('')
  const { syncNow } = useSync()

  const load = useCallback(async () => {
    try {
      const res = await gamingApi.picks()
      setPicks(res.picks ?? [])
      setWeek(res.week ?? '')
      setStatus('ready')
    } catch (e) {
      setStatus(e instanceof TypeError || !navigator.onLine ? 'offline' : 'unavailable')
    }
  }, [])
  useEffect(() => { load() }, [load])

  const handleAction = async (pick: GamePick, action: PickAction) => {
    setBusy(pick.id + action)
    try {
      await gamingApi.updatePick(pick.id, action)
      setPicks(prev => prev?.filter(p => p.id !== pick.id) ?? null)
      toaster.create({ title: DONE[action], description: action === 'dismiss' ? undefined : pick.title, type: 'success', duration: 3000 })
      if (action !== 'dismiss') syncNow().catch(() => {}) // bring the new game in now
    } catch {
      toaster.create({ title: "Couldn't update the pick", description: 'Try again in a moment.', type: 'error', duration: 4000 })
      load()
    } finally {
      setBusy('')
    }
  }

  return (
    <Stack gap={3}>
      <SectionTitle>Picks for you</SectionTitle>
      {status === 'loading' && <Flex justify="center" py={8}><Spinner color="accent" /></Flex>}
      {status === 'offline' && <Note>Picks need a connection.</Note>}
      {status === 'unavailable' && <Note>Picks aren't available right now.</Note>}
      {status === 'ready' && picks?.length === 0 && (
        <Note>
          {week
            ? "You've been through this week's picks. New ones arrive on Sunday morning."
            : 'Your first picks are on their way. They are made every Sunday morning from games on sale.'}
        </Note>
      )}
      {status === 'ready' && !!picks?.length && (
        <SimpleGrid columns={{ base: 1, md: 2, xl: 3 }} gap={4}>
          {picks.map(p => <PickCard key={p.id} pick={p} busy={busy} onAction={handleAction} />)}
        </SimpleGrid>
      )}
    </Stack>
  )
}

function PickCard({ pick, busy, onAction }: {
  pick: GamePick; busy: string; onAction: (pick: GamePick, action: PickAction) => void
}) {
  const device = pick.store === 'steam' ? 'PC' : 'Switch 2'
  return (
    <Flex bg="bg.surface" border="1px solid" borderColor="border.default" borderRadius="10px" overflow="hidden">
      <Box w="96px" flexShrink={0} bg="bg.subtle">
        {pick.coverUrl && (
          <img src={pick.coverUrl} alt={pick.title} loading="lazy" style={{ width: '100%', height: '100%', objectFit: 'cover', display: 'block' }} />
        )}
      </Box>
      <Stack gap={1.5} p={3} minW={0} flex={1}>
        <Box>
          <Text fontFamily="heading" fontWeight="600" fontSize="sm" color="text.primary" lineClamp={2}>{pick.title}</Text>
          <Text fontSize="xs" color="text.muted">{[pick.year, `for ${device}`].filter(Boolean).join(' · ')}</Text>
        </Box>
        <Text fontSize="xs" color="text.secondary">{pick.pitch}</Text>
        <DealTag deal={pick} size="xs" />
        <Flex gap={1.5} wrap="wrap" mt="auto">
          <Button size="xs" borderRadius="6px" onClick={() => onAction(pick, 'wishlist')} loading={busy === pick.id + 'wishlist'} disabled={!!busy}>
            + Wishlist
          </Button>
          <Button size="xs" variant="outline" borderRadius="6px" onClick={() => onAction(pick, 'own')} loading={busy === pick.id + 'own'} disabled={!!busy}>
            Own it
          </Button>
          <Button size="xs" variant="ghost" borderRadius="6px" onClick={() => onAction(pick, 'dismiss')} loading={busy === pick.id + 'dismiss'} disabled={!!busy}>
            Not for me
          </Button>
          <LinkButton href={pick.url}>Store</LinkButton>
        </Flex>
      </Stack>
    </Flex>
  )
}

function Note({ children }: { children: string }) {
  return <Text fontSize="sm" color="text.muted" py={4}>{children}</Text>
}
