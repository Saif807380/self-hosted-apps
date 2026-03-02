import { Box, Stack, Input, Button, NativeSelect, Text } from '@chakra-ui/react'
import type { Tag } from '@/types/api'

interface BookFiltersProps {
  search: string
  onSearch: (v: string) => void
  yearRead: number | undefined
  onYearRead: (v: number | undefined) => void
  tagId: string
  onTagId: (v: string) => void
  sort: string
  onSort: (v: string) => void
  tags: Tag[]
  onAdd: () => void
  onClear: () => void
  hasFilters: boolean
}

const CURRENT_YEAR = new Date().getFullYear()
const YEAR_OPTIONS = Array.from({ length: 30 }, (_, i) => CURRENT_YEAR - i)

export default function BookFilters({
  search, onSearch,
  yearRead, onYearRead,
  tagId, onTagId,
  sort, onSort,
  tags,
  onAdd,
  onClear,
  hasFilters,
}: BookFiltersProps) {
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

      {/* Tag filter */}
      <Box>
        <Text fontSize="xs" fontWeight="600" color="text.secondary" mb={1.5} letterSpacing="0.04em" textTransform="uppercase">
          Tag
        </Text>
        <NativeSelect.Root size="sm" w="full">
          <NativeSelect.Field
            value={tagId}
            onChange={e => onTagId(e.target.value)}
            bg="bg.surface"
            borderColor="border.default"
            borderRadius="7px"
          >
            <option value="">All tags</option>
            {tags.map(t => (
              <option key={t.id} value={t.id}>{t.name}</option>
            ))}
          </NativeSelect.Field>
          <NativeSelect.Indicator />
        </NativeSelect.Root>
      </Box>

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
