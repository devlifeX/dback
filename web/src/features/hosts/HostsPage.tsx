import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { toast } from 'sonner'
import { hostsApi } from '@/api/hosts'
import { operationsApi } from '@/api/operations'
import { Button } from '@/components/ui/button'
import { DataTable, EmptyState, ErrorAlert, PageHeader } from '@/components/shared/page'
import { Skeleton } from '@/components/ui/badge'

export function HostsPage() {
  const qc = useQueryClient()
  const { data, isLoading, isError, refetch } = useQuery({ queryKey: ['hosts'], queryFn: hostsApi.list })

  const test = useMutation({
    mutationFn: (id: string) => hostsApi.testConnection(id),
    onSuccess: () => toast.success('Connection OK'),
    onError: (e: Error) => toast.error(e.message),
  })

  const runBackup = useMutation({
    mutationFn: (profileId: string) => operationsApi.create('backup_db', profileId),
    onSuccess: () => {
      toast.success('Backup queued')
      void qc.invalidateQueries({ queryKey: ['operations'] })
    },
    onError: (e: Error) => toast.error(e.message),
  })

  if (isLoading) return <Skeleton className="h-40 w-full" />
  if (isError) return <ErrorAlert message="Could not load hosts" onRetry={() => void refetch()} />

  const items = data?.items ?? []
  if (items.length === 0) {
    return (
      <div>
        <PageHeader title="Hosts" description="Manage backup profiles" />
        <EmptyState title="No hosts" description="Add hosts via API or import from desktop vault." />
      </div>
    )
  }

  return (
    <div>
      <PageHeader title="Hosts" description={`${items.length} profiles`} />
      <DataTable
        headers={['Name', 'Type', 'Group', 'Actions']}
        rows={items.map((h) => [
          <Link key={h.id} to={`/hosts/${h.id}`} className="font-medium hover:underline">
            {h.name}
          </Link>,
          h.connection_type,
          h.group ?? 'Default',
          <div key={`${h.id}-actions`} className="flex gap-2">
            <Button size="sm" variant="outline" disabled={test.isPending} onClick={() => test.mutate(h.id)}>
              Test
            </Button>
            <Button size="sm" disabled={runBackup.isPending} onClick={() => runBackup.mutate(h.id)}>
              Backup
            </Button>
          </div>,
        ])}
      />
    </div>
  )
}

export function HostDetailPage({ id }: { id: string }) {
  const { data, isLoading, isError } = useQuery({
    queryKey: ['hosts', id],
    queryFn: () => hostsApi.get(id),
  })

  if (isLoading) return <Skeleton className="h-40 w-full" />
  if (isError || !data) return <ErrorAlert message="Host not found" />

  return (
    <div>
      <PageHeader title={data.name} description={data.connection_type} />
      <dl className="grid max-w-xl gap-3 text-sm">
        <div>
          <dt className="text-[hsl(var(--muted-foreground))]">Host</dt>
          <dd>{data.host || '—'}</dd>
        </div>
        <div>
          <dt className="text-[hsl(var(--muted-foreground))]">Database</dt>
          <dd>{data.target_db_name || '—'}</dd>
        </div>
        <div>
          <dt className="text-[hsl(var(--muted-foreground))]">Group</dt>
          <dd>{data.group ?? 'Default'}</dd>
        </div>
      </dl>
    </div>
  )
}
