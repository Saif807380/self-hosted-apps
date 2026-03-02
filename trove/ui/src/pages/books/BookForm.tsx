import { useState, useEffect, useRef } from 'react'
import type { FormEvent, KeyboardEvent, ChangeEvent } from 'react'
import {
  Dialog,
  Portal,
  Button,
  Input,
  Textarea,
  Stack,
  Flex,
  Box,
  Text,
  Spinner,
  CloseButton,
} from '@chakra-ui/react'
import type { Book, BookFormData, Tag } from '@/types/api'
import StarRating from '@/components/StarRating'
import { uploadFile } from '@/services/upload'

interface BookFormProps {
  isOpen: boolean
  onClose: () => void
  onSubmit: (data: BookFormData) => Promise<void>
  tags: Tag[]
  onCreateTag: (name: string) => Promise<Tag>
  editBook?: Book | null
}

const CURRENT_YEAR = new Date().getFullYear()

interface FormState {
  title: string
  author: string
  rating: number
  review: string
  coverImage: string
  yearsRead: number[]
  selectedTagIds: Set<string>
  yearInput: string
  newTagInput: string
}

function initState(book?: Book | null): FormState {
  return {
    title: book?.title ?? '',
    author: book?.author ?? '',
    rating: book?.rating ?? 0,
    review: book?.review ?? '',
    coverImage: book?.coverImage ?? '',
    yearsRead: book?.yearsRead ?? [],
    selectedTagIds: new Set(book?.tags.map(t => t.id) ?? []),
    yearInput: '',
    newTagInput: '',
  }
}

const EMPTY_ERRORS = { title: '', author: '', yearsRead: '' }

