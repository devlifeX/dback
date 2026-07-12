import { apiRequest } from './client'
import type { AuditEntry, LogEntry, Paginated, ServerInfo, StorageInfo } from './types'

export const systemApi = {
  version: () => apiRequest<{ version: string; api: string }>('/api/v1/version'),
  revision: () => apiRequest<{ revision: number }>('/api/v1/system/revision'),
  serverInfo: () => apiRequest<ServerInfo>('/api/v1/system/server-info'),
  logs: (params?: { limit?: number; offset?: number }) => {
    const q = new URLSearchParams()
    if (params?.limit) q.set('limit', String(params.limit))
    if (params?.offset) q.set('offset', String(params.offset))
    const qs = q.toString()
    return apiRequest<Paginated<LogEntry>>(`/api/v1/logs${qs ? `?${qs}` : ''}`)
  },
  storage: () => apiRequest<StorageInfo>('/api/v1/system/storage'),
  audit: (params?: { limit?: number }) => {
    const q = new URLSearchParams()
    if (params?.limit) q.set('limit', String(params.limit))
    const qs = q.toString()
    return apiRequest<Paginated<AuditEntry>>(`/api/v1/system/audit${qs ? `?${qs}` : ''}`)
  },
}
