import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { toast } from 'sonner'
import type { ColumnDef } from '@tanstack/react-table'
import { hostsApi } from '@/api/hosts'
import { operationsApi } from '@/api/operations'
import type { Operation } from '@/api/types'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { DataTable } from '@/components/shared/data-table'
import { EmptyState, ErrorAlert, PageHeader } from '@/components/shared/page'
import { Skeleton } from '@/components/ui/badge'
import { formatDate } from '@/lib/utils'
import { useOperationsSSE } from './useOperationsSSE'
import { CreateOperationForm } from './CreateOperationForm'

export function OperationsPage() {
  useOperationsSSE(true)
  const [createOpen, setCreateOpen] = useState(false)
  const hosts = useQuery({ queryKey: ['hosts'], queryFn: hostsApi.list })

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['operations'],
    queryFn: operationsApi.list,
    refetchInterval: 10_000,
  })

  const items = data?.items ?? []

  const columns: ColumnDef<Operation>[] = [
    { accessorKey: 'kind', header: 'Kind' },
    { accessorKey: 'profile_id', header: 'Profile' },
    {
      accessorKey: 'status',
      header: 'Status',
      cell: ({ row }) => <Badge status={row.original.status} />,
    },
    {
      accessorKey: 'progress',
      header: 'Progress',
      cell: ({ row }) => row.original.progress ?? '—',
    },
    {
      accessorKey: 'started_at',
      header: 'Started',
      cell: ({ row }) => formatDate(row.original.started_at),
    },
    {
      id: 'view',
      header: '',
      cell: ({ row }) => (
        <Link to={`/operations/${row.original.id}`} className="text-[hsl(var(--primary))] hover:underline">View</Link>
      ),
    },
  ]

  if (isLoading) return <Skeleton className="h-40 w-full" />
  if (isError) return <ErrorAlert message="Could not load operations" onRetry={() => void refetch()} />

  return (
    <div>
      <PageHeader
        title="Operations"
        description="Live operation queue and history"
        actions={<Button onClick={() => setCreateOpen(true)}>New operation</Button>}
      />
      {items.length === 0 ? (
        <EmptyState
          title="No operations"
          description="Start a backup or other operation."
          action={<Button onClick={() => setCreateOpen(true)}>Create operation</Button>}
        />
      ) : (
        <DataTable columns={columns} data={items} />
      )}

      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent>
          <DialogHeader><DialogTitle>New operation</DialogTitle></DialogHeader>
          <CreateOperationForm
            hosts={hosts.data?.items ?? []}
            onCancel={() => setCreateOpen(false)}
            onSuccess={() => setCreateOpen(false)}
          />
        </DialogContent>
      </Dialog>
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
    onSuccess: () => { toast.success('Canceled'); void qc.invalidateQueries({ queryKey: ['operations'] }) },
    onError: (e: Error) => toast.error(e.message),
  })

  const retry = useMutation({
    mutationFn: () => operationsApi.retry(id),
    onSuccess: () => { toast.success('Retry queued'); void qc.invalidateQueries({ queryKey: ['operations'] }) },
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
              <Button variant="outline" size="sm" onClick={() => cancel.mutate()} disabled={cancel.isPending}>Cancel</Button>
            ) : null}
            {data.status === 'failed' ? (
              <Button size="sm" onClick={() => retry.mutate()} disabled={retry.isPending}>Retry</Button>
            ) : null}
          </>
        }
      />
      <div className="mb-4 flex flex-wrap gap-2">
        <Badge status={data.status} />
        <span className="text-sm text-[hsl(var(--muted-foreground))]">Profile {data.profile_id}</span>
        {data.progress ? <span className="text-sm">{data.progress}</span> : null}
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
