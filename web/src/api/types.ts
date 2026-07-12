export type ApiError = {
  code: string
  message: string
}

export type Paginated<T> = {
  items: T[]
  meta: { total: number; limit: number; offset: number }
  etag?: string
}

export type ConnectionType = 'SSH' | 'JumpHost' | 'Localhost' | 'WordPress'
export type AuthType = 'Password' | 'Key File'
export type DBType = 'MySQL' | 'MariaDB'
export type ArchiveCompression = 'zstd' | 'gzip'
export type ExportType = 'database' | 'files'

export type FileBackupPath = {
  id: string
  name: string
  remote_path: string
  canonical_key?: string
}

export type Profile = {
  id: string
  name: string
  group?: string
  host: string
  port: string
  connection_type: ConnectionType

  ssh_user: string
  ssh_password?: string
  auth_type: AuthType
  auth_key_path: string
  auth_key_pem?: string

  jump_host?: string
  jump_port?: string
  jump_user?: string
  jump_password?: string
  jump_auth_type?: AuthType
  jump_auth_key_path?: string
  jump_auth_key_pem?: string

  wp_url?: string
  wp_key?: string

  db_host: string
  db_port: string
  db_user: string
  db_password?: string
  db_type: DBType
  is_docker: boolean
  container_id: string

  target_db_name: string
  destination: string
  pre_import_query?: string
  run_query_before_import?: boolean
  post_import_query?: string
  run_query_after_import?: boolean

  import_protected?: boolean

  file_backup_enabled?: boolean
  file_backup_destination?: string
  file_backup_compression?: ArchiveCompression
  file_backup_exclude?: string[]
  file_backup_paths?: FileBackupPath[]

  remote_upload_destination_ids?: string[]
  remote_auto_upload_db?: boolean
  remote_auto_upload_files?: boolean

  url_check?: URLCheck
  backup_policy?: BackupPolicy

  created_at?: string
  updated_at?: string
}

export type URLTarget = { url: string }

export type URLCheck = {
  primary?: URLTarget
  secondary?: URLTarget[]
  proxy_ids?: string[]
}

export type BackupPolicy = {
  enabled?: boolean
  db_local_keep?: number
  db_remote_keep?: number
  files_local_keep?: number
  files_remote_keep?: number
}

export type URLCheckHourlyBucket = {
  hour: string
  url: string
  source_label: string
  proxy_id?: string
  country_code?: string
  avg_ttfb_ms: number
  min_ttfb_ms: number
  max_ttfb_ms: number
  samples: number
  ok_count: number
  fail_count: number
  last_status_code: number
}

export type SquidProxy = {
  id: string
  name: string
  url: string
  country: string
  country_code?: string
  enabled: boolean
}

export type SquidSettings = {
  primary_host_country: string
  primary_host_country_code?: string
}

/** Host is the API alias for Profile (secrets redacted on read). */
export type Host = Profile

export type LastVerified = {
  verified_at: string
  method: string
  passed: boolean
  report?: { table: string; expected: number; actual: number; match: boolean }[]
}

export type RemoteUploadState = {
  destination_id: string
  status: string
  remote_key?: string
  uploaded_at?: string
  error?: string
}

export type ExportRecord = {
  id: string
  operation_id?: string
  profile_id: string
  profile_name: string
  database_name: string
  export_type?: ExportType
  file_backup_path_id?: string
  source_label?: string
  source_path?: string
  export_date: string
  file_path: string
  file_size: string
  file_size_bytes: number
  connection_type?: ConnectionType
  sha256?: string
  quick_verified?: LastVerified
  deep_verified?: LastVerified
  remote_uploads?: RemoteUploadState[]
}

export type OperationKind = 'backup_db' | 'backup_files' | 'upload' | 'restore' | 'deep_verify' | 'url_checker'

export type Operation = {
  id: string
  kind: string
  profile_id: string
  profile_name?: string
  trigger_ref?: string
  status: string
  started_at?: string
  finished_at?: string
  error?: string
  progress?: string
  artifacts?: { type: string; id?: string; path?: string }[]
}

export type TriggerType = 'cron' | 'interval' | 'one_shot' | 'on_boot'

export type TriggerSpec = {
  type: TriggerType
  cron?: { expr: string; timezone: string }
  interval?: { every: number }
  one_shot?: { at: string }
  on_boot?: { delay: number }
}

export type ActionSpec = {
  operation: string
  params?: Record<string, unknown>
}

export type Task = {
  id: string
  name: string
  enabled: boolean
  profile_ids: string[]
  trigger: TriggerSpec
  actions: ActionSpec[]
  overlap_policy?: string
  max_concurrent_profiles?: number
  notify_channel_ids?: string[]
  state?: {
    next_run_at?: string
    last_run_status?: string
    last_fired_at?: string
  }
}

