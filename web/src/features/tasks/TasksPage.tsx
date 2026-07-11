import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { tasksApi } from '@/api/tasks'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { DataTable, EmptyState, ErrorAlert, PageHeader } from '@/components/shared/page'
import { Skeleton } from '@/components/ui/badge'
import { formatDate } from '@/lib/utils'

export function TasksPage() {
  const qc = useQueryClient()
  const { data, isLoading, isError, refetch } = useQuery({ queryKey: ['tasks'], queryFn: tasksApi.list })

  const run = useMutation({
    mutationFn: (id: string) => tasksApi.run(id),
    onSuccess: () => {
      toast.success('Task started')
      void qc.invalidateQueries({ queryKey: ['operations'] })
    },
    onError: (e: Error) => toast.error(e.message),
  })

  if (isLoading) return <Skeleton className="h-40 w-full" />
  if (isError) return <ErrorAlert message="Could not load tasks" onRetry={() => void refetch()} />

  const items = data?.items ?? []
  return (
    <div>
      <PageHeader title="Tasks" description="Scheduled backup workflows" />
      {items.length === 0 ? (
        <EmptyState title="No tasks" description="Create a task via API with cron or interval triggers." />
      ) : (
        <DataTable
          headers={['Name', 'Trigger', 'Enabled', 'Next run', 'Actions']}
          rows={items.map((t) => [
            t.name,
            t.trigger.type,
            t.enabled ? 'Yes' : 'No',
            formatDate(t.state?.next_run_at),
            <div key={t.id} className="flex gap-2">
              <Button size="sm" variant="outline" disabled={run.isPending} onClick={() => run.mutate(t.id)}>
                Run now
              </Button>
            </div>,
          ])}
        />
      )}
    </div>
  )
}

export function TaskDetailPage({ id }: { id: string }) {
  const { data, isLoading, isError } = useQuery({ queryKey: ['tasks', id], queryFn: () => tasksApi.get(id) })

  if (isLoading) return <Skeleton className="h-40 w-full" />
  if (isError || !data) return <ErrorAlert message="Task not found" />

  return (
    <div>
      <PageHeader title={data.name} description={`Trigger: ${data.trigger.type}`} />
      <div className="mb-4">
        <Badge status={data.enabled ? 'succeeded' : 'canceled'} />
        <span className="ml-2 text-sm">{data.enabled ? 'Enabled' : 'Disabled'}</span>
      </div>
      <h2 className="mb-2 font-medium">Workflow</h2>
      <ol className="list-decimal space-y-2 pl-5 text-sm">
        {data.actions.map((a, i) => (
          <li key={i}>{a.operation}</li>
        ))}
      </ol>
      <p className="mt-4 text-sm text-[hsl(var(--muted-foreground))]">
        Profiles: {data.profile_ids.join(', ')}
      </p>
      {data.state?.next_run_at ? (
        <p className="text-sm text-[hsl(var(--muted-foreground))]">Next run: {formatDate(data.state.next_run_at)}</p>
      ) : null}
    </div>
  )
}
