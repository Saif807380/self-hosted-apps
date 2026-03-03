import { useState, useMemo } from 'react'
import { Box, Flex, Heading, SimpleGrid, Text, Spinner } from '@chakra-ui/react'
import { useGames } from '@/hooks/useGames'
import { useDebounce } from '@/hooks/useDebounce'
import { toaster } from '@/lib/toaster'
import type { VideoGame, VideoGameFormData } from '@/types/api'
import GameCard from './games/GameCard'
import GameListItem from './games/GameListItem'
import GameFilters from './games/GameFilters'
import GameForm from './games/GameForm'
import GameDetailModal from './games/GameDetailModal'
import EmptyState from '@/components/EmptyState'
import ConfirmDialog from '@/components/ConfirmDialog'
import ViewToggle from '@/components/ViewToggle'

const SIDEBAR_WIDTH = 240

export default function GamesPage() {
  const [search, setSearch] = useState('')
  const [yearPlayed, setYearPlayed] = useState<number | undefined>()
  const [sort, setSort] = useState('title_asc')
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const [formOpen, setFormOpen] = useState(false)
  const [editGame, setEditGame] = useState<VideoGame | null>(null)
  const [viewGame, setViewGame] = useState<VideoGame | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<VideoGame | null>(null)
  const [view, setView] = useState<'grid' | 'list'>('grid')

  const debouncedSearch = useDebounce(search, 300)
  const { games, loading, error, createGame, updateGame, deleteGame } = useGames({
    search: debouncedSearch,
    yearPlayed,
  })

  const hasFilters = !!search || !!yearPlayed

  const sortedGames = useMemo(() => {
    return [...games].sort((a, b) => {
      switch (sort) {
        case 'title_desc': return b.title.localeCompare(a.title)
        case 'year_desc': {
          const aYear = a.yearsPlayed.length ? Math.max(...a.yearsPlayed) : -1
          const bYear = b.yearsPlayed.length ? Math.max(...b.yearsPlayed) : -1
          return bYear - aYear
        }
        case 'year_asc': {
          const aYear = a.yearsPlayed.length ? Math.min(...a.yearsPlayed) : Infinity
          const bYear = b.yearsPlayed.length ? Math.min(...b.yearsPlayed) : Infinity
          return aYear - bYear
        }
        default: return a.title.localeCompare(b.title)
      }
    })
  }, [games, sort])

  const handleOpenAdd = () => { setEditGame(null); setFormOpen(true) }
  const handleEdit = (game: VideoGame) => { setEditGame(game); setFormOpen(true) }
  const handleFormClose = () => { setFormOpen(false); setEditGame(null) }

  const handleSubmit = async (data: VideoGameFormData) => {
    if (editGame) {
      await updateGame(editGame.id, data)
      toaster.create({ title: 'Game updated', type: 'success', duration: 3000 })
    } else {
      await createGame(data)
      toaster.create({ title: 'Game added', type: 'success', duration: 3000 })
    }
    handleFormClose()
  }

  const handleDeleteConfirm = async () => {
    if (!deleteTarget) return
    await deleteGame(deleteTarget.id)
    toaster.create({ title: 'Game deleted', type: 'success', duration: 3000 })
    setDeleteTarget(null)
  }

  const handleClearFilters = () => {
    setSearch('')
    setYearPlayed(undefined)
    setSort('title_asc')
  }

  return (
    <Box>
      <Flex align="center" justify="space-between" mb={5}>
        <Box>
          <Heading
            fontFamily="heading" fontWeight="700" fontSize="2xl"
            color="text.primary" letterSpacing="-0.02em"
          >
            🎮 Gaming
          </Heading>
          {!loading && (
            <Text fontSize="sm" color="text.muted" mt={0.5}>
              {sortedGames.length} {sortedGames.length === 1 ? 'game' : 'games'}
            </Text>
          )}
        </Box>
        <ViewToggle view={view} onChange={setView} />
      </Flex>

      <Flex gap={6} align="flex-start">
        <Box flexShrink={0}>
          <button
            type="button"
            onClick={() => setSidebarOpen(prev => !prev)}
            aria-label={sidebarOpen ? 'Hide filters' : 'Show filters'}
            style={{
              display: 'flex', alignItems: 'center', gap: '6px',
              padding: '5px 12px', borderRadius: '7px',
              fontSize: '0.8rem', fontWeight: '500', cursor: 'pointer',
              border: '1px solid var(--chakra-colors-border-default)',
              background: 'var(--chakra-colors-bg-surface)',
              color: 'var(--chakra-colors-text-secondary)',
              transition: 'all 0.15s ease', whiteSpace: 'nowrap',
            }}
          >
            <span style={{ fontSize: '1rem', lineHeight: 1 }}>☰</span>
            {sidebarOpen ? 'Hide Filters' : 'Filters'}
          </button>

          <Box
            style={{
              width: sidebarOpen ? `${SIDEBAR_WIDTH}px` : '0px',
              opacity: sidebarOpen ? 1 : 0,
              overflow: 'hidden',
              transition: 'width 0.25s ease, opacity 0.2s ease',
              marginTop: sidebarOpen ? '10px' : 0,
            }}
          >
            <Box
              w={`${SIDEBAR_WIDTH}px`}
              bg="bg.surface"
              border="1px solid"
              borderColor="border.default"
              borderRadius="10px"
              p={4}
              position="sticky"
              top="72px"
            >
              <GameFilters
                search={search}
                onSearch={setSearch}
                yearPlayed={yearPlayed}
                onYearPlayed={setYearPlayed}
                sort={sort}
                onSort={setSort}
                onAdd={handleOpenAdd}
                onClear={handleClearFilters}
                hasFilters={hasFilters}
              />
            </Box>
          </Box>
        </Box>

        <Box flex={1} minW={0}>
          {loading ? (
            <Flex justify="center" py={24}><Spinner color="accent" size="lg" /></Flex>
          ) : error ? (
            <Flex justify="center" py={16}><Text color="red.500" fontSize="sm">{error}</Text></Flex>
          ) : sortedGames.length === 0 ? (
            <EmptyState
              noun="game"
              icon="🎮"
              hasFilters={hasFilters}
              onAdd={handleOpenAdd}
              onClear={handleClearFilters}
            />
          ) : view === 'grid' ? (
            <SimpleGrid columns={{ base: 3, sm: 4, md: sidebarOpen ? 4 : 5, lg: sidebarOpen ? 5 : 6, xl: sidebarOpen ? 6 : 7 }} gap={6}>
              {sortedGames.map(game => (
                <GameCard key={game.id} game={game} onClick={setViewGame} />
              ))}
            </SimpleGrid>
          ) : (
            <Flex direction="column" gap={2}>
              {sortedGames.map(game => (
                <GameListItem key={game.id} game={game} onClick={setViewGame} />
              ))}
            </Flex>
          )}
        </Box>
      </Flex>

      <GameDetailModal
        game={viewGame}
        isOpen={!!viewGame}
        onClose={() => setViewGame(null)}
        onEdit={handleEdit}
        onDelete={setDeleteTarget}
      />

      <GameForm
        isOpen={formOpen}
        onClose={handleFormClose}
        onSubmit={handleSubmit}
        editGame={editGame}
      />

      <ConfirmDialog
        isOpen={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDeleteConfirm}
        title="Delete game"
        message={`Are you sure you want to delete "${deleteTarget?.title}"? This action cannot be undone.`}
      />
    </Box>
  )
}
