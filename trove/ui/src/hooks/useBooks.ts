import { useState, useEffect, useCallback } from 'react'
import { booksApi } from '@/services/books'
import type { Book, BookFormData } from '@/types/api'

export function useBooks({
  search = '',
  tagId = '',
  yearRead,
}: {
  search?: string
  tagId?: string
  yearRead?: number
} = {}) {
  const [books, setBooks] = useState<Book[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const refetch = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await booksApi.list({
        search: search || undefined,
        tagId: tagId || undefined,
        yearRead,
      })
      setBooks(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load books')
    } finally {
      setLoading(false)
    }
  }, [search, tagId, yearRead])

  useEffect(() => {
    refetch()
  }, [refetch])

  const createBook = useCallback(
    async (data: BookFormData): Promise<Book> => {
      const book = await booksApi.create(data)
      await refetch()
      return book
    },
    [refetch],
  )

  const updateBook = useCallback(
    async (id: string, data: BookFormData): Promise<Book> => {
      const book = await booksApi.update(id, data)
      await refetch()
      return book
    },
    [refetch],
  )

  const deleteBook = useCallback(
    async (id: string): Promise<void> => {
      await booksApi.delete(id)
      setBooks(prev => prev.filter(b => b.id !== id))
    },
    [],
  )

  return { books, loading, error, refetch, createBook, updateBook, deleteBook }
}
