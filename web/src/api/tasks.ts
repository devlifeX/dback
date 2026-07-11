import { apiRequest } from './client'
import type { Paginated, Task } from './types'

export const tasksApi = {
  list: () => apiRequest<Paginated<Task>>('/api/v1/tasks'),
  get: (id: string) => apiRequest<Task>(`/api/v1/tasks/${id}`),
  save: (task: Task, etag?: string) =>
    apiRequest<Task>(task.id ? `/api/v1/tasks/${task.id}` : '/api/v1/tasks', {
      method: task.id ? 'PUT' : 'POST',
      body: task,
      etag,
    }),
  remove: (id: string, etag?: string) =>
    apiRequest<void>(`/api/v1/tasks/${id}`, { method: 'DELETE', etag }),
  run: (id: string) =>
    apiRequest<{ task_id: string; status: string }>(`/api/v1/tasks/${id}/run`, { method: 'POST' }),
}
