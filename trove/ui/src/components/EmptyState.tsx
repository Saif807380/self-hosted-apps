import { Box, Flex, Heading, Text, Button } from '@chakra-ui/react'

interface EmptyStateProps {
  hasFilters: boolean
  onAdd?: () => void
  onClear?: () => void
}

export default function EmptyState({ hasFilters, onAdd, onClear }: EmptyStateProps) {
  return (
    <Flex
      direction="column"
      align="center"
      justify="center"
      py={20}
      px={6}
      textAlign="center"
    >
      {/* Decorative stack of book spines */}
      <Box mb={6} position="relative" w="80px" h="80px">
        {[
          { bg: '#d1fae5', rotate: '-12deg', z: 1 },
          { bg: '#dbeafe', rotate: '6deg', z: 2 },
          { bg: '#f3e8ff', rotate: '-3deg', z: 3 },
        ].map((book, i) => (
          <Box
            key={i}
            position="absolute"
            inset={0}
            borderRadius="sm"
            bg={book.bg}
            transform={`rotate(${book.rotate})`}
            zIndex={book.z}
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
          📚
        </Box>
      </Box>

      {hasFilters ? (
        <>
          <Heading size="md" fontFamily="heading" fontWeight="600" color="text.primary" mb={2}>
            No books found
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
            Your bookshelf is empty
          </Heading>
          <Text color="text.secondary" fontSize="sm" maxW="300px" mb={6}>
            Start tracking your reading journey by adding the first book to your collection.
          </Text>
          {onAdd && (
            <Button onClick={onAdd} size="sm">
              Add your first book
            </Button>
          )}
        </>
      )}
    </Flex>
  )
}
