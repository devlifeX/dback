import { useQuery } from '@tanstack/react-query'
import {
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { hostsApi } from '@/api/hosts'

function formatHour(iso: string) {
  try {
    return new Date(iso).toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit' })
  } catch {
    return iso
  }
}

export function UrlChecksChart({ hostId, url }: { hostId: string; url?: string }) {
  const checks = useQuery({
    queryKey: ['url-checks', hostId, url],
    queryFn: () => hostsApi.urlChecks(hostId, { url }),
    enabled: !!hostId,
  })

  const items = checks.data?.items ?? []
  if (checks.isLoading) {
    return <p className="text-sm text-[hsl(var(--muted-foreground))]">Loading URL check history…</p>
  }
  if (items.length === 0) {
    return <p className="text-sm text-[hsl(var(--muted-foreground))]">No URL check samples yet. Run a url_checker operation to populate this chart.</p>
  }

  const byHour = new Map<string, Record<string, number | string>>()
  for (const b of items) {
    const row = byHour.get(b.hour) ?? { hour: formatHour(b.hour) }
    row[`${b.url} TTFB`] = Math.round(b.avg_ttfb_ms)
    row[`${b.url} status`] = b.last_status_code
    byHour.set(b.hour, row)
  }
  const data = [...byHour.values()]
  const ttfbKeys = [...new Set(items.map((b) => `${b.url} TTFB`))]

  return (
    <div className="h-64 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={data} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
          <CartesianGrid strokeDasharray="3 3" className="stroke-[hsl(var(--border))]" />
          <XAxis dataKey="hour" tick={{ fontSize: 11 }} />
          <YAxis tick={{ fontSize: 11 }} unit="ms" />
          <Tooltip />
          <Legend />
          {ttfbKeys.map((key, i) => (
            <Line key={key} type="monotone" dataKey={key} stroke={i === 0 ? '#2563eb' : '#16a34a'} dot={false} strokeWidth={2} />
          ))}
        </LineChart>
      </ResponsiveContainer>
    </div>
  )
}
