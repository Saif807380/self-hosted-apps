import { useState, useEffect, useRef } from 'react'
import type { FormEvent, KeyboardEvent, ChangeEvent } from 'react'
import {
  Dialog, Portal, Button, Input, Textarea, Stack, Flex, Box, Text, Spinner, CloseButton,
} from '@chakra-ui/react'
import type { VideoGame, VideoGameFormData } from '@/types/api'
import StarRating from '@/components/StarRating'
import { uploadFile } from '@/services/upload'

interface GameFormProps {
  isOpen: boolean
  onClose: () => void
  onSubmit: (data: VideoGameFormData) => Promise<void>
  editGame?: VideoGame | null
}

const CURRENT_YEAR = new Date().getFullYear()

interface FormState {
  title: string
  studio: string
  rating: number
  review: string
  coverImage: string
  yearsPlayed: number[]
  yearInput: string
}

function initState(game?: VideoGame | null): FormState {
  return {
    title: game?.title ?? '',
    studio: game?.studio ?? '',
    rating: game?.rating ?? 0,
    review: game?.review ?? '',
    coverImage: game?.coverImage ?? '',
    yearsPlayed: game?.yearsPlayed ?? [],
    yearInput: '',
  }
}

const EMPTY_ERRORS = { title: '', yearsPlayed: '' }

