import type { ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { hostsApi } from '@/api/hosts'
import { operationsApi } from '@/api/operations'
import { tasksApi } from '@/api/tasks'
import { notificationsApi } from '@/api/notifications'
import { systemApi } from '@/api/system'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/badge'
import { PageHeader } from '@/components/shared/page'
import { Button } from '@/components/ui/button'
import { formatBytes } from '@/lib/utils'

function Widget({ title, loading, error, children }: { title: string; loading?: boolean; error?: boolean; children: ReactNode }) {
  return (
    <Card>
      <CardHeader><CardTitle>{title}</CardTitle></CardHeader>
      <CardContent>
        {loading ? <Skeleton className="h-20 w-full" /> : error ? <p className="text-sm text-[hsl(var(--destructive))]">Failed to load</p> : children}
      </CardContent>
    </Card>
  )
}

export function DashboardPage() {
  const version = useQuery({ queryKey: ['version'], queryFn: systemApi.version })
  const storage = useQuery({ queryKey: ['storage'], queryFn: systemApi.storage })
  const hosts = useQuery({ queryKey: ['hosts'], queryFn: hostsApi.list })
  const ops = useQuery({ queryKey: ['operations'], queryFn: operationsApi.list })
  const tasks = useQuery({ queryKey: ['tasks'], queryFn: tasksApi.list })
  const channels = useQuery({ queryKey: ['notifications'], queryFn: notificationsApi.list })

  const operations = ops.data?.items ?? []
  const running = operations.filter((o) => o.status === 'running')
  const failed = operations.filter((o) => o.status === 'failed').slice(0, 5)
  const recent = operations.slice(0, 8)
  const enabledTasks = (tasks.data?.items ?? []).filter((t) => t.enabled).length

  const healthOk = !version.isError && version.data

  return (
    <div>
      <PageHeader
        title="Dashboard"
        description={version.data ? `API ${version.data.api} · ${version.data.version}` : 'Overview'}
        actions={
          <>
            <Button asChild variant="outline" size="sm"><Link to="/operations">Operations</Link></Button>
            <Button asChild size="sm"><Link to="/hosts">Manage hosts</Link></Button>
          </>
        }
      />
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        <Widget title="Health" loading={version.isLoading} error={version.isError}>
          <p className="text-sm">{healthOk ? 'Server reachable and ready' : 'Checking…'}</p>
          <Badge status={healthOk ? 'succeeded' : 'queued'} />
        </Widget>
        <Widget title="Storage (local)" loading={storage.isLoading} error={storage.isError}>
          <p className="text-3xl font-semibold">{formatBytes(storage.data?.local?.bytes)}</p>
          <p className="text-sm text-[hsl(var(--muted-foreground))]">
            {storage.data?.local?.files ?? 0} files · {storage.data?.local?.roots ?? 0} folders
          </p>
          <Button asChild variant="ghost" className="mt-1 h-auto p-0 text-sm text-[hsl(var(--primary))]">
            <Link to="/storage/local">Browse local</Link>
          </Button>
        </Widget>
        <Widget title="Storage (remote)" loading={storage.isLoading} error={storage.isError}>
          <p className="text-3xl font-semibold">{formatBytes(storage.data?.remote?.bytes)}</p>
          <p className="text-sm text-[hsl(var(--muted-foreground))]">
            {storage.data?.remote?.objects ?? 0} objects · {storage.data?.remote?.destinations ?? 0} destinations
          </p>
          <Button asChild variant="ghost" className="mt-1 h-auto p-0 text-sm text-[hsl(var(--primary))]">
            <Link to="/storage/remote">Browse remote</Link>
          </Button>
        </Widget>
        <Widget title="Hosts" loading={hosts.isLoading} error={hosts.isError}>
          <p className="text-3xl font-semibold">{hosts.data?.meta.total ?? 0}</p>
        </Widget>
        <Widget title="Running operations" loading={ops.isLoading} error={ops.isError}>
          <p className="text-3xl font-semibold">{running.length}</p>
        </Widget>
        <Widget title="Scheduled tasks" loading={tasks.isLoading} error={tasks.isError}>
          <p className="text-3xl font-semibold">{enabledTasks}</p>
        </Widget>
        <Widget title="Notification channels" loading={channels.isLoading} error={channels.isError}>
          <p className="text-3xl font-semibold">{channels.data?.meta.total ?? 0}</p>
        </Widget>
        <Widget title="Failed operations" loading={ops.isLoading} error={ops.isError}>
          <ul className="space-y-2 text-sm">
            {failed.length === 0 ? <li className="text-[hsl(var(--muted-foreground))]">None</li> : null}
            {failed.map((o) => (
              <li key={o.id}>
                <Link className="hover:underline" to={`/operations/${o.id}`}>{o.kind} · {o.profile_id}</Link>
              </li>
            ))}
          </ul>
        </Widget>
      </div>
      <Card className="mt-4">
        <CardHeader><CardTitle>Recent operations</CardTitle></CardHeader>
        <CardContent className="space-y-2">
          {recent.length === 0 ? <p className="text-sm text-[hsl(var(--muted-foreground))]">No operations yet</p> : null}
          {recent.map((o) => (
            <div key={o.id} className="flex items-center justify-between gap-2 text-sm">
              <Link to={`/operations/${o.id}`} className="hover:underline">{o.kind} · {o.profile_id}</Link>
              <Badge status={o.status} />
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  )
}
