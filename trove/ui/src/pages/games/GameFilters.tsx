import { Box, Stack, Input, Button, NativeSelect, Text } from '@chakra-ui/react'

interface GameFiltersProps {
  search: string
  onSearch: (v: string) => void
  yearPlayed: number | undefined
  onYearPlayed: (v: number | undefined) => void
  sort: string
  onSort: (v: string) => void
  onAdd: () => void
  onClear: () => void
  hasFilters: boolean
}

const CURRENT_YEAR = new Date().getFullYear()
const MIN_YEAR = 2022
const YEAR_OPTIONS = Array.from({ length: CURRENT_YEAR - MIN_YEAR + 1 }, (_, i) => CURRENT_YEAR - i)

export default function GameFilters({
  search, onSearch,
  yearPlayed, onYearPlayed,
  sort, onSort,
  onAdd, onClear, hasFilters,
}: GameFiltersProps) {
  return (
    <Stack gap={5} h="full">
      <Button size="sm" onClick={onAdd} borderRadius="8px" fontWeight="600" w="full">
        + Add Game
      </Button>

      <Box>
        <Text fontSize="xs" fontWeight="600" color="text.secondary" mb={1.5} letterSpacing="0.04em" textTransform="uppercase">
          Search
        </Text>
        <Input
          placeholder="Title or studio…"
          value={search}
          onChange={e => onSearch(e.target.value)}
          size="sm"
          bg="bg.surface"
          borderColor="border.default"
          _placeholder={{ color: 'text.muted' }}
          borderRadius="7px"
        />
      </Box>

      <Box>
        <Text fontSize="xs" fontWeight="600" color="text.secondary" mb={1.5} letterSpacing="0.04em" textTransform="uppercase">
          Year Played
        </Text>
        <NativeSelect.Root size="sm" w="full">
          <NativeSelect.Field
            value={yearPlayed ?? ''}
            onChange={e => onYearPlayed(e.target.value ? Number(e.target.value) : undefined)}
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

      {hasFilters && (
        <Button variant="outline" size="xs" onClick={onClear} borderRadius="7px" fontWeight="500" w="full">
          Clear filters
        </Button>
      )}
    </Stack>
  )
}
