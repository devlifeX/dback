import type { LucideIcon } from 'lucide-react'
import { Settings2, UserPlus, Users } from 'lucide-react'

export type UsersTab = {
  segment: string
  label: string
  description: string
  icon: LucideIcon
}

export const USERS_TABS: UsersTab[] = [
  { segment: 'list', label: 'All users', description: 'View and manage accounts', icon: Users },
  { segment: 'new', label: 'Add user', description: 'Create a new admin user', icon: UserPlus },
  { segment: 'settings', label: 'User settings', description: 'SMS OTP and two-factor auth', icon: Settings2 },
]

export function usersPath(segment: string) {
  return `/users/${segment}`
}

export function usersSegment(pathname: string) {
  const segment = pathname.split('/')[2]
  return USERS_TABS.some((t) => t.segment === segment) ? segment! : 'list'
}
