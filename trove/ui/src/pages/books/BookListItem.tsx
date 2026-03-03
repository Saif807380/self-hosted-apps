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
      overflow="hidden"
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
      <Flex align="stretch">
        {/* Thumbnail — fixed width, fills item height */}
        <Box w="90px" flexShrink={0} position="relative">
          {book.coverImage ? (
            <img
              src={book.coverImage}
              alt={book.title}
              style={{ width: '100%', height: '100%', objectFit: 'cover', display: 'block' }}
            />
          ) : (
            <Box
              w="full" h="full" minH="120px"
              display="flex" alignItems="center" justifyContent="center"
              style={{ background: `linear-gradient(150deg, ${gradFrom} 0%, ${gradTo} 100%)` }}
            >
              <Text
                fontFamily="heading" fontWeight="700" fontSize="2xl"
                style={{ color: 'rgba(0,0,0,0.25)', userSelect: 'none' }}
              >
                {initial}
              </Text>
            </Box>
          )}
        </Box>

        {/* Content — all stacked vertically, no right-side items */}
        <Box px={3} py={2.5} minW={0}>
          <Text
            fontFamily="heading" fontWeight="600" fontSize="sm" color="text.primary"
            mb={0.5}
            style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
          >
            {book.title}
          </Text>

          <Text
            fontSize="xs" color="text.secondary" mb={1.5}
            style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
          >
            {book.author}
          </Text>

          <Flex gap={2} align="center" wrap="wrap" mb={1}>
            {book.yearsRead?.length > 0 && (
              <Flex gap={1}>
                {book.yearsRead.map(year => (
                  <Box
                    key={year}
                    px={1.5} py="1px" borderRadius="4px"
                    fontSize="10px" fontWeight="600"
                    bg="bg.subtle" color="text.muted" letterSpacing="0.03em"
                  >
                    {year}
                  </Box>
                ))}
              </Flex>
            )}
            {book.rating != null && book.rating > 0 && (
              <StarRating value={book.rating} readOnly size="sm" />
            )}
          </Flex>

          {book.tags.length > 0 && (
            <Flex gap={1} wrap="wrap">
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
        </Box>
      </Flex>
    </Box>
  )
}
