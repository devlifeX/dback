import type { ReactNode } from 'react'
import { ChevronRight, Download, File, Folder } from 'lucide-react'
import { Button } from '@/components/ui/button'
import type { StorageEntry } from '@/api/storage'
import { formatBytes, formatDate } from '@/lib/utils'
import { cn } from '@/lib/utils'

type StorageExplorerProps = {
  path: string
  parent?: string
  entries: StorageEntry[]
  loading?: boolean
  emptyMessage?: string
  onOpen: (entry: StorageEntry) => void
  onNavigate: (path: string) => void
  onDownload?: (entry: StorageEntry) => void
  downloadingPath?: string
  breadcrumbLabel?: (segment: string, index: number, parts: string[]) => string
  profileNames?: Record<string, string>
}

function hostLabelForEntry(name: string, profileNames?: Record<string, string>) {
  if (!profileNames || !/^\d+$/.test(name)) return undefined
  return profileNames[name]
}

function defaultBreadcrumb(parts: string[]) {
  if (parts.length === 0) return 'Roots'
  return parts[parts.length - 1] ?? 'Browse'
}

export function StorageExplorer({
  path,
  parent,
  entries,
  loading,
  emptyMessage = 'This folder is empty',
  onOpen,
  onNavigate,
  onDownload,
  downloadingPath,
  breadcrumbLabel,
  profileNames,
}: StorageExplorerProps) {
  const parts = path ? path.replace(/\\/g, '/').split('/').filter(Boolean) : []

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-1 text-sm text-[hsl(var(--muted-foreground))]">
        <Button variant="ghost" size="sm" className="h-7 px-2" onClick={() => onNavigate('')}>
          Home
        </Button>
        {parts.map((part, index) => {
          const label = breadcrumbLabel?.(part, index, parts) ?? part
          return (
            <span key={`${part}-${index}`} className="flex items-center gap-1">
              <ChevronRight className="h-3.5 w-3.5" />
              <span>{label}</span>
            </span>
          )
        })}
        {parts.length === 0 ? (
          <span className="flex items-center gap-1">
            <ChevronRight className="h-3.5 w-3.5" />
            <span>{defaultBreadcrumb(parts)}</span>
          </span>
        ) : null}
        {parent !== undefined && path ? (
          <Button variant="outline" size="sm" className="ml-auto h-7" onClick={() => onNavigate(parent)}>
            Up
          </Button>
        ) : null}
      </div>

      <div className="overflow-hidden rounded-lg border border-[hsl(var(--border))]">
        <table className="w-full text-sm">
          <thead className="bg-[hsl(var(--muted)/0.5)] text-left">
            <tr>
              <th className="px-3 py-2 font-medium">Name</th>
              <th className="px-3 py-2 font-medium">Size</th>
              <th className="px-3 py-2 font-medium">Modified</th>
              <th className="px-3 py-2 font-medium w-24" />
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td colSpan={4} className="px-3 py-8 text-center text-[hsl(var(--muted-foreground))]">
                  Loading…
                </td>
              </tr>
            ) : entries.length === 0 ? (
              <tr>
                <td colSpan={4} className="px-3 py-8 text-center text-[hsl(var(--muted-foreground))]">
                  {emptyMessage}
                </td>
              </tr>
            ) : (
              entries.map((entry) => (
                <StorageRow
                  key={entry.path}
                  entry={entry}
                  onOpen={onOpen}
                  onDownload={onDownload}
                  downloading={downloadingPath === entry.path}
                  hostLabel={entry.is_dir ? hostLabelForEntry(entry.name, profileNames) : undefined}
                />
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function StorageRow({
  entry,
  onOpen,
  onDownload,
  downloading,
  hostLabel,
}: {
  entry: StorageEntry
  onOpen: (entry: StorageEntry) => void
  onDownload?: (entry: StorageEntry) => void
  downloading?: boolean
  hostLabel?: string
}) {
  const Icon = entry.is_dir ? Folder : File
  return (
    <tr className="border-t border-[hsl(var(--border))] hover:bg-[hsl(var(--muted)/0.35)]">
      <td className="px-3 py-2">
        <button
          type="button"
          className={cn('flex flex-col items-start gap-0.5 text-left', entry.is_dir && 'font-medium hover:underline')}
          onClick={() => onOpen(entry)}
        >
          <span className="flex items-center gap-2">
            <Icon className="h-4 w-4 shrink-0 text-[hsl(var(--muted-foreground))]" />
            {entry.name}
          </span>
          {hostLabel ? (
            <span className="ml-6 inline-flex rounded bg-[hsl(var(--muted))] px-2 py-0.5 text-xs text-[hsl(var(--muted-foreground))]">
              {hostLabel}
            </span>
          ) : null}
        </button>
      </td>
      <td className="px-3 py-2 text-[hsl(var(--muted-foreground))]">
        {entry.is_dir ? '—' : formatBytes(entry.size)}
      </td>
      <td className="px-3 py-2 text-[hsl(var(--muted-foreground))]">
        {entry.modified ? formatDate(entry.modified) : '—'}
      </td>
      <td className="px-3 py-2 text-right">
        {!entry.is_dir && onDownload ? (
          <Button variant="ghost" size="sm" className="h-7" disabled={downloading} onClick={() => onDownload(entry)}>
            <Download className="h-3.5 w-3.5" />
          </Button>
        ) : null}
      </td>
    </tr>
  )
}

export function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

export function StorageHint({ children }: { children: ReactNode }) {
  return <p className="text-sm text-[hsl(var(--muted-foreground))]">{children}</p>
}
