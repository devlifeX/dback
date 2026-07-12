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
import { countryLabel } from '@/lib/country-flag'
import { useFormatDate } from '@/lib/datetime'
import type { URLCheckHourlyBucket } from '@/api/types'

const LINE_COLORS = ['#2563eb', '#16a34a', '#dc2626', '#9333ea', '#ea580c', '#0891b2', '#4f46e5', '#be123c']

function chartKey(b: URLCheckHourlyBucket) {
  return countryLabel(b.source_label || 'Direct', b.country_code)
}

export function UrlChecksChart({ hostId, url }: { hostId: string; url?: string }) {
  const formatDate = useFormatDate()
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
    const key = chartKey(b)
    const row = byHour.get(b.hour) ?? { hour: formatDate(b.hour, 'MMM D HH:mm') }
    row[key] = Math.round(b.avg_ttfb_ms)
    byHour.set(b.hour, row)
  }
  const data = [...byHour.values()].sort((a, b) => String(a.hour).localeCompare(String(b.hour)))
  const sourceKeys = [...new Set(items.map(chartKey))]

  return (
    <div className="h-64 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={data} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
          <CartesianGrid strokeDasharray="3 3" className="stroke-[hsl(var(--border))]" />
          <XAxis dataKey="hour" tick={{ fontSize: 11 }} />
          <YAxis tick={{ fontSize: 11 }} unit="ms" />
          <Tooltip formatter={(value: number) => [`${value} ms`, 'TTFB']} />
          <Legend />
          {sourceKeys.map((key, i) => (
            <Line
              key={key}
              type="monotone"
              dataKey={key}
              name={key}
              stroke={LINE_COLORS[i % LINE_COLORS.length]}
              dot={false}
              strokeWidth={2}
            />
          ))}
        </LineChart>
      </ResponsiveContainer>
    </div>
  )
}
