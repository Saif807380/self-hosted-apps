import { useState, useCallback } from 'react'
import { Box, Flex, Text } from '@chakra-ui/react'
import { NavLink } from 'react-router-dom'
import ThemeToggle from './ThemeToggle'
import SyncIndicator from './SyncIndicator'
import OfflineBanner from './OfflineBanner'

const NAV_ITEMS = [
  { label: '📚 Books', path: '/books' },
  { label: '🎮 Gaming', path: '/games' },
  { label: '✈️ Travel', path: '/travel' },
  { label: '🏋️ Workouts', path: '/workouts' },
]

const navLinkStyle = (isActive: boolean) => ({
  textDecoration: 'none' as const,
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
})

const mobileNavLinkStyle = (isActive: boolean) => ({
  ...navLinkStyle(isActive),
  padding: '10px 16px',
  fontSize: '1rem',
  borderRadius: '6px',
})

const HamburgerIcon = ({ open }: { open: boolean }) => (
  <svg
    width="20"
    height="20"
    viewBox="0 0 20 20"
    fill="none"
    stroke="currentColor"
    strokeWidth="1.8"
    strokeLinecap="round"
    style={{ transition: 'transform 0.2s ease' }}
  >
    {open ? (
      <>
        <line x1="4" y1="4" x2="16" y2="16" />
        <line x1="16" y1="4" x2="4" y2="16" />
      </>
    ) : (
      <>
        <line x1="3" y1="5" x2="17" y2="5" />
        <line x1="3" y1="10" x2="17" y2="10" />
        <line x1="3" y1="15" x2="17" y2="15" />
      </>
    )}
  </svg>
)

export default function Layout({ children }: { children: React.ReactNode }) {
  const [menuOpen, setMenuOpen] = useState(false)
  const handleNavClick = useCallback(() => setMenuOpen(false), [])

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

          {/* Desktop nav */}
          <Flex
            as="nav"
            gap={1}
            align="center"
            display={{ base: 'none', md: 'flex' }}
          >
            {NAV_ITEMS.map(({ label, path }) => (
              <NavLink key={path} to={path} style={({ isActive }) => navLinkStyle(isActive)}>
                {label}
              </NavLink>
            ))}
            <SyncIndicator />
            <Box ml={1}>
              <ThemeToggle />
            </Box>
          </Flex>

          {/* Mobile controls */}
          <Flex align="center" gap={1} display={{ base: 'flex', md: 'none' }}>
            <SyncIndicator />
            <Box ml={1}>
              <ThemeToggle />
            </Box>
            <button
              type="button"
              aria-label={menuOpen ? 'Close menu' : 'Open menu'}
              onClick={() => setMenuOpen((v) => !v)}
              style={{
                padding: '6px',
                marginLeft: '4px',
                background: 'none',
                border: 'none',
                cursor: 'pointer',
                color: 'var(--chakra-colors-text-primary)',
                borderRadius: '6px',
              }}
            >
              <HamburgerIcon open={menuOpen} />
            </button>
          </Flex>
        </Flex>

        {/* Mobile dropdown menu */}
        {menuOpen && (
          <Box
            display={{ base: 'block', md: 'none' }}
            px={4}
            pb={3}
            borderTopWidth="1px"
            borderColor="border.default"
          >
            <Flex direction="column" gap={1} pt={2}>
              {NAV_ITEMS.map(({ label, path }) => (
                <NavLink
                  key={path}
                  to={path}
                  onClick={handleNavClick}
                  style={({ isActive }) => mobileNavLinkStyle(isActive)}
                >
                  {label}
                </NavLink>
              ))}
            </Flex>
          </Box>
        )}
      </Box>

      <OfflineBanner />
      <Box as="main" flex="1" px={{ base: 3, md: 6 }} py={{ base: 4, md: 8 }} maxW="2000px" mx="auto" w="full">
        {children}
      </Box>
    </Flex>
  )
}
