import { Box, Heading, Text } from '@chakra-ui/react'

export default function BooksPage() {
  return (
    <Box>
      <Heading size="lg" color="text.primary" mb={2}>Books</Heading>
      <Text color="text.secondary">Your book collection will appear here.</Text>
    </Box>
  )
}
