import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { toast } from 'sonner'
import type { ColumnDef } from '@tanstack/react-table'
import { MoreHorizontal } from 'lucide-react'
import { hostsApi } from '@/api/hosts'
import { operationsApi } from '@/api/operations'
import type { Host, Profile } from '@/api/types'
import { useMutationWithRevision } from '@/hooks/use-vault-mutation'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { DataTable } from '@/components/shared/data-table'
import { EmptyState, ErrorAlert, PageHeader } from '@/components/shared/page'
import { Skeleton } from '@/components/ui/badge'
import { HostForm } from './HostForm'
import { CheckHostStatusDialog } from './CheckHostStatusDialog'
import { CreateOperationForm } from '@/features/operations/CreateOperationForm'

export function HostsPage() {
  const qc = useQueryClient()
  const [groupFilter, setGroupFilter] = useState<string>('all')
  const [dialog, setDialog] = useState<'create' | 'edit' | 'delete' | null>(null)
  const [statusHost, setStatusHost] = useState<Host | null>(null)
  const [selected, setSelected] = useState<Host | null>(null)

  const { data, isLoading, isError, refetch } = useQuery({ queryKey: ['hosts'], queryFn: hostsApi.list })

  const save = useMutationWithRevision({
    mutationFn: (host: Profile, etag) => hostsApi.save(host, etag),
    invalidateKeys: [['hosts']],
    onSuccess: () => { toast.success('Host saved'); setDialog(null); setSelected(null) },
  })

  const remove = useMutationWithRevision({
    mutationFn: (id: string, etag) => hostsApi.remove(id, etag),
    invalidateKeys: [['hosts']],
    onSuccess: () => { toast.success('Host deleted'); setDialog(null); setSelected(null) },
  })

  const duplicate = useMutationWithRevision({
    mutationFn: (id: string, etag) => hostsApi.duplicate(id, etag),
    invalidateKeys: [['hosts']],
    onSuccess: () => toast.success('Host duplicated'),
  })

  const test = useMutation({
    mutationFn: (id: string) => hostsApi.testConnection(id),
    onSuccess: () => toast.success('Connection OK'),
    onError: (e: Error) => toast.error(e.message),
  })

  const runBackup = useMutation({
    mutationFn: (profileId: string) => operationsApi.create('backup_db', profileId),
    onSuccess: () => { toast.success('Backup queued'); void qc.invalidateQueries({ queryKey: ['operations'] }) },
    onError: (e: Error) => toast.error(e.message),
  })

  const runFileBackup = useMutation({
    mutationFn: (profileId: string) => operationsApi.create('backup_files', profileId),
    onSuccess: () => { toast.success('File backup queued'); void qc.invalidateQueries({ queryKey: ['operations'] }) },
    onError: (e: Error) => toast.error(e.message),
  })

  const items = data?.items ?? []
  const groups = useMemo(() => {
    const set = new Set(items.map((h) => h.group || 'Default'))
    return ['all', ...Array.from(set).sort()]
  }, [items])

  const filtered = groupFilter === 'all' ? items : items.filter((h) => (h.group || 'Default') === groupFilter)

  const columns: ColumnDef<Host>[] = useMemo(() => [
    {
      accessorKey: 'name',
      header: 'Name',
      cell: ({ row }) => (
        <Link to={`/hosts/${row.original.id}`} className="font-medium hover:underline">{row.original.name}</Link>
      ),
    },
    { accessorKey: 'connection_type', header: 'Type' },
    { accessorKey: 'group', header: 'Group', cell: ({ row }) => row.original.group ?? 'Default' },
    {
      id: 'actions',
      header: 'Actions',
      cell: ({ row }) => (
        <div className="flex flex-wrap items-center gap-2">
          <Button size="sm" disabled={runBackup.isPending} onClick={() => runBackup.mutate(row.original.id)}>Backup</Button>
          <Button size="sm" variant="outline" onClick={() => setStatusHost(row.original)}>Check Host Status</Button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button size="sm" variant="outline" className="h-8 w-8 p-0">
                <MoreHorizontal className="h-4 w-4" />
                <span className="sr-only">More actions</span>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem disabled={test.isPending} onClick={() => test.mutate(row.original.id)}>Test connection</DropdownMenuItem>
              <DropdownMenuItem disabled={runFileBackup.isPending} onClick={() => runFileBackup.mutate(row.original.id)}>File backup</DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={() => { setSelected(row.original); setDialog('edit') }}>Edit</DropdownMenuItem>
              <DropdownMenuItem disabled={duplicate.isPending} onClick={() => duplicate.mutate(row.original.id)}>Duplicate</DropdownMenuItem>
              <DropdownMenuItem className="text-[hsl(var(--destructive))]" onClick={() => { setSelected(row.original); setDialog('delete') }}>Delete</DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      ),
    },
  ], [test.isPending, runBackup.isPending, runFileBackup.isPending, duplicate.isPending])

  if (isLoading) return <Skeleton className="h-40 w-full" />
  if (isError) return <ErrorAlert message="Could not load hosts" onRetry={() => void refetch()} />

  return (
    <div>
      <PageHeader
        title="Hosts"
        description={`${items.length} profiles`}
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <Select value={groupFilter} onValueChange={setGroupFilter}>
              <SelectTrigger className="w-40"><SelectValue placeholder="Group" /></SelectTrigger>
              <SelectContent>
                {groups.map((g) => <SelectItem key={g} value={g}>{g === 'all' ? 'All groups' : g}</SelectItem>)}
              </SelectContent>
            </Select>
            <Button onClick={() => { setSelected(null); setDialog('create') }}>New host</Button>
          </div>
        }
      />
      {items.length === 0 ? (
        <EmptyState title="No hosts" description="Add a backup profile to get started." action={<Button onClick={() => setDialog('create')}>Create host</Button>} />
      ) : (
        <DataTable columns={columns} data={filtered} emptyMessage="No hosts in this group" />
      )}

      <Dialog open={dialog === 'create' || dialog === 'edit'} onOpenChange={(o) => !o && setDialog(null)}>
        <DialogContent className="max-h-[90vh] max-w-3xl overflow-y-auto">
          <DialogHeader><DialogTitle>{dialog === 'edit' ? 'Edit host' : 'New host'}</DialogTitle></DialogHeader>
          <HostForm profile={selected ?? undefined} pending={save.isPending} onCancel={() => setDialog(null)} onSubmit={(v) => save.mutate(v)} />
        </DialogContent>
      </Dialog>

      <Dialog open={dialog === 'delete'} onOpenChange={(o) => !o && setDialog(null)}>
        <DialogContent>
          <DialogHeader><DialogTitle>Delete host</DialogTitle></DialogHeader>
          <p className="text-sm">Delete &quot;{selected?.name}&quot;?</p>
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setDialog(null)}>Cancel</Button>
            <Button variant="destructive" disabled={remove.isPending} onClick={() => selected && remove.mutate(selected.id)}>Delete</Button>
          </div>
        </DialogContent>
      </Dialog>

      {statusHost ? (
        <CheckHostStatusDialog
          hostId={statusHost.id}
          hostName={statusHost.name}
          primaryUrl={statusHost.url_check?.primary?.url}
          open={!!statusHost}
          onOpenChange={(o) => !o && setStatusHost(null)}
        />
      ) : null}
    </div>
  )
}

