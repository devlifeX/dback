import type { ApiError } from './types'

const SESSION_KEY = 'dback_session_token'

let bearerToken: string | null = null

export function initAuth() {
  bearerToken = sessionStorage.getItem(SESSION_KEY)
}

export function setSessionToken(token: string) {
  bearerToken = token.trim()
  sessionStorage.setItem(SESSION_KEY, bearerToken)
}

export function clearSessionToken() {
  bearerToken = null
  sessionStorage.removeItem(SESSION_KEY)
}

export function getSessionToken() {
  return bearerToken ?? sessionStorage.getItem(SESSION_KEY)
}

export function hasSession() {
  return Boolean(getSessionToken())
}

/** @deprecated use setSessionToken */
export function setApiToken(token: string) {
  setSessionToken(token)
}

/** @deprecated use clearSessionToken */
export function clearApiToken() {
  clearSessionToken()
}

/** @deprecated use hasSession */
export function hasApiToken() {
  return hasSession()
}

/** @deprecated use session token */
export function getApiToken() {
  return bearerToken ?? sessionStorage.getItem(SESSION_KEY)
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
  skipAuthRedirect?: boolean
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
  const token = bearerToken ?? sessionStorage.getItem(SESSION_KEY)
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }
  return headers
}

function handleUnauthorized(path: string, skipAuthRedirect?: boolean) {
  if (skipAuthRedirect || path.includes('/auth/login') || path.includes('/auth/verify-otp')) {
    return
  }
  clearSessionToken()
  if (typeof window !== 'undefined' && !window.location.pathname.startsWith('/login')) {
    window.location.assign('/login')
  }
}

export async function fetchRevision(): Promise<{ revision: number; etag: string }> {
  const res = await fetch('/api/v1/system/revision', { headers: authHeaders() })
  const text = await res.text()
  const data = text ? JSON.parse(text) : null
  if (!res.ok) {
    if (res.status === 401) {
      handleUnauthorized('/api/v1/system/revision')
    }
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
  const method = opts.method ?? (opts.body !== undefined ? 'POST' : 'GET')
  const isWrite = method === 'POST' || method === 'PUT' || method === 'PATCH'
  const headers = authHeaders()
  if (isWrite) {
    headers['Content-Type'] = 'application/json'
  }
  if (opts.etag) {
    headers['If-Match'] = opts.etag
  }

  const body = opts.body !== undefined ? JSON.stringify(opts.body) : isWrite ? '{}' : undefined

  const res = await fetch(path, {
    method,
    headers,
    body,
    signal: opts.signal,
  })

  if (res.status === 204) {
    return { data: undefined as T, etag: responseEtag(res) }
  }

  const text = await res.text()
  const data = text ? JSON.parse(text) : null

  if (!res.ok) {
    if (res.status === 401) {
      handleUnauthorized(path, opts.skipAuthRedirect)
    }
    throw new ApiClientError(res.status, data ?? { code: 'unknown', message: res.statusText })
  }
  return { data: data as T, etag: responseEtag(res) }
}

export async function apiDownload(path: string): Promise<Blob> {
  const res = await fetch(path, { headers: authHeaders() })
  if (!res.ok) {
    if (res.status === 401) {
      handleUnauthorized(path)
    }
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
