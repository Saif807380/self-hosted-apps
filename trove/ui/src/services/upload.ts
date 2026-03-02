export async function uploadFile(file: File): Promise<string> {
  const form = new FormData()
  form.append('file', file)

  const res = await fetch('/api/v1/uploads', { method: 'POST', body: form })
  if (!res.ok) throw new Error('Upload failed')

  const data: { path: string } = await res.json()
  return data.path
}
