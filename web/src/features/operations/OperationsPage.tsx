import { useState, useMemo } from 'react'
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
import { useFormatDate } from '@/lib/datetime'
import { useOperationsSSE } from './useOperationsSSE'
import { CreateOperationForm } from './CreateOperationForm'

export function OperationsPage() {
  useOperationsSSE(true)
  const formatDate = useFormatDate()
  const [createOpen, setCreateOpen] = useState(false)
  const hosts = useQuery({ queryKey: ['hosts'], queryFn: hostsApi.list })

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['operations'],
    queryFn: operationsApi.list,
    refetchInterval: 10_000,
  })

  const items = data?.items ?? []

  const columns: ColumnDef<Operation>[] = useMemo(() => [
    { accessorKey: 'kind', header: 'Kind' },
    {
      accessorKey: 'profile_name',
      header: 'Host',
      cell: ({ row }) => row.original.profile_name ?? row.original.profile_id ?? '—',
    },
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
  ], [formatDate])

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
        <DataTable columns={columns} data={items} defaultSortDesc="started_at" />
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
  const formatDate = useFormatDate()
  const qc = useQueryClient()
  useOperationsSSE(true, id)
  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['operations', id],
    queryFn: () => operationsApi.get(id),
    refetchInterval: (q) => (q.state.data?.status === 'running' ? 2000 : false),
  })
  const logs = useQuery({
    queryKey: ['operations', id, 'logs'],
    queryFn: () => operationsApi.logs(id),
    refetchInterval: data?.status === 'running' || data?.status === 'queued' ? 3000 : false,
  })

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

  const hostName = data.profile_name ?? data.profile_id
  const logItems = logs.data?.items ?? []

  return (
    <div>
      <PageHeader
        title={data.kind}
        description={hostName}
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
        {data.progress ? <span className="text-sm">{data.progress}</span> : null}
      </div>

      <dl className="mb-6 grid max-w-2xl gap-3 text-sm sm:grid-cols-2">
        <div><dt className="text-[hsl(var(--muted-foreground))]">Host</dt><dd>{hostName}</dd></div>
        <div><dt className="text-[hsl(var(--muted-foreground))]">Started</dt><dd>{formatDate(data.started_at)}</dd></div>
        <div><dt className="text-[hsl(var(--muted-foreground))]">Finished</dt><dd>{formatDate(data.finished_at)}</dd></div>
        <div><dt className="text-[hsl(var(--muted-foreground))]">Trigger</dt><dd>{data.trigger_ref ?? '—'}</dd></div>
      </dl>

      {data.error ? <p className="mb-4 text-sm text-[hsl(var(--destructive))]">{data.error}</p> : null}

      {(data.artifacts?.length ?? 0) > 0 ? (
        <div className="mb-6">
          <h2 className="mb-2 text-lg font-medium">Artifacts</h2>
          <ul className="space-y-1 text-sm">
            {data.artifacts?.map((a, i) => (
              <li key={i} className="font-mono text-xs">
                {a.type}{a.id ? ` · ${a.id}` : ''}{a.path ? ` · ${a.path}` : ''}
              </li>
            ))}
          </ul>
        </div>
      ) : null}

      <h2 className="mb-2 text-lg font-medium">Logs</h2>
      <div className="max-h-96 overflow-auto rounded border border-[hsl(var(--border))] text-xs">
        {logItems.length === 0 ? (
          <p className="p-3 text-[hsl(var(--muted-foreground))]">No log entries</p>
        ) : (
          <table className="w-full">
            <thead className="bg-[hsl(var(--muted)/0.5)] text-left">
              <tr>
                <th className="px-3 py-2 font-medium">Time</th>
                <th className="px-3 py-2 font-medium">Action</th>
                <th className="px-3 py-2 font-medium">Phase</th>
                <th className="px-3 py-2 font-medium">Status</th>
                <th className="px-3 py-2 font-medium">Details</th>
              </tr>
            </thead>
            <tbody>
              {logItems.map((entry) => (
                <tr key={entry.id} className="border-t border-[hsl(var(--border))]">
                  <td className="px-3 py-2 whitespace-nowrap text-[hsl(var(--muted-foreground))]">{formatDate(entry.timestamp)}</td>
                  <td className="px-3 py-2">{entry.action}</td>
                  <td className="px-3 py-2">{entry.phase ?? '—'}</td>
                  <td className="px-3 py-2">{entry.status ?? '—'}</td>
                  <td className="px-3 py-2">
                    <div>{entry.details}</div>
                    {entry.error ? <div className="text-[hsl(var(--destructive))]">{entry.error}</div> : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}