export default function BookForm({
  isOpen, onClose, onSubmit, tags, onCreateTag, editBook,
}: BookFormProps) {
  const [state, setState] = useState<FormState>(() => initState(editBook))
  const [errors, setErrors] = useState(EMPTY_ERRORS)
  const [uploading, setUploading] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [creatingTag, setCreatingTag] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (isOpen) {
      setState(initState(editBook))
      setErrors(EMPTY_ERRORS)
      setUploading(false)
      setSubmitting(false)
    }
  }, [isOpen, editBook])

  const set = <K extends keyof FormState>(key: K, value: FormState[K]) =>
    setState(prev => ({ ...prev, [key]: value }))

  const handleCoverClick = () => fileInputRef.current?.click()

  const handleFileChange = async (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    setUploading(true)
    try {
      const path = await uploadFile(file)
      set('coverImage', path)
    } catch {
      // silently ignore, user can retry
    } finally {
      setUploading(false)
      // reset so same file can be re-selected
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  const handleYearKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key !== 'Enter') return
    e.preventDefault()
    const year = parseInt(state.yearInput.trim(), 10)
    if (year >= 1900 && year <= CURRENT_YEAR + 1 && !state.yearsRead.includes(year)) {
      setState(prev => ({
        ...prev,
        yearsRead: [...prev.yearsRead, year].sort((a, b) => a - b),
        yearInput: '',
      }))
    }
  }

  const removeYear = (year: number) =>
    setState(prev => ({ ...prev, yearsRead: prev.yearsRead.filter(y => y !== year) }))

  const toggleTag = (id: string) =>
    setState(prev => {
      const next = new Set(prev.selectedTagIds)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return { ...prev, selectedTagIds: next }
    })

  const handleNewTagKeyDown = async (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key !== 'Enter') return
    e.preventDefault()
    const name = state.newTagInput.trim()
    if (!name || creatingTag) return
    setCreatingTag(true)
    try {
      const tag = await onCreateTag(name)
      setState(prev => {
        const next = new Set(prev.selectedTagIds)
        next.add(tag.id)
        return { ...prev, selectedTagIds: next, newTagInput: '' }
      })
    } finally {
      setCreatingTag(false)
    }
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    if (submitting) return

    const newErrors = {
      title: state.title.trim() ? '' : 'Title is required',
      author: state.author.trim() ? '' : 'Author is required',
      yearsRead: state.yearsRead.length > 0 ? '' : 'At least one year is required',
    }

    if (Object.values(newErrors).some(Boolean)) {
      setErrors(newErrors)
      return
    }

    setSubmitting(true)
    try {
      await onSubmit({
        title: state.title.trim(),
        author: state.author.trim(),
        rating: state.rating || undefined,
        review: state.review.trim() || undefined,
        coverImage: state.coverImage || undefined,
        yearsRead: state.yearsRead,
        tagIds: Array.from(state.selectedTagIds),
      })
    } finally {
      setSubmitting(false)
    }
  }

  const isEdit = !!editBook
  const canSubmit = !submitting

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
                  {isEdit ? 'Edit book' : 'Add a book'}
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
                  {/* Cover + title + author row */}
                  <Flex gap={4} align="stretch">
                    {/* Cover upload */}
                    <Box
                      w="76px"
                      borderRadius="8px"
                      border="2px dashed"
                      borderColor="border.default"
                      cursor="pointer"
                      onClick={handleCoverClick}
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
                        <Text
                          fontSize="xs"
                          color="text.muted"
                          textAlign="center"
                          px={2}
                          lineHeight="1.4"
                        >
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

                    {/* Title + author */}
                    <Stack gap={3} flex={1} minW={0}>
                      <Box>
                        <Text fontSize="xs" color="text.secondary" mb={1} fontWeight="500">
                          Title <span style={{ color: 'var(--chakra-colors-accent)' }}>*</span>
                        </Text>
                        <Input
                          size="sm"
                          value={state.title}
                          onChange={e => { set('title', e.target.value); if (errors.title) setErrors(prev => ({ ...prev, title: '' })) }}
                          placeholder="Book title"
                          bg="bg.surface"
                          borderRadius="7px"
                          borderColor={errors.title ? 'red.400' : undefined}
                        />
                        {errors.title && <Text fontSize="xs" color="red.500" mt={1}>{errors.title}</Text>}
                      </Box>
                      <Box>
                        <Text fontSize="xs" color="text.secondary" mb={1} fontWeight="500">
                          Author <span style={{ color: 'var(--chakra-colors-accent)' }}>*</span>
                        </Text>
                        <Input
                          size="sm"
                          value={state.author}
                          onChange={e => { set('author', e.target.value); if (errors.author) setErrors(prev => ({ ...prev, author: '' })) }}
                          placeholder="Author name"
                          bg="bg.surface"
                          borderRadius="7px"
                          borderColor={errors.author ? 'red.400' : undefined}
                        />
                        {errors.author && <Text fontSize="xs" color="red.500" mt={1}>{errors.author}</Text>}
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

                  {/* Years read */}
                  <Box>
                    <Text fontSize="xs" color="text.secondary" mb={1.5} fontWeight="500">
                      Read in <span style={{ color: 'var(--chakra-colors-accent)' }}>*</span>
                    </Text>
                    {state.yearsRead.length > 0 && (
                      <Flex gap={1.5} wrap="wrap" mb={2}>
                        {state.yearsRead.map(year => (
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
                                background: 'none',
                                border: 'none',
                                cursor: 'pointer',
                                padding: 0,
                                lineHeight: 1,
                                color: 'inherit',
                                opacity: 0.6,
                                fontSize: '1em',
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
                      onChange={e => { set('yearInput', e.target.value); if (errors.yearsRead) setErrors(prev => ({ ...prev, yearsRead: '' })) }}
                      onKeyDown={handleYearKeyDown}
                      placeholder={`e.g. ${CURRENT_YEAR} — press Enter to add`}
                      min={1900}
                      max={CURRENT_YEAR + 1}
                      bg="bg.surface"
                      borderRadius="7px"
                      maxW="280px"
                      borderColor={errors.yearsRead ? 'red.400' : undefined}
                    />
                    {errors.yearsRead && <Text fontSize="xs" color="red.500" mt={1}>{errors.yearsRead}</Text>}
                  </Box>

                  {/* Tags */}
                  {tags.length > 0 && (
                    <Box>
                      <Text fontSize="xs" color="text.secondary" mb={1.5} fontWeight="500">
                        Tags
                      </Text>
                      <Flex gap={1.5} wrap="wrap" mb={2}>
                        {tags.map(tag => {
                          const selected = state.selectedTagIds.has(tag.id)
                          return (
                            <button
                              key={tag.id}
                              type="button"
                              onClick={() => toggleTag(tag.id)}
                              style={{
                                display: 'inline-flex',
                                alignItems: 'center',
                                padding: '4px 12px',
                                borderRadius: '999px',
                                fontSize: '0.75rem',
                                fontWeight: '500',
                                cursor: 'pointer',
                                border: '1.5px solid',
                                transition: 'all 0.12s ease',
                                background: selected
                                  ? 'var(--chakra-colors-accent-subtle)'
                                  : 'transparent',
                                borderColor: selected
                                  ? 'var(--chakra-colors-accent)'
                                  : 'var(--chakra-colors-border-default)',
                                color: selected
                                  ? 'var(--chakra-colors-accent)'
                                  : 'var(--chakra-colors-text-secondary)',
                              }}
                            >
                              {tag.name}
                            </button>
                          )
                        })}
                      </Flex>
                    </Box>
                  )}

                  {/* Create new tag */}
                  <Box>
                    <Text fontSize="xs" color="text.secondary" mb={1.5} fontWeight="500">
                      {tags.length > 0 ? 'Create new tag' : 'Tags'}
                    </Text>
                    <Input
                      size="sm"
                      value={state.newTagInput}
                      onChange={e => set('newTagInput', e.target.value)}
                      onKeyDown={handleNewTagKeyDown}
                      placeholder="Tag name — press Enter to create"
                      bg="bg.surface"
                      borderRadius="7px"
                      maxW="280px"
                      disabled={creatingTag}
                    />
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
                      placeholder="Your thoughts on the book…"
                      rows={3}
                      bg="bg.surface"
                      borderRadius="7px"
                      resize="none"
                    />
                  </Box>
                </Stack>
              </Dialog.Body>

              <Dialog.Footer borderTopWidth="1px" borderColor="border.default" pt={3} gap={2}>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={onClose}
                  disabled={submitting}
                  borderRadius="7px"
                >
                  Cancel
                </Button>
                <Button
                  type="submit"
                  size="sm"
                  disabled={!canSubmit}
                  loading={submitting}
                  loadingText={isEdit ? 'Saving…' : 'Adding…'}
                  borderRadius="7px"
                >
                  {isEdit ? 'Save changes' : 'Add book'}
                </Button>
              </Dialog.Footer>
            </form>
          </Dialog.Content>
        </Dialog.Positioner>
      </Portal>
    </Dialog.Root>
  )
}
