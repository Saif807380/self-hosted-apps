import { useState, useMemo } from 'react'
import { Box, Flex, Heading, SimpleGrid, Text, Spinner } from '@chakra-ui/react'
import { useBooks } from '@/hooks/useBooks'
import { useTags } from '@/hooks/useTags'
import { useDebounce } from '@/hooks/useDebounce'
import { toaster } from '@/lib/toaster'
import type { Book, BookFormData } from '@/types/api'
import BookCard from './books/BookCard'
import BookListItem from './books/BookListItem'
import BookFilters from './books/BookFilters'
import BookForm from './books/BookForm'
import BookDetailModal from './books/BookDetailModal'
import EmptyState from '@/components/EmptyState'
import ConfirmDialog from '@/components/ConfirmDialog'
import ViewToggle from '@/components/ViewToggle'

const SIDEBAR_WIDTH = 240

export default function BooksPage() {
  const [search, setSearch] = useState('')
  const [yearRead, setYearRead] = useState<number | undefined>()
  const [tagIds, setTagIds] = useState<string[]>([])
  const [sort, setSort] = useState('title_asc')
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const [formOpen, setFormOpen] = useState(false)
  const [editBook, setEditBook] = useState<Book | null>(null)
  const [viewBook, setViewBook] = useState<Book | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Book | null>(null)
  const [view, setView] = useState<'grid' | 'list'>('grid')

  const debouncedSearch = useDebounce(search, 300)
  const { books, loading, error, createBook, updateBook, deleteBook } = useBooks({
    search: debouncedSearch,
    yearRead,
  })
  const { tags, createTag } = useTags()

  const hasFilters = !!search || tagIds.length > 0 || !!yearRead

  const sortedBooks = useMemo(() => {
    const filtered = tagIds.length > 0
      ? books.filter(b => tagIds.some(id => b.tags.some(t => t.id === id)))
      : books
    return [...filtered].sort((a, b) => {
      switch (sort) {
        case 'title_desc': return b.title.localeCompare(a.title)
        case 'year_desc': {
          const aYear = a.yearsRead.length ? Math.max(...a.yearsRead) : -1
          const bYear = b.yearsRead.length ? Math.max(...b.yearsRead) : -1
          return bYear - aYear
        }
        case 'year_asc': {
          const aYear = a.yearsRead.length ? Math.min(...a.yearsRead) : Infinity
          const bYear = b.yearsRead.length ? Math.min(...b.yearsRead) : Infinity
          return aYear - bYear
        }
        default: return a.title.localeCompare(b.title) // title_asc
      }
    })
  }, [books, sort, tagIds])

  const handleOpenAdd = () => {
    setEditBook(null)
    setFormOpen(true)
  }

  const handleEdit = (book: Book) => {
    setEditBook(book)
    setFormOpen(true)
  }

  const handleFormClose = () => {
    setFormOpen(false)
    setEditBook(null)
  }

  const handleSubmit = async (data: BookFormData) => {
    if (editBook) {
      await updateBook(editBook.id, data)
      toaster.create({ title: 'Book updated', type: 'success', duration: 3000 })
    } else {
      await createBook(data)
      toaster.create({ title: 'Book added', type: 'success', duration: 3000 })
    }
    handleFormClose()
  }

  const handleDeleteConfirm = async () => {
    if (!deleteTarget) return
    await deleteBook(deleteTarget.id)
    toaster.create({ title: 'Book deleted', type: 'success', duration: 3000 })
    setDeleteTarget(null)
  }

  const handleClearFilters = () => {
    setSearch('')
    setYearRead(undefined)
    setTagIds([])
    setSort('title_asc')
  }

  return (
    <Box>
      {/* Page header */}
      <Flex align="center" justify="space-between" mb={5}>
        <Box>
          <Heading
            fontFamily="heading"
            fontWeight="700"
            fontSize="2xl"
            color="text.primary"
            letterSpacing="-0.02em"
          >
            📚 Books
          </Heading>
          {!loading && (
            <Text fontSize="sm" color="text.muted" mt={0.5}>
              {sortedBooks.length} {sortedBooks.length === 1 ? 'book' : 'books'}
            </Text>
          )}
        </Box>
        <ViewToggle view={view} onChange={setView} />
      </Flex>

      {/* Main layout: sidebar column + grid */}
      <Flex gap={6} align="flex-start">
        {/* Sidebar column — toggle always visible, panel collapsible */}
        <Box flexShrink={0}>
          {/* Toggle button lives here, with the filters */}
          <button
            type="button"
            onClick={() => setSidebarOpen(prev => !prev)}
            aria-label={sidebarOpen ? 'Hide filters' : 'Show filters'}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
              padding: '5px 12px',
              borderRadius: '7px',
              fontSize: '0.8rem',
              fontWeight: '500',
              cursor: 'pointer',
              border: '1px solid var(--chakra-colors-border-default)',
              background: 'var(--chakra-colors-bg-surface)',
              color: 'var(--chakra-colors-text-secondary)',
              transition: 'all 0.15s ease',
              whiteSpace: 'nowrap',
            }}
          >
            <span style={{ fontSize: '1rem', lineHeight: 1 }}>☰</span>
            {sidebarOpen ? 'Hide Filters' : 'Filters'}
          </button>

          {/* Collapsible panel */}
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
              <BookFilters
                search={search}
                onSearch={setSearch}
                yearRead={yearRead}
                onYearRead={setYearRead}
                tagIds={tagIds}
                onTagIds={setTagIds}
                sort={sort}
                onSort={setSort}
                tags={tags}
                onAdd={handleOpenAdd}
                onClear={handleClearFilters}
                hasFilters={hasFilters}
              />
            </Box>
          </Box>
        </Box>

        {/* Book grid */}
        <Box flex={1} minW={0}>
          {loading ? (
            <Flex justify="center" py={24}>
              <Spinner color="accent" size="lg" />
            </Flex>
          ) : error ? (
            <Flex justify="center" py={16}>
              <Text color="red.500" fontSize="sm">{error}</Text>
            </Flex>
          ) : sortedBooks.length === 0 ? (
            <EmptyState
              hasFilters={hasFilters}
              onAdd={handleOpenAdd}
              onClear={handleClearFilters}
            />
          ) : view === 'grid' ? (
            <SimpleGrid columns={{ base: 3, sm: 4, md: sidebarOpen ? 4 : 5, lg: sidebarOpen ? 5 : 6, xl: sidebarOpen ? 6 : 7 }} gap={6}>
              {sortedBooks.map(book => (
                <BookCard key={book.id} book={book} onClick={setViewBook} />
              ))}
            </SimpleGrid>
          ) : (
            <Flex direction="column" gap={2}>
              {sortedBooks.map(book => (
                <BookListItem key={book.id} book={book} onClick={setViewBook} />
              ))}
            </Flex>
          )}
        </Box>
      </Flex>

      {/* Detail modal — read-only view */}
      <BookDetailModal
        book={viewBook}
        isOpen={!!viewBook}
        onClose={() => setViewBook(null)}
        onEdit={handleEdit}
        onDelete={setDeleteTarget}
      />

      {/* Add / Edit form */}
      <BookForm
        isOpen={formOpen}
        onClose={handleFormClose}
        onSubmit={handleSubmit}
        tags={tags}
        onCreateTag={createTag}
        editBook={editBook}
      />

      {/* Delete confirmation */}
      <ConfirmDialog
        isOpen={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDeleteConfirm}
        title="Delete book"
        message={`Are you sure you want to delete "${deleteTarget?.title}"? This action cannot be undone.`}
      />
    </Box>
  )
}
