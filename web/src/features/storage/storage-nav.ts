export const STORAGE_TABS = [
  { segment: 'local', label: 'Local' },
  { segment: 'remote', label: 'Remote' },
] as const

export function storagePath(segment: string) {
  return `/storage/${segment}`
}

export function storageSegment(pathname: string) {
  const segment = pathname.split('/')[2]
  return STORAGE_TABS.some((t) => t.segment === segment) ? segment! : 'local'
}
