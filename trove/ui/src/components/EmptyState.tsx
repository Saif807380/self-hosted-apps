import { Box, Flex, Heading, Text, Button } from '@chakra-ui/react'

interface EmptyStateProps {
  hasFilters: boolean
  noun?: string
  icon?: string
  onAdd?: () => void
  onClear?: () => void
}

export default function EmptyState({
  hasFilters,
  noun = 'book',
  icon = '📚',
  onAdd,
  onClear,
}: EmptyStateProps) {
  return (
    <Flex
      direction="column"
      align="center"
      justify="center"
      py={20}
      px={6}
      textAlign="center"
    >
      {/* Decorative stacked cards */}
      <Box mb={6} position="relative" w="80px" h="80px">
        {[
          { bg: '#d1fae5', rotate: '-12deg', z: 1 },
          { bg: '#dbeafe', rotate: '6deg', z: 2 },
          { bg: '#f3e8ff', rotate: '-3deg', z: 3 },
        ].map((card, i) => (
          <Box
            key={i}
            position="absolute"
            inset={0}
            borderRadius="sm"
            bg={card.bg}
            transform={`rotate(${card.rotate})`}
            zIndex={card.z}
            boxShadow="sm"
          />
        ))}
        <Box
          position="absolute"
          inset={0}
          borderRadius="sm"
          bg="bg.surface"
          zIndex={4}
          display="flex"
          alignItems="center"
          justifyContent="center"
          fontSize="2xl"
          boxShadow="sm"
          border="1px solid"
          borderColor="border.default"
        >
          {icon}
        </Box>
      </Box>

      {hasFilters ? (
        <>
          <Heading size="md" fontFamily="heading" fontWeight="600" color="text.primary" mb={2}>
            No {noun}s found
          </Heading>
          <Text color="text.secondary" fontSize="sm" maxW="280px" mb={6}>
            Try adjusting your search or filter criteria.
          </Text>
          {onClear && (
            <Button variant="outline" size="sm" onClick={onClear}>
              Clear filters
            </Button>
          )}
        </>
      ) : (
        <>
          <Heading size="md" fontFamily="heading" fontWeight="600" color="text.primary" mb={2}>
            Nothing here yet
          </Heading>
          <Text color="text.secondary" fontSize="sm" maxW="300px" mb={6}>
            Start by adding your first {noun} to the collection.
          </Text>
          {onAdd && (
            <Button onClick={onAdd} size="sm">
              Add your first {noun}
            </Button>
          )}
        </>
      )}
    </Flex>
  )
}
