import { apiDownload, apiRequest } from './client'

export type StorageEntry = {
  name: string
  path: string
  size: number
  is_dir: boolean
  modified?: string
}

export type StorageRoot = {
  path: string
  label: string
}

export type StorageListing = {
  path: string
  parent?: string
  roots?: StorageRoot[]
  entries: StorageEntry[]
}

export const storageApi = {
  listLocal: (path?: string) => {
    const q = new URLSearchParams()
    if (path) q.set('path', path)
    const qs = q.toString()
    return apiRequest<StorageListing>(`/api/v1/storage/local${qs ? `?${qs}` : ''}`)
  },
  downloadLocal: (path: string) => {
    const q = new URLSearchParams({ path })
    return apiDownload(`/api/v1/storage/local/download?${q.toString()}`)
  },
  listRemote: (destinationId: string, prefix?: string) => {
    const q = new URLSearchParams({ destination_id: destinationId })
    if (prefix) q.set('prefix', prefix)
    return apiRequest<StorageListing>(`/api/v1/storage/remote?${q.toString()}`)
  },
  downloadRemote: (destinationId: string, key: string) => {
    const q = new URLSearchParams({ destination_id: destinationId, key })
    return apiDownload(`/api/v1/storage/remote/download?${q.toString()}`)
  },
}
