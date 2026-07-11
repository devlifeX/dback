import { apiRequest } from './client'
import type { LogEntry, Paginated } from './types'

export const systemApi = {
  version: () => apiRequest<{ version: string; api: string }>('/api/v1/version'),
  revision: () => apiRequest<{ revision: number }>('/api/v1/system/revision'),
  logs: () => apiRequest<Paginated<LogEntry>>('/api/v1/logs'),
}
