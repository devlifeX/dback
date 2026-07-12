import { useQuery } from '@tanstack/react-query'
import { systemApi } from '@/api/system'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/badge'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { formatBytes } from '@/lib/utils'
import { useFormatDate } from '@/lib/datetime'
import { useUIStore } from '@/store/ui-store'

const REFRESH_MS = 30_000

export function GeneralTab() {
  const formatDate = useFormatDate()
  const calendar = useUIStore((s) => s.calendar)
  const setCalendar = useUIStore((s) => s.setCalendar)
  const version = useQuery({ queryKey: ['version'], queryFn: systemApi.version })
  const revision = useQuery({ queryKey: ['revision'], queryFn: systemApi.revision })
  const server = useQuery({
    queryKey: ['server-info'],
    queryFn: systemApi.serverInfo,
    refetchInterval: REFRESH_MS,
  })

  const info = server.data

  return (
    <div className="grid gap-4 lg:grid-cols-2">
      <Card>
        <CardHeader><CardTitle>System</CardTitle></CardHeader>
        <CardContent className="space-y-2 text-sm">
          {version.isLoading ? <Skeleton className="h-4 w-32" /> : <p>Version: {version.data?.version}</p>}
          {revision.isLoading ? <Skeleton className="h-4 w-32" /> : <p>Vault revision: {revision.data?.revision}</p>}
        </CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle>Server</CardTitle></CardHeader>
        <CardContent className="space-y-2 text-sm">
          {server.isLoading ? (
            <>
              <Skeleton className="h-4 w-40" />
              <Skeleton className="h-4 w-48" />
              <Skeleton className="h-4 w-44" />
            </>
          ) : server.isError ? (
            <p className="text-[hsl(var(--destructive))]">Could not load server info</p>
          ) : info ? (
            <>
              <p>CPUs: {info.cpu_count}</p>
              <p>
                RAM: {formatBytes(info.ram_free_bytes)} free / {formatBytes(info.ram_total_bytes)} total
              </p>
              <p>Disk free (data dir): {formatBytes(info.disk_free_bytes)}</p>
              <p className="text-xs text-[hsl(var(--muted-foreground))] break-all">{info.data_dir}</p>
              <p className="flex items-center gap-2">
                Internet ({info.internet_host}):
                <Badge status={info.internet_ok ? 'succeeded' : 'failed'} />
                {!info.internet_ok && info.internet_error ? (
                  <span className="text-xs text-[hsl(var(--muted-foreground))]">{info.internet_error}</span>
                ) : null}
              </p>
              <p className="text-xs text-[hsl(var(--muted-foreground))]">
                Updated {formatDate(info.checked_at)}
              </p>
            </>
          ) : null}
        </CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle>Preferences</CardTitle></CardHeader>
        <CardContent className="space-y-2 text-sm">
          <div className="flex items-center gap-3">
            <span className="text-[hsl(var(--muted-foreground))]">Calendar</span>
            <Select value={calendar} onValueChange={(v) => setCalendar(v as 'jalali' | 'gregorian')}>
              <SelectTrigger className="w-40"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="jalali">Shamsi (Jalali)</SelectItem>
                <SelectItem value="gregorian">Gregorian</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
