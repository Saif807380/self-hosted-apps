import { useState } from 'react'
import type { KeyboardEvent } from 'react'
import {
  Dialog, Portal, Button, Box, Flex, Stack, Text, CloseButton, Input, Spinner,
} from '@chakra-ui/react'
import type { TravelLocation, TouristSpot } from '@/types/api'

interface LocationDetailModalProps {
  location: TravelLocation | null
  isOpen: boolean
  onClose: () => void
  onEdit: (location: TravelLocation) => void
  onDelete: (location: TravelLocation) => void
  onAddSpot: (locationId: string, name: string, description?: string) => Promise<TouristSpot>
  onDeleteSpot: (locationId: string, spotId: string) => Promise<void>
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
  const code = name.charCodeAt(0) || 65
  return COVER_GRADIENTS[code % COVER_GRADIENTS.length]
}

function formatDate(dateStr?: string): string {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
}

export default function LocationDetailModal({
  location, isOpen, onClose, onEdit, onDelete, onAddSpot, onDeleteSpot,
}: LocationDetailModalProps) {
  const [newSpotName, setNewSpotName] = useState('')
  const [addingSpot, setAddingSpot] = useState(false)
  const [deletingSpotId, setDeletingSpotId] = useState<string | null>(null)

  if (!location) return null

  const [gradFrom, gradTo] = getGradient(location.city)
  const initial = location.city[0]?.toUpperCase() ?? '?'

  const handleEdit = () => { onClose(); onEdit(location) }
  const handleDelete = () => { onClose(); onDelete(location) }

  const handleAddSpotKeyDown = async (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key !== 'Enter') return
    e.preventDefault()
    const name = newSpotName.trim()
    if (!name || addingSpot) return
    setAddingSpot(true)
    try {
      await onAddSpot(location.id, name)
      setNewSpotName('')
    } finally {
      setAddingSpot(false)
    }
  }

  const handleDeleteSpot = async (spotId: string) => {
    setDeletingSpotId(spotId)
    try {
      await onDeleteSpot(location.id, spotId)
    } finally {
      setDeletingSpotId(null)
    }
  }

  const dateFrom = formatDate(location.visitedFrom)
  const dateTo = formatDate(location.visitedTo)
  const dateRange = dateFrom && dateTo
    ? `${dateFrom} – ${dateTo}`
    : dateFrom || dateTo || ''

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
                Location Details
              </Dialog.Title>
              <CloseButton size="sm" onClick={onClose} position="absolute" top={3} right={3} />
            </Dialog.Header>

            <Dialog.Body py={5}>
              <Flex gap={5} align="flex-start" direction={{ base: 'column', sm: 'row' }}>
                {/* Color swatch */}
                <Box
                  w={{ base: '80px', sm: '100px' }}
                  h={{ base: '60px', sm: '75px' }}
                  flexShrink={0}
                  borderRadius="8px"
                  overflow="hidden"
                  border="1px solid"
                  borderColor="border.default"
                  style={{ background: `linear-gradient(150deg, ${gradFrom} 0%, ${gradTo} 100%)` }}
                  display="flex" alignItems="center" justifyContent="center"
                >
                  <Text
                    fontFamily="heading" fontWeight="700" fontSize="3xl"
                    style={{ color: 'rgba(0,0,0,0.2)', userSelect: 'none' }}
                  >
                    {initial}
                  </Text>
                </Box>

                <Stack gap={2} flex={1} minW={0}>
                  <Text
                    fontFamily="heading" fontWeight="700" fontSize="xl"
                    color="text.primary" lineHeight="1.3" letterSpacing="-0.01em"
                  >
                    {location.city}
                  </Text>
                  <Text fontSize="sm" color="text.secondary">{location.country}</Text>
                  {dateRange && (
                    <Text fontSize="xs" color="text.muted">{dateRange}</Text>
                  )}
                  {location.photoCollectionUrl && (
                    <Box mt={1}>
                      <a
                        href={location.photoCollectionUrl}
                        target="_blank"
                        rel="noopener noreferrer"
                        style={{
                          fontSize: '0.8rem',
                          color: 'var(--chakra-colors-accent)',
                          textDecoration: 'none',
                          display: 'inline-flex',
                          alignItems: 'center',
                          gap: '4px',
                        }}
                      >
                        📷 View photo collection ↗
                      </a>
                    </Box>
                  )}
                </Stack>
              </Flex>

              {/* Tourist spots */}
              <Box mt={5} pt={4} borderTopWidth="1px" borderColor="border.default">
                <Text fontSize="xs" color="text.muted" mb={3} fontWeight="500" textTransform="uppercase" letterSpacing="0.04em">
                  Tourist Spots
                </Text>

                {location.touristSpots?.length > 0 && (
                  <Stack gap={0} mb={3}>
                    {location.touristSpots.map(spot => (
                      <Flex
                        key={spot.id}
                        align="center"
                        justify="space-between"
                        gap={3}
                        py={2}
                        borderBottomWidth="1px"
                        borderColor="border.default"
                      >
                        <Box flex={1} minW={0}>
                          <Text fontSize="sm" color="text.primary" fontWeight="500">
                            {spot.name}
                          </Text>
                          {spot.description && (
                            <Text fontSize="xs" color="text.muted" mt={0.5}>
                              {spot.description}
                            </Text>
                          )}
                        </Box>
                        <button
                          type="button"
                          onClick={() => handleDeleteSpot(spot.id)}
                          disabled={deletingSpotId === spot.id}
                          aria-label={`Remove ${spot.name}`}
                          style={{
                            background: 'none', border: 'none',
                            cursor: deletingSpotId === spot.id ? 'default' : 'pointer',
                            color: 'var(--chakra-colors-text-muted)',
                            opacity: deletingSpotId === spot.id ? 0.4 : 0.6,
                            fontSize: '1rem', lineHeight: 1, padding: '2px 4px',
                            flexShrink: 0,
                          }}
                        >
                          {deletingSpotId === spot.id ? '…' : '×'}
                        </button>
                      </Flex>
                    ))}
                  </Stack>
                )}

                <Flex gap={2} align="center">
                  <Input
                    size="sm"
                    value={newSpotName}
                    onChange={e => setNewSpotName(e.target.value)}
                    onKeyDown={handleAddSpotKeyDown}
                    placeholder="Add a spot — press Enter"
                    bg="bg.surface"
                    borderRadius="7px"
                    disabled={addingSpot}
                  />
                  {addingSpot && <Spinner size="sm" color="accent" flexShrink={0} />}
                </Flex>
              </Box>
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