export function HostDetailPage({ id }: { id: string }) {
  const qc = useQueryClient()
  const [editing, setEditing] = useState(false)
  const [opDialog, setOpDialog] = useState(false)
  const [statusOpen, setStatusOpen] = useState(false)

  const { data, isLoading, isError, refetch } = useQuery({ queryKey: ['hosts', id], queryFn: () => hostsApi.get(id) })
  const hosts = useQuery({ queryKey: ['hosts'], queryFn: hostsApi.list })

  const save = useMutationWithRevision({
    mutationFn: (host: Profile, etag) => hostsApi.save(host, etag),
    invalidateKeys: [['hosts'], ['hosts', id]],
    onSuccess: () => { toast.success('Host saved'); setEditing(false) },
  })

  const test = useMutation({
    mutationFn: () => hostsApi.testConnection(id),
    onSuccess: () => toast.success('Connection OK'),
    onError: (e: Error) => toast.error(e.message),
  })

  const genKey = useMutationWithRevision({
    mutationFn: (_: void, etag) => hostsApi.generateWPKey(id, etag),
    invalidateKeys: [['hosts', id]],
    onSuccess: () => toast.success('WordPress key generated'),
  })

  const downloadPlugin = useMutation({
    mutationFn: () => hostsApi.downloadPlugin(id),
    onSuccess: (blob) => {
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = 'dback-wp-plugin.zip'
      a.click()
      URL.revokeObjectURL(url)
    },
    onError: (e: Error) => toast.error(e.message),
  })

  if (isLoading) return <Skeleton className="h-40 w-full" />
  if (isError || !data) return <ErrorAlert message="Host not found" onRetry={() => void refetch()} />

  return (
    <div>
      <PageHeader
        title={data.name}
        description={data.connection_type}
        actions={
          <>
            <Button variant="outline" size="sm" disabled={test.isPending} onClick={() => test.mutate()}>Test connection</Button>
            <Button variant="outline" size="sm" onClick={() => setStatusOpen(true)}>Check Host Status</Button>
            <Button variant="outline" size="sm" onClick={() => setOpDialog(true)}>Run operation</Button>
            <Button size="sm" onClick={() => setEditing((e) => !e)}>{editing ? 'Cancel edit' : 'Edit'}</Button>
          </>
        }
      />

      {editing ? (
        <HostForm profile={data} pending={save.isPending} onCancel={() => setEditing(false)} onSubmit={(v) => save.mutate({ ...v, id })} />
      ) : (
        <dl className="grid max-w-2xl gap-3 text-sm sm:grid-cols-2">
          <div><dt className="text-[hsl(var(--muted-foreground))]">Host</dt><dd>{data.host || '—'}</dd></div>
          <div><dt className="text-[hsl(var(--muted-foreground))]">Database</dt><dd>{data.target_db_name || '—'}</dd></div>
          <div><dt className="text-[hsl(var(--muted-foreground))]">Group</dt><dd>{data.group ?? 'Default'}</dd></div>
          <div><dt className="text-[hsl(var(--muted-foreground))]">File backup</dt><dd>{data.file_backup_enabled ? 'Enabled' : 'Disabled'}</dd></div>
        </dl>
      )}

      {data.connection_type === 'WordPress' ? (
        <div className="mt-4 flex gap-2">
          <Button size="sm" variant="outline" disabled={genKey.isPending} onClick={() => genKey.mutate()}>Generate WP key</Button>
          <Button size="sm" variant="outline" disabled={downloadPlugin.isPending} onClick={() => downloadPlugin.mutate()}>Download plugin</Button>
        </div>
      ) : null}

      <Dialog open={opDialog} onOpenChange={setOpDialog}>
        <DialogContent>
          <DialogHeader><DialogTitle>Run operation</DialogTitle></DialogHeader>
          <CreateOperationForm
            defaultProfileId={id}
            hosts={hosts.data?.items ?? []}
            onSuccess={() => { setOpDialog(false); void qc.invalidateQueries({ queryKey: ['operations'] }) }}
            onCancel={() => setOpDialog(false)}
          />
        </DialogContent>
      </Dialog>

      <CheckHostStatusDialog
        hostId={id}
        hostName={data.name}
        primaryUrl={data.url_check?.primary?.url}
        open={statusOpen}
        onOpenChange={setStatusOpen}
      />
    </div>
  )
}
