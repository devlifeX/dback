import { apiRequest } from './client'
import type { Paginated, RemoteDestination } from './types'

export const destinationsApi = {
  list: () => apiRequest<Paginated<RemoteDestination>>('/api/v1/destinations'),
  save: (d: RemoteDestination, etag?: string) =>
    apiRequest<RemoteDestination>(d.id ? `/api/v1/destinations/${d.id}` : '/api/v1/destinations', {
      method: d.id ? 'PUT' : 'POST',
      body: d,
      etag,
    }),
  remove: (id: string, etag?: string) =>
    apiRequest<void>(`/api/v1/destinations/${id}`, { method: 'DELETE', etag }),
  test: (id: string) =>
    apiRequest<{ status: string }>(`/api/v1/destinations/${id}/test`, { method: 'POST' }),
}

export const syncApi = {
  getSettings: () => apiRequest<Record<string, unknown>>('/api/v1/sync/settings'),
  saveSettings: (settings: Record<string, unknown>, etag?: string) =>
    apiRequest<Record<string, unknown>>('/api/v1/sync/settings', {
      method: 'PUT',
      body: settings,
      etag,
    }),
  push: () => apiRequest<{ status: string }>('/api/v1/sync/push', { method: 'POST' }),
  pull: () => apiRequest<{ bytes: number }>('/api/v1/sync/pull', { method: 'POST' }),
}
