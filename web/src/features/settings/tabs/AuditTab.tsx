import { useQuery } from '@tanstack/react-query'
import { systemApi } from '@/api/system'
import { Skeleton } from '@/components/ui/badge'
import { formatDate } from '@/lib/utils'

export function AuditTab() {
  const audit = useQuery({ queryKey: ['audit'], queryFn: () => systemApi.audit({ limit: 50 }) })
  const auditItems = audit.data?.items ?? []

  if (audit.isLoading) return <Skeleton className="h-24" />

  return (
    <div className="max-h-[32rem] overflow-auto rounded border border-[hsl(var(--border))] text-xs font-mono">
      {auditItems.length === 0 ? <p className="p-4 text-[hsl(var(--muted-foreground))]">No audit entries</p> : null}
      {auditItems.map((entry, i) => (
        <div key={i} className="border-b border-[hsl(var(--border))] px-3 py-1">
          {formatDate(entry.timestamp)} {entry.method} {entry.path} → {entry.status}
        </div>
      ))}
    </div>
  )
}
