import { useState, useMemo } from 'react'
import { Box, Flex, Heading, SimpleGrid, Text, Spinner } from '@chakra-ui/react'
import { useTravel } from '@/hooks/useTravel'
import { useDebounce } from '@/hooks/useDebounce'
import { toaster } from '@/lib/toaster'
import type { TravelLocation, TravelLocationFormData } from '@/types/api'
import LocationCard from './travel/LocationCard'
import LocationListItem from './travel/LocationListItem'
import TravelFilters from './travel/TravelFilters'
import LocationForm from './travel/LocationForm'
import LocationDetailModal from './travel/LocationDetailModal'
import EmptyState from '@/components/EmptyState'
import ConfirmDialog from '@/components/ConfirmDialog'
import ViewToggle from '@/components/ViewToggle'

const SIDEBAR_WIDTH = 240

export default function TravelPage() {
  const [search, setSearch] = useState('')
  const [sort, setSort] = useState('city_asc')
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const [formOpen, setFormOpen] = useState(false)
  const [editLocation, setEditLocation] = useState<TravelLocation | null>(null)
  const [viewLocationId, setViewLocationId] = useState<string | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<TravelLocation | null>(null)
  const [view, setView] = useState<'grid' | 'list'>('grid')

  const debouncedSearch = useDebounce(search, 300)
  const {
    locations, loading, error,
    createLocation, updateLocation, deleteLocation,
    createSpot, deleteSpot,
  } = useTravel({ search: debouncedSearch })

  // Always look up the live location so the detail modal reflects spot changes
  const viewLocation = viewLocationId
    ? locations.find(l => l.id === viewLocationId) ?? null
    : null

  const hasFilters = !!search

  const sortedLocations = useMemo(() => {
    return [...locations].sort((a, b) => {
      switch (sort) {
        case 'city_desc': return b.city.localeCompare(a.city)
        case 'date_desc': {
          const aDate = a.visitedFrom ?? ''
          const bDate = b.visitedFrom ?? ''
          return bDate.localeCompare(aDate)
        }
        case 'date_asc': {
          const aDate = a.visitedFrom ?? 'zzzz'
          const bDate = b.visitedFrom ?? 'zzzz'
          return aDate.localeCompare(bDate)
        }
        default: return a.city.localeCompare(b.city)
      }
    })
  }, [locations, sort])

  const handleOpenAdd = () => { setEditLocation(null); setFormOpen(true) }
  const handleEdit = (location: TravelLocation) => { setEditLocation(location); setFormOpen(true) }
  const handleFormClose = () => { setFormOpen(false); setEditLocation(null) }

  const handleSubmit = async (data: TravelLocationFormData) => {
    if (editLocation) {
      await updateLocation(editLocation.id, data)
      toaster.create({ title: 'Location updated', type: 'success', duration: 3000 })
    } else {
      await createLocation(data)
      toaster.create({ title: 'Location added', type: 'success', duration: 3000 })
    }
    handleFormClose()
  }

  const handleDeleteConfirm = async () => {
    if (!deleteTarget) return
    await deleteLocation(deleteTarget.id)
    toaster.create({ title: 'Location deleted', type: 'success', duration: 3000 })
    setDeleteTarget(null)
    if (viewLocationId === deleteTarget.id) setViewLocationId(null)
  }

  const handleClearFilters = () => {
    setSearch('')
    setSort('city_asc')
  }

  return (
    <Box>
      <Flex align="center" justify="space-between" mb={5}>
        <Box>
          <Heading
            fontFamily="heading" fontWeight="700" fontSize="2xl"
            color="text.primary" letterSpacing="-0.02em"
          >
            ✈️ Travel
          </Heading>
          {!loading && (
            <Text fontSize="sm" color="text.muted" mt={0.5}>
              {sortedLocations.length} {sortedLocations.length === 1 ? 'location' : 'locations'}
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
              <TravelFilters
                search={search}
                onSearch={setSearch}
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
          ) : sortedLocations.length === 0 ? (
            <EmptyState
              noun="location"
              icon="✈️"
              hasFilters={hasFilters}
              onAdd={handleOpenAdd}
              onClear={handleClearFilters}
            />
          ) : view === 'grid' ? (
            <SimpleGrid columns={{ base: 2, sm: 3, md: sidebarOpen ? 3 : 4, lg: sidebarOpen ? 4 : 5, xl: sidebarOpen ? 5 : 6 }} gap={6}>
              {sortedLocations.map(location => (
                <LocationCard key={location.id} location={location} onClick={l => setViewLocationId(l.id)} />
              ))}
            </SimpleGrid>
          ) : (
            <Flex direction="column" gap={2}>
              {sortedLocations.map(location => (
                <LocationListItem key={location.id} location={location} onClick={l => setViewLocationId(l.id)} />
              ))}
            </Flex>
          )}
        </Box>
      </Flex>

      <LocationDetailModal
        location={viewLocation}
        isOpen={!!viewLocationId}
        onClose={() => setViewLocationId(null)}
        onEdit={handleEdit}
        onDelete={setDeleteTarget}
        onAddSpot={createSpot}
        onDeleteSpot={deleteSpot}
      />

      <LocationForm
        isOpen={formOpen}
        onClose={handleFormClose}
        onSubmit={handleSubmit}
        editLocation={editLocation}
      />

      <ConfirmDialog
        isOpen={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDeleteConfirm}
        title="Delete location"
        message={`Are you sure you want to delete "${deleteTarget?.city}"? This action cannot be undone.`}
      />
    </Box>
  )
}
