export type ApiError = {
  code: string
  message: string
}

export type Paginated<T> = {
  items: T[]
  meta: { total: number; limit: number; offset: number }
  etag?: string
}

export type Operation = {
  id: string
  kind: string
  profile_id: string
  trigger_ref?: string
  status: string
  started_at?: string
  finished_at?: string
  error?: string
  progress?: string
}

export type Host = {
  id: string
  name: string
  group?: string
  host: string
  connection_type: string
  db_type?: string
  target_db_name?: string
}

export type Task = {
  id: string
  name: string
  enabled: boolean
  profile_ids: string[]
  trigger: {
    type: string
    cron?: { expr: string; timezone: string }
    interval?: { every: number }
  }
  actions: { operation: string }[]
  state?: {
    next_run_at?: string
    last_run_status?: string
  }
}

export type NotifyChannel = {
  id: string
  name: string
  provider: string
  enabled: boolean
  events?: string[]
}

export type SQLTemplate = {
  id: string
  name: string
  body: string
}

export type RemoteDestination = {
  id: string
  name: string
  type: string
}

export type LogEntry = {
  id: string
  timestamp: string
  action: string
  details: string
  level?: string
  status?: string
}
