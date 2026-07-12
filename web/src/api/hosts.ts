import { apiDownload, apiRequest } from './client'
import type { Host, Paginated, Profile, URLCheckHourlyBucket } from './types'

export const hostsApi = {
  list: () => apiRequest<Paginated<Host>>('/api/v1/hosts'),
  get: (id: string) => apiRequest<Host>(`/api/v1/hosts/${id}`),
  save: (host: Profile, etag?: string) =>
    apiRequest<Host>(host.id ? `/api/v1/hosts/${host.id}` : '/api/v1/hosts', {
      method: host.id ? 'PUT' : 'POST',
      body: host,
      etag,
    }),
  remove: (id: string, etag?: string) =>
    apiRequest<void>(`/api/v1/hosts/${id}`, { method: 'DELETE', etag }),
  testConnection: (id: string) =>
    apiRequest<{ status: string }>(`/api/v1/hosts/${id}/test-connection`, { method: 'POST' }),
  duplicate: (id: string, etag?: string) =>
    apiRequest<Host>(`/api/v1/hosts/${id}/duplicate`, { method: 'POST', etag }),
  pendingUploads: (id: string) =>
    apiRequest<{ count: number; details: unknown[] }>(`/api/v1/hosts/${id}/uploads/pending`),
  planUpload: (id: string, recordIds: string[]) =>
    apiRequest<Record<string, unknown>>(`/api/v1/hosts/${id}/uploads/plan`, {
      method: 'POST',
      body: { record_ids: recordIds },
    }),
  getImportDestination: (id: string) =>
    apiRequest<{ destination_profile_id: string }>(`/api/v1/hosts/${id}/import-destination`),
  setImportDestination: (id: string, destinationProfileId: string) =>
    apiRequest<{ status: string }>(`/api/v1/hosts/${id}/import-destination`, {
      method: 'PUT',
      body: { destination_profile_id: destinationProfileId },
    }),
  query: (id: string, sql: string, connectDb = false) =>
    apiRequest<Record<string, unknown>>(`/api/v1/hosts/${id}/query`, {
      method: 'POST',
      body: { sql, connect_db: connectDb },
    }),
  generateWPKey: (id: string, etag?: string) =>
    apiRequest<{ wp_key: string }>(`/api/v1/hosts/${id}/generate-wp-key`, { method: 'POST', etag }),
  downloadPlugin: (id: string) => apiDownload(`/api/v1/hosts/${id}/wordpress-plugin`),
  urlChecks: (id: string, params?: { url?: string; from?: string; to?: string }) => {
    const q = new URLSearchParams()
    if (params?.url) q.set('url', params.url)
    if (params?.from) q.set('from', params.from)
    if (params?.to) q.set('to', params.to)
    const suffix = q.toString() ? `?${q}` : ''
    return apiRequest<Paginated<URLCheckHourlyBucket>>(`/api/v1/hosts/${id}/url-checks${suffix}`)
  },
}
