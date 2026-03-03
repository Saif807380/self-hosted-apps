import { useState } from 'react'
import { Box, Flex, Text } from '@chakra-ui/react'
import type { TravelLocation } from '@/types/api'

interface LocationCardProps {
  location: TravelLocation
  onClick: (location: TravelLocation) => void
}

const COVER_GRADIENTS: [string, string][] = [
  ['#dbeafe', '#93c5fd'], // sky
  ['#d1fae5', '#6ee7b7'], // emerald
  ['#fef3c7', '#fcd34d'], // amber
  ['#fce7f3', '#f9a8d4'], // rose
  ['#ede9fe', '#c4b5fd'], // violet
  ['#e0f2fe', '#7dd3fc'], // light blue
  ['#dcfce7', '#86efac'], // green
  ['#fff7ed', '#fdba74'], // orange
]

function getGradient(name: string): [string, string] {
  const code = name.charCodeAt(0) || 65
  return COVER_GRADIENTS[code % COVER_GRADIENTS.length]
}

function formatDateShort(dateStr?: string): string {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  return d.toLocaleDateString('en-US', { month: 'short', year: 'numeric' })
}

export default function LocationCard({ location, onClick }: LocationCardProps) {
  const [hovered, setHovered] = useState(false)
  const [gradFrom, gradTo] = getGradient(location.city)
  const initial = location.city[0]?.toUpperCase() ?? '?'

  const dateRange = [formatDateShort(location.visitedFrom), formatDateShort(location.visitedTo)]
    .filter(Boolean).join(' – ')

  return (
    <Box
      bg="bg.surface"
      borderRadius="10px"
      overflow="hidden"
      border="1px solid"
      borderColor="border.default"
      cursor="pointer"
      onClick={() => onClick(location)}
      style={{
        transform: hovered ? 'translateY(-3px)' : 'translateY(0)',
        boxShadow: hovered
          ? '0 8px 24px rgba(0,0,0,0.10), 0 2px 6px rgba(0,0,0,0.05)'
          : '0 1px 3px rgba(0,0,0,0.04)',
        transition: 'transform 0.2s ease, box-shadow 0.2s ease',
      }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      {/* Color band */}
      <Box position="relative" pb="75%" overflow="hidden">
        <Box position="absolute" inset={0}>
          <Box
            w="full" h="full"
            style={{ background: `linear-gradient(150deg, ${gradFrom} 0%, ${gradTo} 100%)` }}
            display="flex" alignItems="center" justifyContent="center"
          >
            <Text
              fontFamily="heading" fontWeight="700" fontSize="4xl"
              style={{ color: 'rgba(0,0,0,0.2)', userSelect: 'none' }}
            >
              {initial}
            </Text>
          </Box>
        </Box>
      </Box>

      {/* Card body */}
      <Box p={3}>
        <Text
          fontFamily="heading" fontWeight="600" fontSize="md"
          color="text.primary" mb={0.5}
          style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
        >
          {location.city}
        </Text>

        <Text
          fontSize="sm" color="text.secondary" mb={2}
          style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
        >
          {location.country}
        </Text>

        {dateRange && (
          <Flex gap={1} wrap="wrap" mb={1.5}>
            <Box
              px={1.5} py="1px" borderRadius="4px"
              fontSize="10px" fontWeight="600"
              bg="bg.subtle" color="text.muted" letterSpacing="0.03em"
            >
              {dateRange}
            </Box>
          </Flex>
        )}

        {location.touristSpots?.length > 0 && (
          <Text fontSize="10px" color="text.muted">
            {location.touristSpots.length} spot{location.touristSpots.length !== 1 ? 's' : ''}
          </Text>
        )}
      </Box>
    </Box>
  )
}
