import { useState, useEffect } from 'react'
import { db } from '@/lib/db'

// Resolves a cover image path to either a local object URL (from IndexedDB)
// or the original server path. Revokes object URLs on unmount.
export function useImageUrl(path: string | undefined): string | undefined {
  const [url, setUrl] = useState<string | undefined>(path)

  useEffect(() => {
    if (!path) {
      setUrl(undefined)
      return
    }

    let objectUrl: string | undefined
    let cancelled = false

    const filename = path.split('/').pop() ?? ''
    const lastDot = filename.lastIndexOf('.')
    const id = lastDot > 0 ? filename.substring(0, lastDot) : filename

    db.imageBlobs.get(id).then(record => {
      if (cancelled) return
      if (record) {
        objectUrl = URL.createObjectURL(record.blob)
        setUrl(objectUrl)
      } else {
        setUrl(path)
      }
    })

    return () => {
      cancelled = true
      if (objectUrl) URL.revokeObjectURL(objectUrl)
    }
  }, [path])

  return url
}
