import { Box, Stack, Input, Button, NativeSelect, Text, Flex } from '@chakra-ui/react'
import type { Tag } from '@/types/api'

interface BookFiltersProps {
  search: string
  onSearch: (v: string) => void
  yearRead: number | undefined
  onYearRead: (v: number | undefined) => void
  tagIds: string[]
  onTagIds: (v: string[]) => void
  sort: string
  onSort: (v: string) => void
  tags: Tag[]
  onAdd: () => void
  onClear: () => void
  hasFilters: boolean
}

const CURRENT_YEAR = new Date().getFullYear()
const MIN_YEAR = 2022
const YEAR_OPTIONS = Array.from({ length: CURRENT_YEAR - MIN_YEAR + 1 }, (_, i) => CURRENT_YEAR - i)

export default function BookFilters({
  search, onSearch,
  yearRead, onYearRead,
  tagIds, onTagIds,
  sort, onSort,
  tags,
  onAdd,
  onClear,
  hasFilters,
}: BookFiltersProps) {
  const toggleTag = (id: string) =>
    onTagIds(tagIds.includes(id) ? tagIds.filter(t => t !== id) : [...tagIds, id])

  return (
    <Stack gap={5} h="full">
      {/* Add Book — primary action */}
      <Button
        size="sm"
        onClick={onAdd}
        borderRadius="8px"
        fontWeight="600"
        w="full"
      >
        + Add Book
      </Button>

      {/* Search */}
      <Box>
        <Text fontSize="xs" fontWeight="600" color="text.secondary" mb={1.5} letterSpacing="0.04em" textTransform="uppercase">
          Search
        </Text>
        <Input
          placeholder="Title or author…"
          value={search}
          onChange={e => onSearch(e.target.value)}
          size="sm"
          bg="bg.surface"
          borderColor="border.default"
          _placeholder={{ color: 'text.muted' }}
          borderRadius="7px"
        />
      </Box>

      {/* Year filter */}
      <Box>
        <Text fontSize="xs" fontWeight="600" color="text.secondary" mb={1.5} letterSpacing="0.04em" textTransform="uppercase">
          Year Read
        </Text>
        <NativeSelect.Root size="sm" w="full">
          <NativeSelect.Field
            value={yearRead ?? ''}
            onChange={e => onYearRead(e.target.value ? Number(e.target.value) : undefined)}
            bg="bg.surface"
            borderColor="border.default"
            borderRadius="7px"
          >
            <option value="">All years</option>
            {YEAR_OPTIONS.map(y => (
              <option key={y} value={y}>{y}</option>
            ))}
          </NativeSelect.Field>
          <NativeSelect.Indicator />
        </NativeSelect.Root>
      </Box>

      {/* Tag filter — multi-select pills */}
      {tags.length > 0 && (
        <Box>
          <Text fontSize="xs" fontWeight="600" color="text.secondary" mb={1.5} letterSpacing="0.04em" textTransform="uppercase">
            Tags
          </Text>
          <Flex gap={1.5} wrap="wrap">
            {tags.map(tag => {
              const selected = tagIds.includes(tag.id)
              return (
                <button
                  key={tag.id}
                  type="button"
                  onClick={() => toggleTag(tag.id)}
                  style={{
                    display: 'inline-flex',
                    alignItems: 'center',
                    padding: '3px 10px',
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

      {/* Sort */}
      <Box>
        <Text fontSize="xs" fontWeight="600" color="text.secondary" mb={1.5} letterSpacing="0.04em" textTransform="uppercase">
          Sort
        </Text>
        <NativeSelect.Root size="sm" w="full">
          <NativeSelect.Field
            value={sort}
            onChange={e => onSort(e.target.value)}
            bg="bg.surface"
            borderColor="border.default"
            borderRadius="7px"
          >
            <option value="title_asc">Title: A → Z</option>
            <option value="title_desc">Title: Z → A</option>
            <option value="year_desc">Year: Newest first</option>
            <option value="year_asc">Year: Oldest first</option>
          </NativeSelect.Field>
          <NativeSelect.Indicator />
        </NativeSelect.Root>
      </Box>

      {/* Clear filters */}
      {hasFilters && (
        <Button
          variant="outline"
          size="xs"
          onClick={onClear}
          borderRadius="7px"
          fontWeight="500"
          w="full"
        >
          Clear filters
        </Button>
      )}
    </Stack>
  )
}
