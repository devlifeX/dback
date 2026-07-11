import { apiDownload, apiRequest } from './client'
import type { ExportRecord, LastVerified, Paginated } from './types'

export const backupsApi = {
  list: (params?: { limit?: number; offset?: number }) => {
    const q = new URLSearchParams()
    if (params?.limit) q.set('limit', String(params.limit))
    if (params?.offset) q.set('offset', String(params.offset))
    const qs = q.toString()
    return apiRequest<Paginated<ExportRecord>>(`/api/v1/backups${qs ? `?${qs}` : ''}`)
  },
  get: (id: string) => apiRequest<ExportRecord>(`/api/v1/backups/${id}`),
  quickVerify: (id: string) =>
    apiRequest<LastVerified>(`/api/v1/backups/${id}/verify/quick`, { method: 'POST' }),
  download: (id: string) => apiDownload(`/api/v1/backups/${id}/download`),
}
