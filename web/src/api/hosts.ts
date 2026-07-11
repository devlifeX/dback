import { apiRequest } from './client'
import type { Host, Paginated } from './types'

export const hostsApi = {
  list: () => apiRequest<Paginated<Host>>('/api/v1/hosts'),
  get: (id: string) => apiRequest<Host>(`/api/v1/hosts/${id}`),
  save: (host: Host, etag?: string) =>
    apiRequest<Host>(host.id ? `/api/v1/hosts/${host.id}` : '/api/v1/hosts', {
      method: host.id ? 'PUT' : 'POST',
      body: host,
      etag,
    }),
  remove: (id: string, etag?: string) =>
    apiRequest<void>(`/api/v1/hosts/${id}`, { method: 'DELETE', etag }),
  testConnection: (id: string) =>
    apiRequest<{ status: string }>(`/api/v1/hosts/${id}/test-connection`, { method: 'POST' }),
}
