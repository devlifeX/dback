import { ApiClientError, apiRequest, getApiToken } from './client'

export type ImportPreview = {
  profiles_count: number
  templates_count: number
  history_count: number
  profile_conflicts: unknown[] | null
  template_conflicts: unknown[] | null
}

export type ImportFileOptions = {
  file: File
  passphrase?: string
  includeSecrets?: boolean
}

async function importFileRequest<T>(path: string, opts: ImportFileOptions, etag?: string): Promise<T> {
  const form = new FormData()
  form.append('file', opts.file)
  if (opts.passphrase) form.append('passphrase', opts.passphrase)
  if (opts.includeSecrets) form.append('include_secrets', 'true')

  const headers: Record<string, string> = { Accept: 'application/json' }
  const token = getApiToken()
  if (token) headers.Authorization = `Bearer ${token}`
  if (etag) headers['If-Match'] = etag

  const res = await fetch(path, { method: 'POST', headers, body: form })
  const text = await res.text()
  const data = text ? JSON.parse(text) : null
  if (!res.ok) {
    throw new ApiClientError(res.status, data ?? { code: 'unknown', message: res.statusText })
  }
  return data as T
}

export const vaultApi = {
  exportAppData: async (opts?: { passphrase?: string; includeSecrets?: boolean }) => {
    const q = new URLSearchParams()
    if (opts?.passphrase) q.set('passphrase', opts.passphrase)
    if (opts?.includeSecrets) q.set('include_secrets', 'true')
    const qs = q.toString()
    const headers: Record<string, string> = {}
    const token = getApiToken()
    if (token) headers.Authorization = `Bearer ${token}`
    const res = await fetch(`/api/v1/export/app-data${qs ? `?${qs}` : ''}`, { headers })
    if (!res.ok) throw new Error('Export failed')
    return res.blob()
  },
  importPreview: (body: { passphrase?: string; include_secrets?: boolean; content_base64?: string }) =>
    apiRequest<ImportPreview>('/api/v1/import/preview', { method: 'POST', body }),
  importApply: (body: { passphrase?: string; include_secrets?: boolean; content_base64?: string }, etag?: string) =>
    apiRequest<{ status: string }>('/api/v1/import/apply', { method: 'POST', body, etag }),
  importFilePreview: (opts: ImportFileOptions) =>
    importFileRequest<ImportPreview>('/api/v1/import/preview', opts),
  importFileApply: (opts: ImportFileOptions, etag?: string) =>
    importFileRequest<{ status: string }>('/api/v1/import/apply', opts, etag),
}
