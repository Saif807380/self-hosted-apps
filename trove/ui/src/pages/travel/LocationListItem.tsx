import { useState } from 'react'
import { Box, Flex, Text } from '@chakra-ui/react'
import type { TravelLocation } from '@/types/api'

interface LocationListItemProps {
  location: TravelLocation
  onClick: (location: TravelLocation) => void
}

const COVER_GRADIENTS: [string, string][] = [
  ['#dbeafe', '#93c5fd'],
  ['#d1fae5', '#6ee7b7'],
  ['#fef3c7', '#fcd34d'],
  ['#fce7f3', '#f9a8d4'],
  ['#ede9fe', '#c4b5fd'],
  ['#e0f2fe', '#7dd3fc'],
  ['#dcfce7', '#86efac'],
  ['#fff7ed', '#fdba74'],
]

function getGradient(name: string): [string, string] {
  return COVER_GRADIENTS[(name.charCodeAt(0) || 65) % COVER_GRADIENTS.length]
}

function formatDateShort(dateStr?: string): string {
  if (!dateStr) return ''
  return new Date(dateStr).toLocaleDateString('en-US', { month: 'short', year: 'numeric' })
}

export default function LocationListItem({ location, onClick }: LocationListItemProps) {
  const [hovered, setHovered] = useState(false)
  const [gradFrom, gradTo] = getGradient(location.city)
  const initial = location.city[0]?.toUpperCase() ?? '?'

  const dateRange = [formatDateShort(location.visitedFrom), formatDateShort(location.visitedTo)]
    .filter(Boolean).join(' – ')

  return (
    <Box
      bg="bg.surface"
      borderRadius="10px"
      border="1px solid"
      borderColor="border.default"
      cursor="pointer"
      px={3}
      py={2.5}
      onClick={() => onClick(location)}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        boxShadow: hovered
          ? '0 4px 16px rgba(0,0,0,0.08)'
          : '0 1px 3px rgba(0,0,0,0.04)',
        transition: 'box-shadow 0.2s ease',
      }}
    >
      <Flex align="center" gap={3}>
        {/* Color swatch */}
        <Box
          w="42px" h="42px"
          borderRadius="8px"
          overflow="hidden"
          flexShrink={0}
          display="flex" alignItems="center" justifyContent="center"
          style={{ background: `linear-gradient(150deg, ${gradFrom} 0%, ${gradTo} 100%)` }}
        >
          <Text
            fontFamily="heading" fontWeight="700" fontSize="md"
            style={{ color: 'rgba(0,0,0,0.2)', userSelect: 'none' }}
          >
            {initial}
          </Text>
        </Box>

        {/* City + country */}
        <Box flex={1} minW={0}>
          <Text
            fontFamily="heading" fontWeight="600" fontSize="sm" color="text.primary"
            style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
          >
            {location.city}
          </Text>
          <Text
            fontSize="xs" color="text.secondary"
            style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
          >
            {location.country}
          </Text>
        </Box>

        {/* Date range */}
        {dateRange && (
          <Box
            px={1.5} py="1px"
            borderRadius="4px"
            fontSize="10px" fontWeight="600"
            bg="bg.subtle" color="text.muted"
            letterSpacing="0.03em"
            flexShrink={0}
          >
            {dateRange}
          </Box>
        )}

        {/* Spot count */}
        {location.touristSpots?.length > 0 && (
          <Text fontSize="xs" color="text.muted" flexShrink={0}>
            {location.touristSpots.length} spot{location.touristSpots.length !== 1 ? 's' : ''}
          </Text>
        )}
      </Flex>
    </Box>
  )
}
