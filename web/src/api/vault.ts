import { apiRequest, getApiToken } from './client'

export type ImportPreview = {
  profiles_count: number
  templates_count: number
  history_count: number
  profile_conflicts: unknown[]
  template_conflicts: unknown[]
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
}
