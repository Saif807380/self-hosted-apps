import { Box, Flex, Text } from '@chakra-ui/react'
import { NavLink } from 'react-router-dom'
import ThemeToggle from './ThemeToggle'

const NAV_ITEMS = [
  { label: 'Books', path: '/books' },
  { label: 'Games', path: '/games' },
  { label: 'Travel', path: '/travel' },
  { label: 'Workouts', path: '/workouts' },
]

export default function Layout({ children }: { children: React.ReactNode }) {
  return (
    <Flex direction="column" minH="100vh">
      <Box
        as="header"
        bg="bg.surface"
        borderBottomWidth="1px"
        borderColor="border.default"
        px={6}
        py={3}
      >
        <Flex align="center" justify="space-between" maxW="1400px" mx="auto">
          <Text fontWeight="bold" fontSize="lg" color="text.primary">
            Trove
          </Text>
          <Flex as="nav" gap={6} align="center">
            {NAV_ITEMS.map(({ label, path }) => (
              <NavLink
                key={path}
                to={path}
                style={({ isActive }) => ({
                  fontWeight: 500,
                  textDecoration: 'none',
                  color: isActive ? 'var(--chakra-colors-accent)' : 'inherit',
                })}
              >
                {label}
              </NavLink>
            ))}
            <ThemeToggle />
          </Flex>
        </Flex>
      </Box>
      <Box as="main" flex="1" px={6} py={8} maxW="1400px" mx="auto" w="full">
        {children}
      </Box>
    </Flex>
  )
}
