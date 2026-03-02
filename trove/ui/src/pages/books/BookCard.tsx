import { useState } from 'react'
import { Box, Flex, Text } from '@chakra-ui/react'
import type { Book } from '@/types/api'
import StarRating from '@/components/StarRating'
import TagBadge from '@/components/TagBadge'

interface BookCardProps {
  book: Book
  onClick: (book: Book) => void
}

// Muted, editorial cover gradients — deterministic from title
const COVER_GRADIENTS: [string, string][] = [
  ['#f5f0e8', '#c9a97a'], // warm parchment
  ['#e8eef5', '#7a9cc9'], // slate blue
  ['#f0e8f5', '#a87ac9'], // dusty plum
  ['#e8f5ef', '#7ac9a0'], // sage
  ['#f5ebe8', '#c98078'], // terracotta
  ['#f5f5e8', '#c9c578'], // olive
  ['#eef5f0', '#78c9a0'], // mint
  ['#f5e8f0', '#c978a0'], // rose
]

function getCoverGradient(title: string): [string, string] {
  const code = title.charCodeAt(0) || 65
  return COVER_GRADIENTS[code % COVER_GRADIENTS.length]
}

export default function BookCard({ book, onClick }: BookCardProps) {
  const [hovered, setHovered] = useState(false)
  const [gradFrom, gradTo] = getCoverGradient(book.title)
  const initial = book.title[0]?.toUpperCase() ?? '?'
  return (
    <Box
      bg="bg.surface"
      borderRadius="10px"
      overflow="hidden"
      border="1px solid"
      borderColor="border.default"
      cursor="pointer"
      onClick={() => onClick(book)}
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
          {book.coverImage ? (
            <img
              src={book.coverImage}
              alt={book.title}
              style={{ width: '100%', height: '100%', objectFit: 'cover', display: 'block' }}
            />
          ) : (
            <Box
              w="full"
              h="full"
              style={{
                background: `linear-gradient(150deg, ${gradFrom} 0%, ${gradTo} 100%)`,
              }}
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
        {book.rating != null && book.rating > 0 && (
          <Box mb={1.5}>
            <StarRating value={book.rating} readOnly size="sm" />
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
          {book.title}
        </Text>

        <Text
          fontSize="sm"
          color="text.secondary"
          mb={2}
          style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
        >
          {book.author}
        </Text>

        {book.yearsRead?.length > 0 && (
          <Flex gap={1} wrap="wrap" mb={1.5}>
            {book.yearsRead.map(year => (
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

        {book.tags.length > 0 && (
          <Flex gap={1} wrap="wrap">
            {book.tags.slice(0, 2).map(tag => (
              <TagBadge key={tag.id} name={tag.name} size="sm" />
            ))}
            {book.tags.length > 2 && (
              <Text fontSize="10px" color="text.muted" alignSelf="center">
                +{book.tags.length - 2}
              </Text>
            )}
          </Flex>
        )}
      </Box>
    </Box>
  )
}
