import { useState } from 'react'
import { Box, Flex, Text } from '@chakra-ui/react'
import type { VideoGame } from '@/types/api'
import StarRating from '@/components/StarRating'

interface GameListItemProps {
  game: VideoGame
  onClick: (game: VideoGame) => void
}

const COVER_GRADIENTS: [string, string][] = [
  ['#f5f0e8', '#c9a97a'],
  ['#e8eef5', '#7a9cc9'],
  ['#f0e8f5', '#a87ac9'],
  ['#e8f5ef', '#7ac9a0'],
  ['#f5ebe8', '#c98078'],
  ['#f5f5e8', '#c9c578'],
  ['#eef5f0', '#78c9a0'],
  ['#f5e8f0', '#c978a0'],
]

function getCoverGradient(title: string): [string, string] {
  return COVER_GRADIENTS[(title.charCodeAt(0) || 65) % COVER_GRADIENTS.length]
}

export default function GameListItem({ game, onClick }: GameListItemProps) {
  const [hovered, setHovered] = useState(false)
  const [gradFrom, gradTo] = getCoverGradient(game.title)
  const initial = game.title[0]?.toUpperCase() ?? '?'

  return (
    <Box
      bg="bg.surface"
      borderRadius="10px"
      border="1px solid"
      borderColor="border.default"
      cursor="pointer"
      px={3}
      py={2.5}
      onClick={() => onClick(game)}
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
        {/* Thumbnail */}
        <Box w="108px" h="150px" borderRadius="5px" overflow="hidden" flexShrink={0}>
          {game.coverImage ? (
            <img
              src={game.coverImage}
              alt={game.title}
              style={{ width: '100%', height: '100%', objectFit: 'cover', display: 'block' }}
            />
          ) : (
            <Box
              w="full" h="full"
              display="flex" alignItems="center" justifyContent="center"
              style={{ background: `linear-gradient(150deg, ${gradFrom} 0%, ${gradTo} 100%)` }}
            >
              <Text
                fontFamily="heading" fontWeight="700" fontSize="sm"
                style={{ color: 'rgba(0,0,0,0.25)', userSelect: 'none' }}
              >
                {initial}
              </Text>
            </Box>
          )}
        </Box>

        {/* Title + studio */}
        <Box flex={1} minW={0}>
          <Text
            fontFamily="heading" fontWeight="600" fontSize="sm" color="text.primary"
            style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
          >
            {game.title}
          </Text>
          {game.studio && (
            <Text
              fontSize="xs" color="text.secondary"
              style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
            >
              {game.studio}
            </Text>
          )}
        </Box>

        {/* Years played */}
        {game.yearsPlayed?.length > 0 && (
          <Flex gap={1} flexShrink={0}>
            {game.yearsPlayed.map(year => (
              <Box
                key={year}
                px={1.5} py="1px"
                borderRadius="4px"
                fontSize="10px" fontWeight="600"
                bg="bg.subtle" color="text.muted" letterSpacing="0.03em"
              >
                {year}
              </Box>
            ))}
          </Flex>
        )}

        {/* Rating */}
        {game.rating != null && game.rating > 0 && (
          <Box flexShrink={0}>
            <StarRating value={game.rating} readOnly size="sm" />
          </Box>
        )}
      </Flex>
    </Box>
  )
}
