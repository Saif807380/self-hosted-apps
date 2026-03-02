const BASE = '/api'

export class ApiError extends Error {
  constructor(
    public code: string,
    message: string,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

export async function rpc<Res>(service: string, method: string, body: unknown): Promise<Res> {
  const res = await fetch(`${BASE}/${service}/${method}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Connect-Protocol-Version': '1',
    },
    body: JSON.stringify(body),
  })

  const data = await res.json().catch(() => ({ message: 'Server error' }))

  if (!res.ok) {
    throw new ApiError(data.code ?? 'internal', data.message ?? `Request failed (${res.status})`)
  }

  return data as Res
}
