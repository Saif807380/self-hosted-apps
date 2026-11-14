import { useState, useMemo, useEffect } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Box, Button, Flex, Heading, SimpleGrid, Text, Spinner } from '@chakra-ui/react'
import { useGames } from '@/hooks/useGames'
import { useDebounce } from '@/hooks/useDebounce'
import { useIsMobile } from '@/hooks/useIsMobile'
import { toaster } from '@/lib/toaster'
import type { GameStatus, VideoGame, VideoGameFormData } from '@/types/api'
import GameCard from './games/GameCard'
import GameListItem from './games/GameListItem'
import GameFilters from './games/GameFilters'
import GameForm from './games/GameForm'
import GameDetailModal from './games/GameDetailModal'
import SaleAlertsButton from './games/SaleAlertsButton'
import EmptyState from '@/components/EmptyState'
import ConfirmDialog from '@/components/ConfirmDialog'
import ViewToggle from '@/components/ViewToggle'
import { STATUS_LABELS, STATUS_TABS, isGameStatus } from './games/lifecycle'

const SIDEBAR_WIDTH = 240

const defaultSort = (tab: GameStatus) => (tab === 'played' ? 'title_asc' : 'added_desc')

export default function GamesPage() {
  const isMobile = useIsMobile()
  // ?tab= picks the list and ?game= opens a game, so links (e.g. from a sale
  // notification) can land on a specific game.
  const [params, setParams] = useSearchParams()
  const tabParam = params.get('tab')
  const tab: GameStatus = isGameStatus(tabParam) ? tabParam : 'played'
  const [search, setSearch] = useState('')
  const [yearPlayed, setYearPlayed] = useState<number | undefined>()
  const [sort, setSort] = useState(() => defaultSort(tab))
  const [sidebarOpen, setSidebarOpen] = useState(() => !window.matchMedia('(max-width: 767px)').matches)
  const [formOpen, setFormOpen] = useState(false)
  const [editGame, setEditGame] = useState<VideoGame | null>(null)
  const [viewGame, setViewGame] = useState<VideoGame | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<VideoGame | null>(null)
  const [view, setView] = useState<'grid' | 'list'>(() => window.matchMedia('(max-width: 767px)').matches ? 'list' : 'grid')

  const debouncedSearch = useDebounce(search, 300)
  const { games, counts, loading, error, createGame, updateGame, deleteGame, setStatus } = useGames({
    search: debouncedSearch,
    yearPlayed: tab === 'played' ? yearPlayed : undefined,
    status: tab,
  })

  const hasFilters = !!search || !!yearPlayed

  const handleTabChange = (next: GameStatus) => {
    setParams({ tab: next })
    setYearPlayed(undefined)
    setSort(defaultSort(next))
  }

  const linkedGameId = params.get('game')
  useEffect(() => {
    if (!linkedGameId || loading) return
    const linked = games.find(g => g.id === linkedGameId)
    if (linked) setViewGame(linked)
    setParams({ tab }, { replace: true })
  }, [linkedGameId, loading, games, tab, setParams])

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
        case 'added_desc': return b.createdAt.localeCompare(a.createdAt)
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

  const handleStatusChange = async (game: VideoGame, status: GameStatus) => {
    await setStatus(game.id, status)
    toaster.create({ title: `Moved to ${STATUS_LABELS[status]}`, type: 'success', duration: 3000 })
  }

  // Finishing goes through the form so a rating and the year can be added.
  const handleFinish = (game: VideoGame) => {
    const year = new Date().getFullYear()
    const yearsPlayed = game.yearsPlayed.includes(year) ? game.yearsPlayed : [...game.yearsPlayed, year]
    handleEdit({ ...game, status: 'played', yearsPlayed })
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
    setSort(defaultSort(tab))
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
        <Flex gap={2} align="center">
          <SaleAlertsButton compact={isMobile} />
          <ViewToggle view={view} onChange={setView} />
        </Flex>
      </Flex>

      <Flex gap={1.5} mb={5} wrap="wrap" role="tablist" aria-label="Game lists">
        {STATUS_TABS.map(t => (
          <Button
            key={t.status}
            size="sm"
            role="tab"
            aria-selected={tab === t.status}
            variant={tab === t.status ? 'solid' : 'outline'}
            borderRadius="8px"
            onClick={() => handleTabChange(t.status)}
          >
            {t.icon} {t.label}
            <Text as="span" fontSize="xs" opacity={0.7} ml={1}>{counts[t.status]}</Text>
          </Button>
        ))}
      </Flex>

      <Flex gap={6} align="flex-start" direction={isMobile ? 'column' : 'row'}>
        <Box flexShrink={0} w={isMobile ? '100%' : 'auto'}>
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
              width: isMobile
                ? sidebarOpen ? '100%' : '0px'
                : sidebarOpen ? `${SIDEBAR_WIDTH}px` : '0px',
              height: isMobile && !sidebarOpen ? '0px' : 'auto',
              opacity: sidebarOpen ? 1 : 0,
              overflow: 'hidden',
              transition: 'width 0.25s ease, height 0.25s ease, opacity 0.2s ease',
              marginTop: sidebarOpen ? '10px' : 0,
            }}
          >
            <Box
              w={isMobile ? '100%' : `${SIDEBAR_WIDTH}px`}
              bg="bg.surface"
              border="1px solid"
              borderColor="border.default"
              borderRadius="10px"
              p={4}
              position={isMobile ? 'static' : 'sticky'}
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
                showYears={tab === 'played'}
              />
            </Box>
          </Box>
        </Box>

        <Box flex={1} minW={0} w={isMobile ? '100%' : 'auto'}>
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
            <SimpleGrid columns={{ base: 2, sm: 3, md: sidebarOpen ? 4 : 5, lg: sidebarOpen ? 5 : 6, xl: sidebarOpen ? 6 : 7 }} gap={{ base: 3, md: 6 }}>
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
        onStatusChange={handleStatusChange}
        onFinish={handleFinish}
      />

      <GameForm
        isOpen={formOpen}
        onClose={handleFormClose}
        onSubmit={handleSubmit}
        editGame={editGame}
        defaultStatus={tab}
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
