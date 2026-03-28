import { db } from '@/lib/db'
import { sha256 } from '@/lib/hash'

export async function uploadFile(file: File): Promise<string> {
  const id = crypto.randomUUID()
  const lastDot = file.name.lastIndexOf('.')
  const ext = lastDot > 0 ? file.name.substring(lastDot) : ''
  const checksum = await sha256(file)

  await db.imageBlobs.put({ id, ext, blob: file, checksum, uploaded: false })

  return `/uploads/${id}${ext}`
}
