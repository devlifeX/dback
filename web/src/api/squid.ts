import { apiRequest } from './client'
import type { Paginated, SquidProxy, SquidSettings } from './types'

export const squidApi = {
  listProxies: () => apiRequest<Paginated<SquidProxy>>('/api/v1/squid/proxies'),
  getProxy: (id: string) => apiRequest<SquidProxy>(`/api/v1/squid/proxies/${id}`),
  saveProxy: (p: SquidProxy, etag?: string) =>
    apiRequest<SquidProxy>(p.id ? `/api/v1/squid/proxies/${p.id}` : '/api/v1/squid/proxies', {
      method: p.id ? 'PUT' : 'POST',
      body: p,
      etag,
    }),
  removeProxy: (id: string, etag?: string) =>
    apiRequest<void>(`/api/v1/squid/proxies/${id}`, { method: 'DELETE', etag }),
  getSettings: () => apiRequest<SquidSettings>('/api/v1/squid/settings'),
  saveSettings: (settings: SquidSettings, etag?: string) =>
    apiRequest<SquidSettings>('/api/v1/squid/settings', {
      method: 'PUT',
      body: settings,
      etag,
    }),
}
