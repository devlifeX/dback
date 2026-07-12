import { apiRequest } from './client'
import type { LogEntry, Operation, Paginated } from './types'

export const operationsApi = {
  list: () => apiRequest<Paginated<Operation>>('/api/v1/operations'),
  get: (id: string) => apiRequest<Operation>(`/api/v1/operations/${id}`),
  create: (kind: string, profileId: string, params?: Record<string, unknown>) =>
    apiRequest<Operation>('/api/v1/operations', {
      method: 'POST',
      body: { kind, profile_id: profileId, params },
    }),
  cancel: (id: string) =>
    apiRequest<Operation>(`/api/v1/operations/${id}/cancel`, { method: 'POST' }),
  retry: (id: string) =>
    apiRequest<Operation>(`/api/v1/operations/${id}/retry`, { method: 'POST' }),
  logs: (id: string) =>
    apiRequest<Paginated<LogEntry>>(`/api/v1/operations/${id}/logs`),
}
