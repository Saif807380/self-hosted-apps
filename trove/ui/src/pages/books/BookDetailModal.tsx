import { Dialog, Portal, Button, Box, Flex, Stack, Text, CloseButton } from '@chakra-ui/react'
import type { Book } from '@/types/api'
import StarRating from '@/components/StarRating'
import TagBadge from '@/components/TagBadge'

interface BookDetailModalProps {
  book: Book | null
  isOpen: boolean
  onClose: () => void
  onEdit: (book: Book) => void
  onDelete: (book: Book) => void
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

export default function BookDetailModal({
  book, isOpen, onClose, onEdit, onDelete,
}: BookDetailModalProps) {
  if (!book) return null

  const [gradFrom, gradTo] = getCoverGradient(book.title)
  const initial = book.title[0]?.toUpperCase() ?? '?'

  const handleEdit = () => { onClose(); onEdit(book) }
  const handleDelete = () => { onClose(); onDelete(book) }

  return (
    <Dialog.Root
      open={isOpen}
      onOpenChange={({ open }: { open: boolean }) => !open && onClose()}
      scrollBehavior="inside"
    >
      <Portal>
        <Dialog.Backdrop />
        <Dialog.Positioner>
          {/* Cover: 120px wide, 3:4 ratio → 160px tall. 90% (144px) in blurred area, 10% (16px) below */}
          <Dialog.Content maxW="380px" mx={4} position="relative">
            {/* Blurred background section (200px). Cover top = 200 - 144 = 56px */}
            <Box h="300px" overflow="hidden" position="relative" flexShrink={0}>
              {book.coverImage ? (
                <img
                  src={book.coverImage}
                  alt=""
                  aria-hidden
                  style={{
                    position: 'absolute', inset: 0,
                    width: '100%', height: '100%',
                    objectFit: 'cover',
                    filter: 'blur(8px)',
                    transform: 'scale(1.08)',
                    opacity: 0.75,
                  }}
                />
              ) : (
                <Box
                  position="absolute" inset={0}
                  style={{ background: `linear-gradient(150deg, ${gradFrom} 0%, ${gradTo} 100%)` }}
                />
              )}
              <Box position="absolute" inset={0} bg="rgba(0,0,0,0.3)" />
              <CloseButton
                size="sm"
                onClick={onClose}
                position="absolute"
                top={2}
                right={2}
                style={{ background: 'rgba(0,0,0,0.45)', color: 'white', borderRadius: '50%' }}
              />
            </Box>

            {/* Cover card: absolutely positioned, straddling blurred area and body */}
            <Box
              position="absolute"
              top="50px"
              left="50%"
              w="220px"
              borderRadius="8px"
              overflow="hidden"
              zIndex={2}
              style={{
                transform: 'translateX(-50%)',
                boxShadow: '0 8px 32px rgba(0,0,0,0.45)',
              }}
            >
              <Box position="relative" pb="133%">
                <Box position="absolute" inset={0}>
                  {book.coverImage ? (
                    <img
                      src={book.coverImage}
                      alt={book.title}
                      style={{ width: '100%', height: '100%', objectFit: 'fill', display: 'block' }}
                    />
                  ) : (
                    <Box
                      w="full" h="full"
                      style={{ background: `linear-gradient(150deg, ${gradFrom} 0%, ${gradTo} 100%)` }}
                      display="flex" alignItems="center" justifyContent="center"
                    >
                      <Text
                        fontFamily="heading" fontWeight="700" fontSize="5xl"
                        style={{ color: 'rgba(0,0,0,0.25)', userSelect: 'none' }}
                      >
                        {initial}
                      </Text>
                    </Box>
                  )}
                </Box>
              </Box>
            </Box>

            {/* pt="24px" = 16px cover overlap + 8px breathing room */}
            <Dialog.Body pt="72px" pb={4}>
              <Stack gap={3} align="center">
                <Text
                  fontFamily="heading" fontWeight="700" fontSize="xl"
                  color="text.primary" lineHeight="1.3" letterSpacing="-0.01em"
                  textAlign="center"
                >
                  {book.title}
                </Text>

                <Text fontSize="sm" color="text.secondary">{book.author}</Text>

                {book.rating != null && book.rating > 0 && (
                  <StarRating value={book.rating} readOnly size="md" />
                )}

                {book.yearsRead?.length > 0 && (
                  <Box>
                    <Text fontSize="xs" color="text.muted" mb={1} fontWeight="500">Read in</Text>
                    <Flex gap={1.5} wrap="wrap">
                      {book.yearsRead.map(year => (
                        <Box
                          key={year}
                          px={2} py="2px" borderRadius="5px"
                          fontSize="xs" fontWeight="600"
                          bg="bg.subtle" color="text.secondary" letterSpacing="0.02em"
                        >
                          {year}
                        </Box>
                      ))}
                    </Flex>
                  </Box>
                )}

                {book.tags.length > 0 && (
                  <Box mx="auto">
                    <Text fontSize="xs" color="text.muted" mb={1} fontWeight="500" textAlign="center">Tags</Text>
                    <Flex gap={1.5} wrap="wrap">
                      {book.tags.map(tag => (
                        <TagBadge key={tag.id} name={tag.name} size="md" />
                      ))}
                    </Flex>
                  </Box>
                )}

                {book.review && (
                  <Box pt={3} borderTopWidth="1px" borderColor="border.default">
                    <Text
                      fontSize="xs" color="text.muted" mb={2} fontWeight="500"
                      textTransform="uppercase" letterSpacing="0.04em"
                    >
                      Review
                    </Text>
                    <Text fontSize="sm" color="text.primary" lineHeight="1.7" style={{ whiteSpace: 'pre-wrap' }}>
                      {book.review}
                    </Text>
                  </Box>
                )}
              </Stack>
            </Dialog.Body>

            <Dialog.Footer borderTopWidth="1px" borderColor="border.default" pt={3} gap={2}>
              <Button variant="outline" size="sm" borderRadius="7px" onClick={handleEdit}>Edit</Button>
              <Button colorPalette="red" variant="outline" size="sm" borderRadius="7px" onClick={handleDelete}>Delete</Button>
            </Dialog.Footer>
          </Dialog.Content>
        </Dialog.Positioner>
      </Portal>
    </Dialog.Root>
  )
}
