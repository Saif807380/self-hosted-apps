import { useState, useEffect } from 'react'
import type { FormEvent } from 'react'
import {
  Dialog, Portal, Button, Input, Stack, Flex, Box, Text, CloseButton,
} from '@chakra-ui/react'
import type { TravelLocation, TravelLocationFormData } from '@/types/api'

interface LocationFormProps {
  isOpen: boolean
  onClose: () => void
  onSubmit: (data: TravelLocationFormData) => Promise<void>
  editLocation?: TravelLocation | null
}

interface FormState {
  city: string
  country: string
  visitedFrom: string
  visitedTo: string
  photoCollectionUrl: string
}

function initState(location?: TravelLocation | null): FormState {
  return {
    city: location?.city ?? '',
    country: location?.country ?? '',
    visitedFrom: location?.visitedFrom?.split('T')[0] ?? '',
    visitedTo: location?.visitedTo?.split('T')[0] ?? '',
    photoCollectionUrl: location?.photoCollectionUrl ?? '',
  }
}

const EMPTY_ERRORS = { city: '', country: '' }

export default function LocationForm({ isOpen, onClose, onSubmit, editLocation }: LocationFormProps) {
  const [state, setState] = useState<FormState>(() => initState(editLocation))
  const [errors, setErrors] = useState(EMPTY_ERRORS)
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (isOpen) {
      setState(initState(editLocation))
      setErrors(EMPTY_ERRORS)
      setSubmitting(false)
    }
  }, [isOpen, editLocation])

  const set = <K extends keyof FormState>(key: K, value: FormState[K]) =>
    setState(prev => ({ ...prev, [key]: value }))

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    if (submitting) return

    const newErrors = {
      city: state.city.trim() ? '' : 'City is required',
      country: state.country.trim() ? '' : 'Country is required',
    }
    if (Object.values(newErrors).some(Boolean)) {
      setErrors(newErrors)
      return
    }

    setSubmitting(true)
    try {
      await onSubmit({
        city: state.city.trim(),
        country: state.country.trim(),
        visitedFrom: state.visitedFrom || undefined,
        visitedTo: state.visitedTo || undefined,
        photoCollectionUrl: state.photoCollectionUrl.trim() || undefined,
      })
    } finally {
      setSubmitting(false)
    }
  }

  const isEdit = !!editLocation

  return (
    <Dialog.Root
      open={isOpen}
      onOpenChange={({ open }: { open: boolean }) => !open && !submitting && onClose()}
      scrollBehavior="inside"
    >
      <Portal>
        <Dialog.Backdrop />
        <Dialog.Positioner>
          <Dialog.Content maxW="480px" mx={4}>
            <form onSubmit={handleSubmit} style={{ display: 'contents' }}>
              <Dialog.Header borderBottomWidth="1px" borderColor="border.default" pb={3}>
                <Dialog.Title fontFamily="heading" fontWeight="600" fontSize="lg">
                  {isEdit ? 'Edit location' : 'Add a location'}
                </Dialog.Title>
                <CloseButton
                  size="sm"
                  disabled={submitting}
                  onClick={() => !submitting && onClose()}
                  position="absolute"
                  top={3}
                  right={3}
                />
              </Dialog.Header>

              <Dialog.Body py={5}>
                <Stack gap={4}>
                  <Box>
                    <Text fontSize="xs" color="text.secondary" mb={1} fontWeight="500">
                      City <span style={{ color: 'var(--chakra-colors-accent)' }}>*</span>
                    </Text>
                    <Input
                      size="sm"
                      value={state.city}
                      onChange={e => { set('city', e.target.value); if (errors.city) setErrors(prev => ({ ...prev, city: '' })) }}
                      placeholder="e.g. Tokyo"
                      bg="bg.surface"
                      borderRadius="7px"
                      borderColor={errors.city ? 'red.400' : undefined}
                    />
                    {errors.city && <Text fontSize="xs" color="red.500" mt={1}>{errors.city}</Text>}
                  </Box>

                  <Box>
                    <Text fontSize="xs" color="text.secondary" mb={1} fontWeight="500">
                      Country <span style={{ color: 'var(--chakra-colors-accent)' }}>*</span>
                    </Text>
                    <Input
                      size="sm"
                      value={state.country}
                      onChange={e => { set('country', e.target.value); if (errors.country) setErrors(prev => ({ ...prev, country: '' })) }}
                      placeholder="e.g. Japan"
                      bg="bg.surface"
                      borderRadius="7px"
                      borderColor={errors.country ? 'red.400' : undefined}
                    />
                    {errors.country && <Text fontSize="xs" color="red.500" mt={1}>{errors.country}</Text>}
                  </Box>

                  <Box>
                    <Text fontSize="xs" color="text.secondary" mb={1} fontWeight="500">
                      Visit dates (optional)
                    </Text>
                    <Flex gap={3}>
                      <Box flex={1}>
                        <Text fontSize="xs" color="text.muted" mb={1}>From</Text>
                        <Input
                          size="sm"
                          type="date"
                          value={state.visitedFrom}
                          onChange={e => set('visitedFrom', e.target.value)}
                          bg="bg.surface"
                          borderRadius="7px"
                        />
                      </Box>
                      <Box flex={1}>
                        <Text fontSize="xs" color="text.muted" mb={1}>To</Text>
                        <Input
                          size="sm"
                          type="date"
                          value={state.visitedTo}
                          onChange={e => set('visitedTo', e.target.value)}
                          bg="bg.surface"
                          borderRadius="7px"
                        />
                      </Box>
                    </Flex>
                  </Box>

                  <Box>
                    <Text fontSize="xs" color="text.secondary" mb={1} fontWeight="500">
                      Photo collection URL (optional)
                    </Text>
                    <Input
                      size="sm"
                      value={state.photoCollectionUrl}
                      onChange={e => set('photoCollectionUrl', e.target.value)}
                      placeholder="https://…"
                      bg="bg.surface"
                      borderRadius="7px"
                    />
                  </Box>
                </Stack>
              </Dialog.Body>

              <Dialog.Footer borderTopWidth="1px" borderColor="border.default" pt={3} gap={2}>
                <Button variant="outline" size="sm" onClick={onClose} disabled={submitting} borderRadius="7px">
                  Cancel
                </Button>
                <Button
                  type="submit"
                  size="sm"
                  disabled={submitting}
                  loading={submitting}
                  loadingText={isEdit ? 'Saving…' : 'Adding…'}
                  borderRadius="7px"
                >
                  {isEdit ? 'Save changes' : 'Add location'}
                </Button>
              </Dialog.Footer>
            </form>
          </Dialog.Content>
        </Dialog.Positioner>
      </Portal>
    </Dialog.Root>
  )
}
