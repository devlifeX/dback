import { cn } from '@/lib/utils'

const map: Record<string, string> = {
  succeeded: 'bg-[hsl(var(--success)/0.15)] text-[hsl(var(--success))]',
  failed: 'bg-[hsl(var(--destructive)/0.15)] text-[hsl(var(--destructive))]',
  running: 'bg-[hsl(var(--warning)/0.15)] text-[hsl(var(--warning))]',
  queued: 'bg-[hsl(var(--muted))] text-[hsl(var(--muted-foreground))]',
  canceled: 'bg-[hsl(var(--muted))] text-[hsl(var(--muted-foreground))]',
  skipped: 'bg-[hsl(var(--muted))] text-[hsl(var(--muted-foreground))]',
}

export function Badge({ status, className }: { status: string; className?: string }) {
  const key = status.toLowerCase()
  return (
    <span className={cn('inline-flex rounded px-2 py-0.5 text-xs font-medium capitalize', map[key] ?? map.queued, className)}>
      {status}
    </span>
  )
}

export function Skeleton({ className }: { className?: string }) {
  return <div className={cn('animate-pulse rounded-md bg-[hsl(var(--muted))]', className)} />
}
