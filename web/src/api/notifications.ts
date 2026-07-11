import { apiRequest } from './client'
import type { NotifyChannel, Paginated } from './types'

export const notificationsApi = {
  list: () => apiRequest<Paginated<NotifyChannel>>('/api/v1/notifications'),
  save: (ch: NotifyChannel, etag?: string) =>
    apiRequest<NotifyChannel>(ch.id ? `/api/v1/notifications/${ch.id}` : '/api/v1/notifications', {
      method: ch.id ? 'PUT' : 'POST',
      body: ch,
      etag,
    }),
  remove: (id: string, etag?: string) =>
    apiRequest<void>(`/api/v1/notifications/${id}`, { method: 'DELETE', etag }),
  test: (id: string) =>
    apiRequest<{ status: string }>(`/api/v1/notifications/${id}/test`, { method: 'POST' }),
}
