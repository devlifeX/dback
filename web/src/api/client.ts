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

export type ApiResponse<T> = {
  data: T
  etag?: string
}

export function responseEtag(res: Response): string | undefined {
  return res.headers.get('ETag') ?? undefined
}

/** @deprecated use responseEtag */
export function apiETag(res: Response): string | undefined {
  return responseEtag(res)
}

export function etagForRevision(revision: number): string {
  return `W/"${revision}"`
}

function authHeaders(extra: Record<string, string> = {}): Record<string, string> {
  const headers: Record<string, string> = { Accept: 'application/json', ...extra }
  if (bearerToken) {
    headers.Authorization = `Bearer ${bearerToken}`
  }
  return headers
}

export async function fetchRevision(): Promise<{ revision: number; etag: string }> {
  const res = await fetch('/api/v1/system/revision', { headers: authHeaders() })
  const text = await res.text()
  const data = text ? JSON.parse(text) : null
  if (!res.ok) {
    throw new ApiClientError(res.status, data ?? { code: 'unknown', message: res.statusText })
  }
  const revision = Number(data?.revision ?? 0)
  const etag = responseEtag(res) ?? etagForRevision(revision)
  return { revision, etag }
}

export async function apiRequest<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const result = await apiRequestWithMeta<T>(path, opts)
  return result.data
}

export async function apiRequestWithMeta<T>(path: string, opts: RequestOptions = {}): Promise<ApiResponse<T>> {
  const headers = authHeaders()
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
    return { data: undefined as T, etag: responseEtag(res) }
  }

  const text = await res.text()
  const data = text ? JSON.parse(text) : null

  if (!res.ok) {
    throw new ApiClientError(res.status, data ?? { code: 'unknown', message: res.statusText })
  }
  return { data: data as T, etag: responseEtag(res) }
}

export async function apiDownload(path: string): Promise<Blob> {
  const res = await fetch(path, { headers: authHeaders() })
  if (!res.ok) {
    const text = await res.text()
    let data: ApiError = { code: 'unknown', message: res.statusText }
    try {
      data = text ? JSON.parse(text) : data
    } catch {
      /* binary error body */
    }
    throw new ApiClientError(res.status, data)
  }
  return res.blob()
}