export type TaskRunRecord = {
  id: string
  task_id: string
  profile_id: string
  trigger_type: TriggerType
  started_at: string
  finished_at: string
  status: string
  action_results?: { operation_id: string; kind: string; status: string; error?: string }[]
}

export type NotifyProvider = 'telegram' | 'slack' | 'bale' | 'webhook' | 'kavenegar' | 'melipayamak'

export type TelegramConfig = { token: string; chat_id: string; thread_id?: number; parse_mode?: string }
export type SlackConfig = { webhook_url: string; channel?: string; username?: string; icon_emoji?: string }
export type BaleConfig = { token: string; chat_id: string }
export type WebhookConfig = {
  url: string
  method?: string
  headers?: Record<string, string>
  timeout_sec?: number
  retry_count?: number
}
export type KavenegarConfig = { api_key: string; line: string; receptor: string }
export type MeliPayamakConfig = { username: string; password: string; from: string; to: string }

export type NotifyChannel = {
  id: string
  name: string
  provider: NotifyProvider
  enabled: boolean
  events?: string[]
  config?: TelegramConfig | SlackConfig | BaleConfig | WebhookConfig | KavenegarConfig | MeliPayamakConfig
}

export type SMSProvider = 'kavenegar' | 'melipayamak'

export type AuthSMSConfig = {
  api_key?: string
  line?: string
  username?: string
  password?: string
  from?: string
}

export type AuthSettings = {
  two_factor_enabled: boolean
  sms_provider?: SMSProvider
  sms_config?: AuthSMSConfig
  otp_ttl_seconds?: number
  otp_length?: number
}

export type User = {
  id: string
  phone: string
  name: string
  enabled: boolean
  created_at?: string
  updated_at?: string
}

export type LoginResponse = {
  token?: string
  otp_required?: boolean
  challenge_id?: string
  expires_in_sec?: number
  user?: User
}

export type SQLTemplate = {
  id: string
  name: string
  body: string
  description?: string
  created_at?: string
  updated_at?: string
}

export type S3DestinationConfig = {
  endpoint: string
  region?: string
  bucket: string
  access_key_id: string
  secret_key?: string
  use_ssl: boolean
}

export type RemoteDestination = {
  id: string
  name: string
  type: string
  s3?: S3DestinationConfig
}

export type SyncSettings = {
  endpoint: string
  region?: string
  bucket: string
  access_key_id: string
  secret_key?: string
  use_ssl: boolean
}

export type LogEntry = {
  id: string
  timestamp: string
  action: string
  details: string
  level?: string
  status?: string
  phase?: string
  error?: string
  profile_id?: string
  profile_name?: string
  operation_id?: string
}

export type StorageInfo = {
  local: {
    bytes: number
    files: number
    roots: number
  }
  remote: {
    bytes: number
    objects: number
    destinations: number
  }
  backup_records: number
  backup_bytes: number
  hosts: number
}

export type ServerInfo = {
  cpu_count: number
  ram_total_bytes: number
  ram_free_bytes: number
  disk_free_bytes: number
  data_dir: string
  internet_ok: boolean
  internet_error?: string
  internet_host: string
  checked_at: string
}

export type AuditEntry = {
  timestamp: string
  method: string
  path: string
  status: number
  client_ip?: string
}

export const OPERATION_KINDS: { value: OperationKind; label: string }[] = [
  { value: 'backup_db', label: 'Database backup' },
  { value: 'backup_files', label: 'File backup' },
  { value: 'upload', label: 'Remote upload' },
  { value: 'restore', label: 'Restore' },
  { value: 'deep_verify', label: 'Deep verify' },
  { value: 'url_checker', label: 'URL checker' },
]

export const NOTIFY_EVENTS = [
  'operation.started',
  'operation.completed',
  'operation.failed',
  'operation.canceled',
  'task.skipped',
] as const

/** Recommended defaults — includes success (completed) which is easy to miss. */
export const DEFAULT_NOTIFY_EVENTS: string[] = [...NOTIFY_EVENTS]

export function emptyProfile(): Profile {
  return {
    id: '',
    name: '',
    host: '',
    port: '22',
    connection_type: 'SSH',
    ssh_user: 'root',
    auth_type: 'Password',
    auth_key_path: '',
    db_host: '127.0.0.1',
    db_port: '3306',
    db_user: 'root',
    db_type: 'MySQL',
    is_docker: false,
    container_id: '',
    target_db_name: '',
    destination: '',
    file_backup_paths: [],
    file_backup_exclude: [],
    remote_upload_destination_ids: [],
    url_check: { primary: { url: '' }, secondary: [], proxy_ids: [] },
    backup_policy: { enabled: false, db_local_keep: 0, db_remote_keep: 0, files_local_keep: 0, files_remote_keep: 0 },
  }
}
