import { Box, Flex, Text } from '@chakra-ui/react'
import { NavLink } from 'react-router-dom'
import ThemeToggle from './ThemeToggle'

const NAV_ITEMS = [
  { label: '📚 Books', path: '/books' },
  { label: '🎮 Gaming', path: '/games' },
  { label: '✈️ Travel', path: '/travel' },
  { label: '🏋️ Workouts', path: '/workouts' },
]

export default function Layout({ children }: { children: React.ReactNode }) {
  return (
    <Flex direction="column" minH="100vh">
      <Box
        as="header"
        bg="bg.surface"
        borderBottomWidth="1px"
        borderColor="border.default"
        position="sticky"
        top={0}
        zIndex={100}
      >
        <Flex align="center" justify="space-between" maxW="2000px" mx="auto" px={6} py={3}>
          <Flex align="center" gap={2}>
            <img src="/favicon.svg" alt="" width={22} height={22} />
            <Text
              fontFamily="heading"
              fontWeight="700"
              fontSize="xl"
              color="text.primary"
              letterSpacing="-0.02em"
            >
              Trove
            </Text>
          </Flex>

          <Flex as="nav" gap={1} align="center">
            {NAV_ITEMS.map(({ label, path }) => (
              <NavLink
                key={path}
                to={path}
                style={({ isActive }) => ({
                  textDecoration: 'none',
                  display: 'block',
                  padding: '5px 13px',
                  borderRadius: '8px',
                  fontSize: '0.875rem',
                  fontWeight: isActive ? '600' : '400',
                  color: isActive
                    ? 'var(--chakra-colors-accent)'
                    : 'var(--chakra-colors-text-secondary)',
                  backgroundColor: isActive
                    ? 'var(--chakra-colors-accent-subtle)'
                    : 'transparent',
                  transition: 'all 0.15s ease',
                })}
              >
                {label}
              </NavLink>
            ))}
            <Box ml={2}>
              <ThemeToggle />
            </Box>
          </Flex>
        </Flex>
      </Box>

      <Box as="main" flex="1" px={6} py={8} maxW="2000px" mx="auto" w="full">
        {children}
      </Box>
    </Flex>
  )
}
