import type { LucideIcon } from 'lucide-react'
import {
  Cloud,
  FileKey,
  Globe,
  HardDrive,
  ScrollText,
  Server,
  Settings2,
} from 'lucide-react'

export type SettingsTab = {
  segment: string
  label: string
  description: string
  icon: LucideIcon
}

export const SETTINGS_TABS: SettingsTab[] = [
  { segment: 'general', label: 'General', description: 'Version and vault revision', icon: Settings2 },
  { segment: 'destinations', label: 'Destinations', description: 'Remote storage targets', icon: HardDrive },
  { segment: 'squid', label: 'Squid', description: 'Regional proxies and primary host country', icon: Globe },
  { segment: 'sync', label: 'Sync', description: 'Cloud sync settings and push/pull', icon: Cloud },
  { segment: 'vault', label: 'Vault', description: 'Export and import app data', icon: FileKey },
  { segment: 'audit', label: 'Audit', description: 'API request audit trail', icon: ScrollText },
  { segment: 'logs', label: 'Logs', description: 'Backup and operation activity', icon: Server },
]

export function settingsPath(segment: string) {
  return `/settings/${segment}`
}

export function settingsSegment(pathname: string) {
  const segment = pathname.split('/')[2]
  return SETTINGS_TABS.some((t) => t.segment === segment) ? segment! : 'general'
}
