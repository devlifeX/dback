import { apiRequest } from './client'
import type { Paginated, SQLTemplate } from './types'

export const templatesApi = {
  list: () => apiRequest<Paginated<SQLTemplate>>('/api/v1/templates'),
  save: (t: SQLTemplate, etag?: string) =>
    apiRequest<SQLTemplate>(t.id ? `/api/v1/templates/${t.id}` : '/api/v1/templates', {
      method: t.id ? 'PUT' : 'POST',
      body: t,
      etag,
    }),
  remove: (id: string, etag?: string) =>
    apiRequest<void>(`/api/v1/templates/${id}`, { method: 'DELETE', etag }),
}
