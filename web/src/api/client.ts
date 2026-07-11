import type { ApiError } from './types'

let bearerToken: string | null = null

export function setApiToken(token: string) {
  bearerToken = token.trim()
}

export function clearApiToken() {
  bearerToken = null
}

export function hasApiToken() {
  return Boolean(bearerToken)
}

export function getApiToken() {
  return bearerToken
}

export class ApiClientError extends Error {
  code: string
  status: number

  constructor(status: number, body: ApiError) {
    super(body.message || 'Request failed')
    this.code = body.code
    this.status = status
  }
}

type RequestOptions = {
  method?: string
  body?: unknown
  etag?: string
  signal?: AbortSignal
}

export async function apiRequest<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const headers: Record<string, string> = {
    Accept: 'application/json',
  }
  if (bearerToken) {
    headers.Authorization = `Bearer ${bearerToken}`
  }
  if (opts.body !== undefined) {
    headers['Content-Type'] = 'application/json'
  }
  if (opts.etag) {
    headers['If-Match'] = opts.etag
  }

  const res = await fetch(path, {
    method: opts.method ?? (opts.body !== undefined ? 'POST' : 'GET'),
    headers,
    body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
    signal: opts.signal,
  })

  if (res.status === 204) {
    return undefined as T
  }

  const text = await res.text()
  const data = text ? JSON.parse(text) : null

  if (!res.ok) {
    throw new ApiClientError(res.status, data ?? { code: 'unknown', message: res.statusText })
  }
  return data as T
}

export function apiETag(res: Response): string | undefined {
  return res.headers.get('ETag') ?? undefined
}
