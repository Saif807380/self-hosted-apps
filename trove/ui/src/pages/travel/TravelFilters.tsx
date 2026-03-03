import { Box, Stack, Input, Button, NativeSelect, Text } from '@chakra-ui/react'

interface TravelFiltersProps {
  search: string
  onSearch: (v: string) => void
  sort: string
  onSort: (v: string) => void
  onAdd: () => void
  onClear: () => void
  hasFilters: boolean
}

export default function TravelFilters({
  search, onSearch,
  sort, onSort,
  onAdd, onClear, hasFilters,
}: TravelFiltersProps) {
  return (
    <Stack gap={5} h="full">
      <Button size="sm" onClick={onAdd} borderRadius="8px" fontWeight="600" w="full">
        + Add Location
      </Button>

      <Box>
        <Text fontSize="xs" fontWeight="600" color="text.secondary" mb={1.5} letterSpacing="0.04em" textTransform="uppercase">
          Search
        </Text>
        <Input
          placeholder="City or country…"
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
            <option value="city_asc">City: A → Z</option>
            <option value="city_desc">City: Z → A</option>
            <option value="date_desc">Date: Newest first</option>
            <option value="date_asc">Date: Oldest first</option>
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
