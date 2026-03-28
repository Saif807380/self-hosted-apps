import { db } from './db'

interface ManifestEntry {
  id: string
  path: string
  checksum: string
  size: number
}

// Push locally-stored blobs that haven't been uploaded yet
export async function pushImages(): Promise<void> {
  const pending = await db.imageBlobs.where('uploaded').equals(0).toArray()
  console.debug('[image-sync] push: pending =', pending.length)

  for (const record of pending) {
    const form = new FormData()
    form.append('id', record.id)
    form.append('checksum', record.checksum)
    form.append('file', record.blob, `${record.id}${record.ext}`)

    const res = await fetch('/api/v1/sync/images', { method: 'POST', body: form })
    if (!res.ok) {
      console.warn('[image-sync] upload failed for', record.id, res.status)
      continue
    }

    await db.imageBlobs.update(record.id, { uploaded: true })
    console.debug('[image-sync] uploaded', record.id)
  }
}

// Pull images from server that we don't have locally
export async function pullImages(): Promise<void> {
  const res = await fetch('/api/v1/sync/images/manifest')
  if (!res.ok) {
    console.warn('[image-sync] manifest fetch failed', res.status)
    return
  }

  const manifest: ManifestEntry[] = await res.json()
  console.debug('[image-sync] pull: manifest entries =', manifest.length)

  for (const entry of manifest) {
    const existing = await db.imageBlobs.get(entry.id)
    if (existing) continue

    const imgRes = await fetch(entry.path)
    if (!imgRes.ok) {
      console.warn('[image-sync] failed to download', entry.path)
      continue
    }

    const blob = await imgRes.blob()
    const lastDot = entry.path.lastIndexOf('.')
    const ext = lastDot > 0 ? entry.path.substring(lastDot) : ''

    await db.imageBlobs.put({
      id: entry.id,
      ext,
      blob,
      checksum: entry.checksum,
      uploaded: true,
    })
    console.debug('[image-sync] downloaded', entry.id)
  }
}
