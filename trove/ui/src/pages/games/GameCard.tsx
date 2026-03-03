import { useState } from 'react'
import { Box, Flex, Text } from '@chakra-ui/react'
import type { VideoGame } from '@/types/api'
import StarRating from '@/components/StarRating'

interface GameCardProps {
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
  const code = title.charCodeAt(0) || 65
  return COVER_GRADIENTS[code % COVER_GRADIENTS.length]
}

export default function GameCard({ game, onClick }: GameCardProps) {
  const [hovered, setHovered] = useState(false)
  const [gradFrom, gradTo] = getCoverGradient(game.title)
  const initial = game.title[0]?.toUpperCase() ?? '?'

  return (
    <Box
      bg="bg.surface"
      borderRadius="10px"
      overflow="hidden"
      border="1px solid"
      borderColor="border.default"
      cursor="pointer"
      onClick={() => onClick(game)}
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
      {/* Cover */}
      <Box position="relative" pb="150%" overflow="hidden">
        <Box position="absolute" inset={0}>
          {game.coverImage ? (
            <img
              src={game.coverImage}
              alt={game.title}
              style={{ width: '100%', height: '100%', objectFit: 'cover', display: 'block' }}
            />
          ) : (
            <Box
              w="full"
              h="full"
              style={{ background: `linear-gradient(150deg, ${gradFrom} 0%, ${gradTo} 100%)` }}
              display="flex"
              alignItems="center"
              justifyContent="center"
            >
              <Text
                fontFamily="heading"
                fontWeight="700"
                fontSize="4xl"
                style={{ color: 'rgba(0,0,0,0.25)', userSelect: 'none' }}
              >
                {initial}
              </Text>
            </Box>
          )}
        </Box>
      </Box>

      {/* Card body */}
      <Box p={3}>
        {game.rating != null && game.rating > 0 && (
          <Box mb={1.5}>
            <StarRating value={game.rating} readOnly size="sm" />
          </Box>
        )}

        <Text
          fontFamily="heading"
          fontWeight="600"
          fontSize="md"
          color="text.primary"
          mb={0.5}
          style={{
            display: '-webkit-box',
            WebkitLineClamp: 2,
            WebkitBoxOrient: 'vertical',
            overflow: 'hidden',
            lineHeight: '1.35',
          }}
        >
          {game.title}
        </Text>

        {game.studio && (
          <Text
            fontSize="sm"
            color="text.secondary"
            mb={2}
            style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
          >
            {game.studio}
          </Text>
        )}

        {game.yearsPlayed?.length > 0 && (
          <Flex gap={1} wrap="wrap">
            {game.yearsPlayed.map(year => (
              <Box
                key={year}
                px={1.5}
                py="1px"
                borderRadius="4px"
                fontSize="10px"
                fontWeight="600"
                bg="bg.subtle"
                color="text.muted"
                letterSpacing="0.03em"
              >
                {year}
              </Box>
            ))}
          </Flex>
        )}
      </Box>
    </Box>
  )
}
