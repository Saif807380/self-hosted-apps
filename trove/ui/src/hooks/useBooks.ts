import { useCallback } from 'react'
import { useLiveQuery } from 'dexie-react-hooks'
import { db } from '@/lib/db'
import {
  createBook as createBookOp,
  updateBook as updateBookOp,
  deleteBook as deleteBookOp,
} from '@/lib/operations'
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
  const raw = useLiveQuery(async () => {
    const allBooks = await db.books.filter(b => !b.deleted).toArray()
    const allYears = await db.bookYearsRead.toArray()
    const allBookTags = await db.bookTags.toArray()
    const allTags = await db.tags.filter(t => !t.deleted).toArray()
    const tagMap = new Map(allTags.map(t => [t.id, t]))

    return allBooks.map(b => ({
      id: b.id,
      title: b.title,
      author: b.author,
      rating: b.rating,
      review: b.review,
      coverImage: b.coverImage,
      createdAt: b.createdAt,
      updatedAt: b.updatedAt,
      yearsRead: allYears.filter(y => y.bookId === b.id).map(y => y.year),
      tags: allBookTags
        .filter(bt => bt.bookId === b.id)
        .map(bt => tagMap.get(bt.tagId))
        .filter(Boolean)
        .map(t => ({ id: t!.id, name: t!.name })),
    }))
  })

  // Client-side filtering
  let books: Book[] = raw ?? []
  if (search) {
    const q = search.toLowerCase()
    books = books.filter(b =>
      b.title.toLowerCase().includes(q) || b.author.toLowerCase().includes(q),
    )
  }
  if (tagId) {
    books = books.filter(b => b.tags.some(t => t.id === tagId))
  }
  if (yearRead) {
    books = books.filter(b => b.yearsRead.includes(yearRead))
  }

  const createBook = useCallback(async (data: BookFormData): Promise<Book> => {
    const rec = await createBookOp(data)
    return {
      ...rec, yearsRead: data.yearsRead,
      tags: [], // Will be populated on next live query tick
    }
  }, [])

  const updateBook = useCallback(async (id: string, data: BookFormData): Promise<Book> => {
    const rec = await updateBookOp(id, data)
    return { ...rec, yearsRead: data.yearsRead, tags: [] }
  }, [])

  const deleteBook = useCallback(async (id: string): Promise<void> => {
    await deleteBookOp(id)
  }, [])

  // refetch is a no-op with IndexedDB (useLiveQuery auto-updates)
  const refetch = useCallback(async () => {}, [])

  return { books, loading: raw === undefined, error: null, refetch, createBook, updateBook, deleteBook }
}