export default function GameForm({ isOpen, onClose, onSubmit, editGame }: GameFormProps) {
  const [state, setState] = useState<FormState>(() => initState(editGame))
  const [errors, setErrors] = useState(EMPTY_ERRORS)
  const [uploading, setUploading] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (isOpen) {
      setState(initState(editGame))
      setErrors(EMPTY_ERRORS)
      setUploading(false)
      setSubmitting(false)
    }
  }, [isOpen, editGame])

  const set = <K extends keyof FormState>(key: K, value: FormState[K]) =>
    setState(prev => ({ ...prev, [key]: value }))

  const handleFileChange = async (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    setUploading(true)
    try {
      const path = await uploadFile(file)
      set('coverImage', path)
    } catch {
      // ignore, user can retry
    } finally {
      setUploading(false)
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  const handleYearKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key !== 'Enter') return
    e.preventDefault()
    const year = parseInt(state.yearInput.trim(), 10)
    if (year >= 1970 && year <= CURRENT_YEAR + 1 && !state.yearsPlayed.includes(year)) {
      setState(prev => ({
        ...prev,
        yearsPlayed: [...prev.yearsPlayed, year].sort((a, b) => a - b),
        yearInput: '',
      }))
    }
  }

  const removeYear = (year: number) =>
    setState(prev => ({ ...prev, yearsPlayed: prev.yearsPlayed.filter(y => y !== year) }))

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    if (submitting) return

    const newErrors = {
      title: state.title.trim() ? '' : 'Title is required',
      yearsPlayed: state.yearsPlayed.length > 0 ? '' : 'At least one year is required',
    }
    if (Object.values(newErrors).some(Boolean)) {
      setErrors(newErrors)
      return
    }

    setSubmitting(true)
    try {
      await onSubmit({
        title: state.title.trim(),
        studio: state.studio.trim() || undefined,
        rating: state.rating || undefined,
        review: state.review.trim() || undefined,
        coverImage: state.coverImage || undefined,
        yearsPlayed: state.yearsPlayed,
      })
    } finally {
      setSubmitting(false)
    }
  }

  const isEdit = !!editGame

  return (
    <Dialog.Root
      open={isOpen}
      onOpenChange={({ open }: { open: boolean }) => !open && !submitting && onClose()}
      scrollBehavior="inside"
    >
      <Portal>
        <Dialog.Backdrop />
        <Dialog.Positioner>
          <Dialog.Content maxW="500px" mx={4}>
            <form onSubmit={handleSubmit} style={{ display: 'contents' }}>
              <Dialog.Header borderBottomWidth="1px" borderColor="border.default" pb={3}>
                <Dialog.Title fontFamily="heading" fontWeight="600" fontSize="lg">
                  {isEdit ? 'Edit game' : 'Add a game'}
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
                <Stack gap={5}>
                  {/* Cover + title + studio */}
                  <Flex gap={4} align="stretch">
                    <Box
                      w="76px"
                      borderRadius="8px"
                      border="2px dashed"
                      borderColor="border.default"
                      cursor="pointer"
                      onClick={() => fileInputRef.current?.click()}
                      overflow="hidden"
                      flexShrink={0}
                      display="flex"
                      alignItems="center"
                      justifyContent="center"
                      bg="bg.subtle"
                      position="relative"
                      style={{ transition: 'border-color 0.15s ease' }}
                      _hover={{ borderColor: 'accent' }}
                    >
                      {uploading ? (
                        <Spinner size="sm" color="accent" />
                      ) : state.coverImage ? (
                        <img
                          src={state.coverImage}
                          alt="Cover preview"
                          style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                        />
                      ) : (
                        <Text fontSize="xs" color="text.muted" textAlign="center" px={2} lineHeight="1.4">
                          Add cover
                        </Text>
                      )}
                    </Box>
                    <input
                      ref={fileInputRef}
                      type="file"
                      accept=".jpg,.jpeg,.png,.webp"
                      style={{ display: 'none' }}
                      onChange={handleFileChange}
                    />

                    <Stack gap={3} flex={1} minW={0}>
                      <Box>
                        <Text fontSize="xs" color="text.secondary" mb={1} fontWeight="500">
                          Title <span style={{ color: 'var(--chakra-colors-accent)' }}>*</span>
                        </Text>
                        <Input
                          size="sm"
                          value={state.title}
                          onChange={e => { set('title', e.target.value); if (errors.title) setErrors(prev => ({ ...prev, title: '' })) }}
                          placeholder="Game title"
                          bg="bg.surface"
                          borderRadius="7px"
                          borderColor={errors.title ? 'red.400' : undefined}
                        />
                        {errors.title && <Text fontSize="xs" color="red.500" mt={1}>{errors.title}</Text>}
                      </Box>
                      <Box>
                        <Text fontSize="xs" color="text.secondary" mb={1} fontWeight="500">
                          Studio / Publisher
                        </Text>
                        <Input
                          size="sm"
                          value={state.studio}
                          onChange={e => set('studio', e.target.value)}
                          placeholder="Studio name"
                          bg="bg.surface"
                          borderRadius="7px"
                        />
                      </Box>
                    </Stack>
                  </Flex>

                  {/* Rating */}
                  <Box>
                    <Text fontSize="xs" color="text.secondary" mb={1.5} fontWeight="500">
                      Rating
                    </Text>
                    <StarRating value={state.rating} onChange={v => set('rating', v)} size="lg" />
                  </Box>

                  {/* Years played */}
                  <Box>
                    <Text fontSize="xs" color="text.secondary" mb={1.5} fontWeight="500">
                      Played in <span style={{ color: 'var(--chakra-colors-accent)' }}>*</span>
                    </Text>
                    {state.yearsPlayed.length > 0 && (
                      <Flex gap={1.5} wrap="wrap" mb={2}>
                        {state.yearsPlayed.map(year => (
                          <Box
                            key={year}
                            display="inline-flex"
                            alignItems="center"
                            gap={1}
                            px={2}
                            py="3px"
                            borderRadius="6px"
                            fontSize="xs"
                            fontWeight="500"
                            bg="bg.subtle"
                            color="text.secondary"
                          >
                            {year}
                            <button
                              type="button"
                              onClick={() => removeYear(year)}
                              style={{
                                background: 'none', border: 'none', cursor: 'pointer',
                                padding: 0, lineHeight: 1, color: 'inherit',
                                opacity: 0.6, fontSize: '1em',
                              }}
                              aria-label={`Remove ${year}`}
                            >
                              ×
                            </button>
                          </Box>
                        ))}
                      </Flex>
                    )}
                    <Input
                      size="sm"
                      type="number"
                      value={state.yearInput}
                      onChange={e => { set('yearInput', e.target.value); if (errors.yearsPlayed) setErrors(prev => ({ ...prev, yearsPlayed: '' })) }}
                      onKeyDown={handleYearKeyDown}
                      placeholder={`e.g. ${CURRENT_YEAR} — press Enter to add`}
                      min={1970}
                      max={CURRENT_YEAR + 1}
                      bg="bg.surface"
                      borderRadius="7px"
                      maxW="280px"
                      borderColor={errors.yearsPlayed ? 'red.400' : undefined}
                    />
                    {errors.yearsPlayed && <Text fontSize="xs" color="red.500" mt={1}>{errors.yearsPlayed}</Text>}
                  </Box>

                  {/* Review */}
                  <Box>
                    <Text fontSize="xs" color="text.secondary" mb={1.5} fontWeight="500">
                      Review (optional)
                    </Text>
                    <Textarea
                      size="sm"
                      value={state.review}
                      onChange={e => set('review', e.target.value)}
                      placeholder="Your thoughts on the game…"
                      rows={3}
                      bg="bg.surface"
                      borderRadius="7px"
                      resize="none"
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
                  {isEdit ? 'Save changes' : 'Add game'}
                </Button>
              </Dialog.Footer>
            </form>
          </Dialog.Content>
        </Dialog.Positioner>
      </Portal>
    </Dialog.Root>
  )
}
