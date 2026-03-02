import { useState, useEffect, useCallback } from 'react'
import { booksApi } from '@/services/books'
import type { Tag } from '@/types/api'

export function useTags() {
  const [tags, setTags] = useState<Tag[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    booksApi
      .listTags()
      .then(setTags)
      .catch(console.error)
      .finally(() => setLoading(false))
  }, [])

  const createTag = useCallback(async (name: string): Promise<Tag> => {
    const tag = await booksApi.createTag(name)
    setTags(prev => [...prev, tag].sort((a, b) => a.name.localeCompare(b.name)))
    return tag
  }, [])

  return { tags, loading, createTag }
}
