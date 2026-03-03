import { useState } from 'react'
import { Box, Flex, Text } from '@chakra-ui/react'
import type { Book } from '@/types/api'
import StarRating from '@/components/StarRating'
import TagBadge from '@/components/TagBadge'

interface BookListItemProps {
  book: Book
  onClick: (book: Book) => void
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

export default function BookListItem({ book, onClick }: BookListItemProps) {
  const [hovered, setHovered] = useState(false)
  const [gradFrom, gradTo] = getCoverGradient(book.title)
  const initial = book.title[0]?.toUpperCase() ?? '?'

  return (
    <Box
      bg="bg.surface"
      borderRadius="10px"
      border="1px solid"
      borderColor="border.default"
      cursor="pointer"
      px={3}
      py={2.5}
      onClick={() => onClick(book)}
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
        <Box
          w="108px"
          h="150px"
          borderRadius="5px"
          overflow="hidden"
          flexShrink={0}
        >
          {book.coverImage ? (
            <img
              src={book.coverImage}
              alt={book.title}
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

        {/* Title + author */}
        <Box flex={1} minW={0}>
          <Text
            fontFamily="heading" fontWeight="600" fontSize="sm" color="text.primary"
            style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
          >
            {book.title}
          </Text>
          <Text
            fontSize="xs" color="text.secondary"
            style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
          >
            {book.author}
          </Text>
        </Box>

        {/* Tags */}
        {book.tags.length > 0 && (
          <Flex gap={1} wrap="nowrap" flexShrink={0}>
            {book.tags.slice(0, 3).map(tag => (
              <TagBadge key={tag.id} name={tag.name} size="sm" />
            ))}
            {book.tags.length > 3 && (
              <Text fontSize="10px" color="text.muted" alignSelf="center">
                +{book.tags.length - 3}
              </Text>
            )}
          </Flex>
        )}

        {/* Years */}
        {book.yearsRead?.length > 0 && (
          <Flex gap={1} flexShrink={0}>
            {book.yearsRead.map(year => (
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
        {book.rating != null && book.rating > 0 && (
          <Box flexShrink={0}>
            <StarRating value={book.rating} readOnly size="sm" />
          </Box>
        )}
      </Flex>
    </Box>
  )
}
