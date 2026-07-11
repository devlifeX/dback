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

function Widget({ title, loading, error, children }: { title: string; loading?: boolean; error?: boolean; children: ReactNode }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent>
        {loading ? <Skeleton className="h-20 w-full" /> : error ? <p className="text-sm text-[hsl(var(--destructive))]">Failed to load</p> : children}
      </CardContent>
    </Card>
  )
}

export function DashboardPage() {
  const version = useQuery({ queryKey: ['version'], queryFn: systemApi.version })
  const hosts = useQuery({ queryKey: ['hosts'], queryFn: hostsApi.list })
  const ops = useQuery({ queryKey: ['operations'], queryFn: operationsApi.list })
  const tasks = useQuery({ queryKey: ['tasks'], queryFn: tasksApi.list })
  const channels = useQuery({ queryKey: ['notifications'], queryFn: notificationsApi.list })

  const operations = ops.data?.items ?? []
  const running = operations.filter((o) => o.status === 'running')
  const failed = operations.filter((o) => o.status === 'failed').slice(0, 5)
  const recent = operations.slice(0, 8)
  const enabledTasks = (tasks.data?.items ?? []).filter((t) => t.enabled).length

  return (
    <div>
      <PageHeader
        title="Dashboard"
        description={version.data ? `API ${version.data.api} · ${version.data.version}` : 'Overview'}
        actions={
          <>
            <Button asChild variant="outline" size="sm">
              <Link to="/operations">Operations</Link>
            </Button>
            <Button asChild size="sm">
              <Link to="/hosts">Manage hosts</Link>
            </Button>
          </>
        }
      />
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        <Widget title="Health" loading={version.isLoading} error={version.isError}>
          <p className="text-sm">Server reachable</p>
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
                <Link className="hover:underline" to={`/operations/${o.id}`}>
                  {o.kind} · {o.profile_id}
                </Link>
              </li>
            ))}
          </ul>
        </Widget>
      </div>
      <Card className="mt-4">
        <CardHeader>
          <CardTitle>Recent operations</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          {recent.length === 0 ? <p className="text-sm text-[hsl(var(--muted-foreground))]">No operations yet</p> : null}
          {recent.map((o) => (
            <div key={o.id} className="flex items-center justify-between gap-2 text-sm">
              <Link to={`/operations/${o.id}`} className="hover:underline">
                {o.kind} · {o.profile_id}
              </Link>
              <Badge status={o.status} />
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  )
}
