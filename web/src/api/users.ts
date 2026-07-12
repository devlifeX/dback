import { apiRequest } from './client'
import type { AuthSettings, Paginated, User } from './types'

export type CreateUserInput = {
  phone: string
  password: string
  name: string
}

export type UpdateUserInput = {
  phone?: string
  password?: string
  name?: string
  enabled?: boolean
}

export const usersApi = {
  list: () => apiRequest<Paginated<User>>('/api/v1/users'),

  get: (id: string) => apiRequest<User>(`/api/v1/users/${id}`),

  create: (input: CreateUserInput, etag?: string) =>
    apiRequest<User>('/api/v1/users', { method: 'POST', body: input, etag }),

  update: (id: string, input: UpdateUserInput, etag?: string) =>
    apiRequest<User>(`/api/v1/users/${id}`, { method: 'PUT', body: input, etag }),

  remove: (id: string, etag?: string) =>
    apiRequest<void>(`/api/v1/users/${id}`, { method: 'DELETE', etag }),

  getSettings: () => apiRequest<AuthSettings>('/api/v1/users/settings'),

  saveSettings: (settings: AuthSettings, etag?: string) =>
    apiRequest<AuthSettings>('/api/v1/users/settings', { method: 'PUT', body: settings, etag }),
}
