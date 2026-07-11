import { useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { toast } from 'sonner'
import type { ColumnDef } from '@tanstack/react-table'
import { notificationsApi } from '@/api/notifications'
import type { NotifyChannel } from '@/api/types'
import { useMutationWithRevision } from '@/hooks/use-vault-mutation'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { DataTable } from '@/components/shared/data-table'
import { EmptyState, ErrorAlert, PageHeader } from '@/components/shared/page'
import { Skeleton } from '@/components/ui/badge'
import { NotificationForm } from './NotificationForm'

export function NotificationsPage() {
  const [dialog, setDialog] = useState<'create' | 'edit' | 'delete' | null>(null)
  const [selected, setSelected] = useState<NotifyChannel | null>(null)

  const { data, isLoading, isError, refetch } = useQuery({ queryKey: ['notifications'], queryFn: notificationsApi.list })

  const save = useMutationWithRevision({
    mutationFn: (ch: NotifyChannel, etag) => notificationsApi.save(ch, etag),
    invalidateKeys: [['notifications']],
    onSuccess: () => { toast.success('Channel saved'); setDialog(null); setSelected(null) },
  })

  const remove = useMutationWithRevision({
    mutationFn: (id: string, etag) => notificationsApi.remove(id, etag),
    invalidateKeys: [['notifications']],
    onSuccess: () => { toast.success('Channel deleted'); setDialog(null); setSelected(null) },
  })

  const test = useMutation({
    mutationFn: (id: string) => notificationsApi.test(id),
    onSuccess: () => toast.success('Test notification sent'),
    onError: (e: Error) => toast.error(e.message),
  })

  const items = data?.items ?? []

  const columns: ColumnDef<NotifyChannel>[] = [
    {
      accessorKey: 'name',
      header: 'Name',
      cell: ({ row }) => <Link to={`/notifications/${row.original.id}`} className="font-medium hover:underline">{row.original.name}</Link>,
    },
    { accessorKey: 'provider', header: 'Provider' },
    {
      accessorKey: 'enabled',
      header: 'Status',
      cell: ({ row }) => (row.original.enabled ? 'Enabled' : 'Disabled'),
    },
    {
      accessorKey: 'events',
      header: 'Events',
      cell: ({ row }) => ((row.original.events ?? []).length === 0 ? 'all' : (row.original.events ?? []).join(', ')),
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }) => (
        <div className="flex gap-2">
          <Button size="sm" variant="outline" disabled={test.isPending} onClick={() => test.mutate(row.original.id)}>Test</Button>
          <Button size="sm" variant="outline" onClick={() => { setSelected(row.original); setDialog('edit') }}>Edit</Button>
          <Button size="sm" variant="destructive" onClick={() => { setSelected(row.original); setDialog('delete') }}>Delete</Button>
        </div>
      ),
    },
  ]

  if (isLoading) return <Skeleton className="h-40 w-full" />
  if (isError) return <ErrorAlert message="Could not load channels" onRetry={() => void refetch()} />

  return (
    <div>
      <PageHeader title="Notifications" description="Alert channels for operation events" actions={<Button onClick={() => { setSelected(null); setDialog('create') }}>New channel</Button>} />
      {items.length === 0 ? (
        <EmptyState title="No channels" description="Configure Telegram, Slack, Bale, or Webhook alerts." action={<Button onClick={() => setDialog('create')}>Add channel</Button>} />
      ) : (
        <DataTable columns={columns} data={items} />
      )}

      <Dialog open={dialog === 'create' || dialog === 'edit'} onOpenChange={(o) => !o && setDialog(null)}>
        <DialogContent className="max-h-[90vh] max-w-lg overflow-y-auto">
          <DialogHeader><DialogTitle>{dialog === 'edit' ? 'Edit channel' : 'New channel'}</DialogTitle></DialogHeader>
          <NotificationForm channel={selected ?? undefined} pending={save.isPending} onCancel={() => setDialog(null)} onSubmit={(v) => save.mutate(v)} />
        </DialogContent>
      </Dialog>

      <Dialog open={dialog === 'delete'} onOpenChange={(o) => !o && setDialog(null)}>
        <DialogContent>
          <DialogHeader><DialogTitle>Delete channel</DialogTitle></DialogHeader>
          <p className="text-sm">Delete &quot;{selected?.name}&quot;?</p>
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setDialog(null)}>Cancel</Button>
            <Button variant="destructive" disabled={remove.isPending} onClick={() => selected && remove.mutate(selected.id)}>Delete</Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}

export function NotificationDetailPage({ id }: { id: string }) {
  const [editing, setEditing] = useState(false)
  const { data, isLoading, isError, refetch } = useQuery({ queryKey: ['notifications', id], queryFn: () => notificationsApi.get(id) })

  const save = useMutationWithRevision({
    mutationFn: (ch: NotifyChannel, etag) => notificationsApi.save(ch, etag),
    invalidateKeys: [['notifications'], ['notifications', id]],
    onSuccess: () => { toast.success('Saved'); setEditing(false) },
  })

  const test = useMutation({
    mutationFn: () => notificationsApi.test(id),
    onSuccess: () => toast.success('Test sent'),
    onError: (e: Error) => toast.error(e.message),
  })

  if (isLoading) return <Skeleton className="h-40 w-full" />
  if (isError || !data) return <ErrorAlert message="Channel not found" onRetry={() => void refetch()} />

  return (
    <div>
      <PageHeader
        title={data.name}
        description={data.provider}
        actions={
          <>
            <Button size="sm" variant="outline" disabled={test.isPending} onClick={() => test.mutate()}>Test</Button>
            <Button size="sm" onClick={() => setEditing((e) => !e)}>{editing ? 'Cancel' : 'Edit'}</Button>
          </>
        }
      />
      {editing ? (
        <NotificationForm channel={data} pending={save.isPending} onCancel={() => setEditing(false)} onSubmit={(v) => save.mutate({ ...v, id })} />
      ) : (
        <dl className="grid max-w-xl gap-3 text-sm">
          <div><dt className="text-[hsl(var(--muted-foreground))]">Status</dt><dd>{data.enabled ? 'Enabled' : 'Disabled'}</dd></div>
          <div><dt className="text-[hsl(var(--muted-foreground))]">Events</dt><dd>{(data.events ?? []).length === 0 ? 'All events' : data.events?.join(', ')}</dd></div>
        </dl>
      )}
    </div>
  )
}
