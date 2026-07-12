import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { hostsApi } from '@/api/hosts'
import { operationsApi } from '@/api/operations'
import type { URLCheckHourlyBucket } from '@/api/types'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { countryLabel } from '@/lib/country-flag'
import { UrlChecksChart } from './UrlChecksChart'

function sourceLabel(b: URLCheckHourlyBucket) {
  return countryLabel(b.source_label || 'Direct', b.country_code)
}

function latestBySource(buckets: URLCheckHourlyBucket[]) {
  const bySource = new Map<string, URLCheckHourlyBucket>()
  for (const b of buckets) {
    const key = sourceLabel(b)
    const existing = bySource.get(key)
    if (!existing || b.hour > existing.hour) {
      bySource.set(key, b)
    }
  }
  return [...bySource.values()].sort((a, b) => sourceLabel(a).localeCompare(sourceLabel(b)))
}

export function CheckHostStatusDialog({
  hostId,
  hostName,
  primaryUrl,
  open,
  onOpenChange,
}: {
  hostId: string
  hostName: string
  primaryUrl?: string
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const qc = useQueryClient()
  const checks = useQuery({
    queryKey: ['url-checks', hostId, primaryUrl],
    queryFn: () => hostsApi.urlChecks(hostId, { url: primaryUrl }),
    enabled: open && !!hostId,
  })

  const runCheck = useMutation({
    mutationFn: () => operationsApi.create('url_checker', hostId, { url_index: -1 }),
    onSuccess: () => {
      toast.success('URL check queued')
      void qc.invalidateQueries({ queryKey: ['url-checks', hostId] })
      void qc.invalidateQueries({ queryKey: ['operations'] })
    },
    onError: (e: Error) => toast.error(e.message),
  })

  const items = checks.data?.items ?? []
  const latest = latestBySource(items)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] max-w-2xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Check host status — {hostName}</DialogTitle>
        </DialogHeader>

        <div className="flex flex-wrap items-center gap-2">
          <Button size="sm" disabled={runCheck.isPending} onClick={() => runCheck.mutate()}>
            Run check now
          </Button>
          {primaryUrl ? (
            <span className="text-xs text-[hsl(var(--muted-foreground))] truncate">{primaryUrl}</span>
          ) : (
            <span className="text-xs text-[hsl(var(--muted-foreground))]">No primary URL configured</span>
          )}
        </div>

        {checks.isLoading ? (
          <p className="text-sm text-[hsl(var(--muted-foreground))]">Loading status…</p>
        ) : latest.length === 0 ? (
          <p className="text-sm text-[hsl(var(--muted-foreground))]">No URL check samples yet. Run a check to see status.</p>
        ) : (
          <div className="space-y-2">
            <p className="text-sm font-medium">Latest status by source</p>
            <div className="divide-y divide-[hsl(var(--border))] rounded border border-[hsl(var(--border))] text-sm">
              {latest.map((b) => (
                <div key={`${b.hour}-${b.source_label}`} className="flex flex-wrap items-center justify-between gap-2 px-3 py-2">
                  <span>{sourceLabel(b)}</span>
                  <div className="flex items-center gap-3 text-[hsl(var(--muted-foreground))]">
                    <span>HTTP {b.last_status_code || '—'}</span>
                    <span>{Math.round(b.avg_ttfb_ms)} ms TTFB</span>
                    <Badge status={b.fail_count > 0 && b.ok_count === 0 ? 'failed' : 'succeeded'} />
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        <div className="space-y-2 border-t border-[hsl(var(--border))] pt-4">
          <p className="text-sm font-medium">TTFB history (last 7 days)</p>
          <UrlChecksChart hostId={hostId} url={primaryUrl} />
        </div>
      </DialogContent>
    </Dialog>
  )
}
