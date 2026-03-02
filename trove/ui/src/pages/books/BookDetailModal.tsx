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

// Same gradient logic as BookCard
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

export default function BookDetailModal({
  book, isOpen, onClose, onEdit, onDelete,
}: BookDetailModalProps) {
  if (!book) return null

  const [gradFrom, gradTo] = getCoverGradient(book.title)
  const initial = book.title[0]?.toUpperCase() ?? '?'

  const handleEdit = () => {
    onClose()
    onEdit(book)
  }

  const handleDelete = () => {
    onClose()
    onDelete(book)
  }

  return (
    <Dialog.Root
      open={isOpen}
      onOpenChange={({ open }: { open: boolean }) => !open && onClose()}
      scrollBehavior="inside"
    >
      <Portal>
        <Dialog.Backdrop />
        <Dialog.Positioner>
          <Dialog.Content maxW="560px" mx={4}>
            <Dialog.Header borderBottomWidth="1px" borderColor="border.default" pb={3}>
              <Dialog.Title fontFamily="heading" fontWeight="600" fontSize="lg">
                Book Details
              </Dialog.Title>
              <CloseButton
                size="sm"
                onClick={onClose}
                position="absolute"
                top={3}
                right={3}
              />
            </Dialog.Header>

            <Dialog.Body py={5}>
              <Flex gap={5} align="flex-start" direction={{ base: 'column', sm: 'row' }}>
                {/* Cover image */}
                <Box
                  w={{ base: '120px', sm: '140px' }}
                  flexShrink={0}
                  borderRadius="8px"
                  overflow="hidden"
                  border="1px solid"
                  borderColor="border.default"
                >
                  <Box position="relative" pb="150%">
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
                            fontSize="3xl"
                            style={{ color: 'rgba(0,0,0,0.25)', userSelect: 'none' }}
                          >
                            {initial}
                          </Text>
                        </Box>
                      )}
                    </Box>
                  </Box>
                </Box>

                {/* Details */}
                <Stack gap={3} flex={1} minW={0}>
                  {/* Title */}
                  <Text
                    fontFamily="heading"
                    fontWeight="700"
                    fontSize="xl"
                    color="text.primary"
                    lineHeight="1.3"
                    letterSpacing="-0.01em"
                  >
                    {book.title}
                  </Text>

                  {/* Author */}
                  <Text fontSize="sm" color="text.secondary">
                    {book.author}
                  </Text>

                  {/* Rating */}
                  {book.rating != null && book.rating > 0 && (
                    <StarRating value={book.rating} readOnly size="md" />
                  )}

                  {/* Years read */}
                  {book.yearsRead?.length > 0 && (
                    <Box>
                      <Text fontSize="xs" color="text.muted" mb={1} fontWeight="500">
                        Read in
                      </Text>
                      <Flex gap={1.5} wrap="wrap">
                        {book.yearsRead.map(year => (
                          <Box
                            key={year}
                            px={2}
                            py="2px"
                            borderRadius="5px"
                            fontSize="xs"
                            fontWeight="600"
                            bg="bg.subtle"
                            color="text.secondary"
                            letterSpacing="0.02em"
                          >
                            {year}
                          </Box>
                        ))}
                      </Flex>
                    </Box>
                  )}

                  {/* Tags */}
                  {book.tags.length > 0 && (
                    <Box>
                      <Text fontSize="xs" color="text.muted" mb={1} fontWeight="500">
                        Tags
                      </Text>
                      <Flex gap={1.5} wrap="wrap">
                        {book.tags.map(tag => (
                          <TagBadge key={tag.id} name={tag.name} size="md" />
                        ))}
                      </Flex>
                    </Box>
                  )}
                </Stack>
              </Flex>

              {/* Review */}
              {book.review && (
                <Box mt={5} pt={4} borderTopWidth="1px" borderColor="border.default">
                  <Text fontSize="xs" color="text.muted" mb={2} fontWeight="500" textTransform="uppercase" letterSpacing="0.04em">
                    Review
                  </Text>
                  <Text
                    fontSize="sm"
                    color="text.primary"
                    lineHeight="1.7"
                    style={{ whiteSpace: 'pre-wrap' }}
                  >
                    {book.review}
                  </Text>
                </Box>
              )}
            </Dialog.Body>

            <Dialog.Footer borderTopWidth="1px" borderColor="border.default" pt={3} gap={2}>
              <Button
                variant="outline"
                size="sm"
                borderRadius="7px"
                onClick={handleEdit}
              >
                Edit
              </Button>
              <Button
                colorPalette="red"
                variant="outline"
                size="sm"
                borderRadius="7px"
                onClick={handleDelete}
              >
                Delete
              </Button>
            </Dialog.Footer>
          </Dialog.Content>
        </Dialog.Positioner>
      </Portal>
    </Dialog.Root>
  )
}
