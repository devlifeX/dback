import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { toast } from 'sonner'
import type { ColumnDef } from '@tanstack/react-table'
import { backupsApi } from '@/api/backups'
import { hostsApi } from '@/api/hosts'
import { operationsApi } from '@/api/operations'
import type { ExportRecord, Host } from '@/api/types'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { DataTable } from '@/components/shared/data-table'
import { EmptyState, ErrorAlert, PageHeader } from '@/components/shared/page'
import { Skeleton } from '@/components/ui/badge'
import { formatDate, formatBytes } from '@/lib/utils'

export function BackupsPage() {
  const [hostFilter, setHostFilter] = useState('all')
  const { data, isLoading, isError, refetch } = useQuery({ queryKey: ['backups'], queryFn: () => backupsApi.list({ limit: 200 }) })
  const hosts = useQuery({ queryKey: ['hosts'], queryFn: hostsApi.list })

  const items = data?.items ?? []
  const hostItems = hosts.data?.items ?? []
  const filtered = hostFilter === 'all' ? items : items.filter((b) => b.profile_id === hostFilter)

  const columns: ColumnDef<ExportRecord>[] = [
    {
      accessorKey: 'profile_name',
      header: 'Host',
      cell: ({ row }) => (
        <Link to={`/backups/${row.original.id}`} className="font-medium hover:underline">{row.original.profile_name}</Link>
      ),
    },
    { accessorKey: 'database_name', header: 'Database' },
    { accessorKey: 'export_type', header: 'Type', cell: ({ row }) => row.original.export_type ?? 'database' },
    {
      accessorKey: 'export_date',
      header: 'Date',
      cell: ({ row }) => formatDate(row.original.export_date),
    },
    {
      accessorKey: 'file_size_bytes',
      header: 'Size',
      cell: ({ row }) => formatBytes(row.original.file_size_bytes),
    },
    {
      id: 'verified',
      header: 'Verified',
      cell: ({ row }) => {
        const q = row.original.quick_verified
        return q ? <Badge status={q.passed ? 'succeeded' : 'failed'} /> : '—'
      },
    },
  ]

  if (isLoading) return <Skeleton className="h-40 w-full" />
  if (isError) return <ErrorAlert message="Could not load backups" onRetry={() => void refetch()} />

  return (
    <div>
      <PageHeader
        title="Backups"
        description={`${items.length} backup records`}
        actions={
          <Select value={hostFilter} onValueChange={setHostFilter}>
            <SelectTrigger className="w-48"><SelectValue placeholder="Filter by host" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All hosts</SelectItem>
              {hostItems.map((h) => <SelectItem key={h.id} value={h.id}>{h.name}</SelectItem>)}
            </SelectContent>
          </Select>
        }
      />
      {items.length === 0 ? (
        <EmptyState title="No backups" description="Run a backup from Hosts to create records." action={<Button asChild><Link to="/hosts">Go to hosts</Link></Button>} />
      ) : (
        <DataTable columns={columns} data={filtered} />
      )}
    </div>
  )
}

export function BackupDetailPage({ id }: { id: string }) {
  const qc = useQueryClient()
  const [restoreOpen, setRestoreOpen] = useState(false)
  const [destId, setDestId] = useState('')

  const { data, isLoading, isError, refetch } = useQuery({ queryKey: ['backups', id], queryFn: () => backupsApi.get(id) })
  const hosts = useQuery({ queryKey: ['hosts'], queryFn: hostsApi.list })

  const verify = useMutation({
    mutationFn: () => backupsApi.quickVerify(id),
    onSuccess: () => { toast.success('Quick verify complete'); void refetch() },
    onError: (e: Error) => toast.error(e.message),
  })

  const download = useMutation({
    mutationFn: () => backupsApi.download(id),
    onSuccess: (blob) => {
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = data?.file_path?.split('/').pop() ?? 'backup.sql.gz'
      a.click()
      URL.revokeObjectURL(url)
    },
    onError: (e: Error) => toast.error(e.message),
  })

  const restore = useMutation({
    mutationFn: () => operationsApi.create('restore', destId, { record_id: id, destination_profile_id: destId }),
    onSuccess: () => {
      toast.success('Restore queued')
      setRestoreOpen(false)
      void qc.invalidateQueries({ queryKey: ['operations'] })
    },
    onError: (e: Error) => toast.error(e.message),
  })

  const deepVerify = useMutation({
    mutationFn: () => operationsApi.create('deep_verify', destId, { record_id: id, destination_profile_id: destId }),
    onSuccess: () => {
      toast.success('Deep verify queued')
      void qc.invalidateQueries({ queryKey: ['operations'] })
    },
    onError: (e: Error) => toast.error(e.message),
  })

  if (isLoading) return <Skeleton className="h-40 w-full" />
  if (isError || !data) return <ErrorAlert message="Backup not found" onRetry={() => void refetch()} />

  const hostList = hosts.data?.items ?? []

  return (
    <div>
      <PageHeader
        title={data.profile_name}
        description={`${data.database_name} · ${formatDate(data.export_date)}`}
        actions={
          <>
            <Button size="sm" variant="outline" disabled={verify.isPending} onClick={() => verify.mutate()}>Quick verify</Button>
            <Button size="sm" variant="outline" disabled={download.isPending} onClick={() => download.mutate()}>Download</Button>
            <Button size="sm" onClick={() => setRestoreOpen(true)}>Restore</Button>
            <Button size="sm" variant="outline" onClick={() => {
              if (!destId && hostList[0]) setDestId(hostList[0].id)
              deepVerify.mutate()
            }} disabled={deepVerify.isPending}>Deep verify</Button>
          </>
        }
      />
      <dl className="grid max-w-2xl gap-3 text-sm sm:grid-cols-2">
        <div><dt className="text-[hsl(var(--muted-foreground))]">Type</dt><dd>{data.export_type ?? 'database'}</dd></div>
        <div><dt className="text-[hsl(var(--muted-foreground))]">Size</dt><dd>{data.file_size || formatBytes(data.file_size_bytes)}</dd></div>
        <div><dt className="text-[hsl(var(--muted-foreground))]">SHA256</dt><dd className="font-mono text-xs break-all">{data.sha256 ?? '—'}</dd></div>
        <div><dt className="text-[hsl(var(--muted-foreground))]">Quick verify</dt><dd>{data.quick_verified ? (data.quick_verified.passed ? 'Passed' : 'Failed') : '—'}</dd></div>
        <div><dt className="text-[hsl(var(--muted-foreground))]">Deep verify</dt><dd>{data.deep_verified ? (data.deep_verified.passed ? 'Passed' : 'Failed') : '—'}</dd></div>
      </dl>

      <Dialog open={restoreOpen} onOpenChange={setRestoreOpen}>
        <DialogContent>
          <DialogHeader><DialogTitle>Restore backup</DialogTitle></DialogHeader>
          <p className="text-sm text-[hsl(var(--muted-foreground))]">Select destination host for restore.</p>
          <Select value={destId} onValueChange={setDestId}>
            <SelectTrigger><SelectValue placeholder="Destination host" /></SelectTrigger>
            <SelectContent>
              {hostList.map((h: Host) => <SelectItem key={h.id} value={h.id}>{h.name}</SelectItem>)}
            </SelectContent>
          </Select>
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setRestoreOpen(false)}>Cancel</Button>
            <Button disabled={!destId || restore.isPending} onClick={() => restore.mutate()}>Start restore</Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
