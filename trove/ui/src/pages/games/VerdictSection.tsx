import { Badge, Box, Flex, Stack, Text } from '@chakra-ui/react'
import type { DeviceRating, DeviceRatingName, GameInsights } from '@/services/gaming'
import { PLATFORM_LABELS } from './lifecycle'
import { Loading, SectionTitle } from './panel'

// The label always shows, so the colour is never the only cue.
const RATING_PALETTE: Record<DeviceRatingName, string> = {
  great: 'green',
  good: 'blue',
  playable: 'orange',
  avoid: 'red',
}

/** Which of your devices to play the game on, and why. */
export default function VerdictSection({ insights, pending }: { insights: GameInsights; pending: boolean }) {
  const verdict = insights.verdict
  if (!verdict) {
    return pending ? <Loading label="Working out where to play it…" /> : null
  }
  const devices = verdict.devices ?? []
  const unavailable = verdict.unavailable ?? []
  const sources = verdict.sources ?? []

  return (
    <Stack gap={2}>
      <SectionTitle>Where to play</SectionTitle>
      {verdict.summary && <Text fontSize="sm" color="text.primary">{verdict.summary}</Text>}
      {devices.map(d => <DeviceRow key={d.device} rating={d} />)}
      {unavailable.length > 0 && (
        <Text fontSize="xs" color="text.muted">
          Not on {unavailable.map(d => PLATFORM_LABELS[d]).join(' or ')}
        </Text>
      )}
      {sources.length > 0 && (
        <Box as="details" fontSize="xs" color="text.muted">
          <Box as="summary" cursor="pointer">Sources ({sources.length})</Box>
          <Stack gap={1} mt={1} pl={3}>
            {sources.map(s => (
              <a key={s.url} href={s.url} target="_blank" rel="noreferrer" style={{ textDecoration: 'underline' }}>
                <Text as="span" truncate display="block">{s.label || s.url}</Text>
              </a>
            ))}
          </Stack>
        </Box>
      )}
      {!verdict.complete && (
        <Text fontSize="xs" color="text.muted">
          Some sources couldn't be reached; this is redone once they can.
        </Text>
      )}
    </Stack>
  )
}

function DeviceRow({ rating }: { rating: DeviceRating }) {
  const reasons = rating.reasons ?? []
  return (
    <Box p={2.5} borderRadius="8px" bg="bg.subtle">
      <Flex justify="space-between" align="center" gap={2}>
        <Text fontSize="sm" fontWeight="600" color="text.primary">{PLATFORM_LABELS[rating.device]}</Text>
        <Badge colorPalette={RATING_PALETTE[rating.rating]} variant="subtle" textTransform="capitalize">
          {rating.rating}
        </Badge>
      </Flex>
      {rating.expect && rating.expect !== 'unknown' && (
        <Text fontSize="xs" color="text.secondary" mt={0.5}>{rating.expect}</Text>
      )}
      {reasons.length > 0 && (
        <Stack as="ul" gap={0.5} mt={1} pl={4} fontSize="xs" color="text.secondary" listStyleType="disc">
          {reasons.map(r => <li key={r}>{r}</li>)}
        </Stack>
      )}
    </Box>
  )
}
