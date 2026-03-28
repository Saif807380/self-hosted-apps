import { useCallback } from 'react'
import { useLiveQuery } from 'dexie-react-hooks'
import { db } from '@/lib/db'
import { createTag as createTagOp } from '@/lib/operations'
import type { Tag } from '@/types/api'

export function useTags() {
  const raw = useLiveQuery(() => db.tags.filter(t => !t.deleted).toArray())
  const tags: Tag[] = (raw ?? []).sort((a, b) => a.name.localeCompare(b.name))

  const createTag = useCallback(async (name: string): Promise<Tag> => {
    const tag = await createTagOp(name)
    return { id: tag.id, name: tag.name }
  }, [])

  return { tags, loading: raw === undefined, createTag }
}
