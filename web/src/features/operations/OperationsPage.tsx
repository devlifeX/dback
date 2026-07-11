import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { toast } from 'sonner'
import { operationsApi } from '@/api/operations'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { DataTable, EmptyState, ErrorAlert, PageHeader } from '@/components/shared/page'
import { Skeleton } from '@/components/ui/badge'
import { formatDate } from '@/lib/utils'
import { useOperationsSSE } from './useOperationsSSE'

export function OperationsPage() {
  useOperationsSSE(true)
  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['operations'],
    queryFn: operationsApi.list,
    refetchInterval: 10_000,
  })

  if (isLoading) return <Skeleton className="h-40 w-full" />
  if (isError) return <ErrorAlert message="Could not load operations" onRetry={() => void refetch()} />

  const items = data?.items ?? []
  return (
    <div>
      <PageHeader title="Operations" description="Live operation queue and history" />
      {items.length === 0 ? (
        <EmptyState title="No operations" description="Run a backup from Hosts or schedule a task." />
      ) : (
        <DataTable
          headers={['Kind', 'Profile', 'Status', 'Started', '']}
          rows={items.map((o) => [
            o.kind,
            o.profile_id,
            <Badge key={`${o.id}-s`} status={o.status} />,
            formatDate(o.started_at),
            <Link key={`${o.id}-l`} to={`/operations/${o.id}`} className="text-[hsl(var(--primary))] hover:underline">
              View
            </Link>,
          ])}
        />
      )}
    </div>
  )
}

export function OperationDetailPage({ id }: { id: string }) {
  const qc = useQueryClient()
  useOperationsSSE(true)
  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['operations', id],
    queryFn: () => operationsApi.get(id),
    refetchInterval: (q) => (q.state.data?.status === 'running' ? 2000 : false),
  })
  const logs = useQuery({ queryKey: ['operations', id, 'logs'], queryFn: () => operationsApi.logs(id) })

  const cancel = useMutation({
    mutationFn: () => operationsApi.cancel(id),
    onSuccess: () => {
      toast.success('Canceled')
      void qc.invalidateQueries({ queryKey: ['operations'] })
    },
    onError: (e: Error) => toast.error(e.message),
  })

  const retry = useMutation({
    mutationFn: () => operationsApi.retry(id),
    onSuccess: () => {
      toast.success('Retry queued')
      void qc.invalidateQueries({ queryKey: ['operations'] })
    },
    onError: (e: Error) => toast.error(e.message),
  })

  if (isLoading) return <Skeleton className="h-40 w-full" />
  if (isError || !data) return <ErrorAlert message="Operation not found" onRetry={() => void refetch()} />

  return (
    <div>
      <PageHeader
        title={data.kind}
        description={data.id}
        actions={
          <>
            {data.status === 'running' || data.status === 'queued' ? (
              <Button variant="outline" size="sm" onClick={() => cancel.mutate()} disabled={cancel.isPending}>
                Cancel
              </Button>
            ) : null}
            {data.status === 'failed' ? (
              <Button size="sm" onClick={() => retry.mutate()} disabled={retry.isPending}>
                Retry
              </Button>
            ) : null}
          </>
        }
      />
      <div className="mb-4 flex gap-2">
        <Badge status={data.status} />
        <span className="text-sm text-[hsl(var(--muted-foreground))]">Profile {data.profile_id}</span>
      </div>
      {data.error ? <p className="mb-4 text-sm text-[hsl(var(--destructive))]">{data.error}</p> : null}
      <h2 className="mb-2 text-lg font-medium">Logs</h2>
      <div className="max-h-96 overflow-auto rounded border border-[hsl(var(--border))] p-3 font-mono text-xs">
        {(logs.data?.items ?? []).length === 0 ? (
          <p className="text-[hsl(var(--muted-foreground))]">No log entries</p>
        ) : (
          (logs.data?.items ?? []).map((entry, i) => (
            <div key={i} className="border-b border-[hsl(var(--border))] py-1 last:border-0">
              {String(entry.details ?? entry.action ?? JSON.stringify(entry))}
            </div>
          ))
        )}
      </div>
    </div>
  )
}
