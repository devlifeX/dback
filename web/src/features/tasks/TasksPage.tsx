import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { toast } from 'sonner'
import type { ColumnDef } from '@tanstack/react-table'
import { hostsApi } from '@/api/hosts'
import { tasksApi } from '@/api/tasks'
import type { Task } from '@/api/types'
import { useMutationWithRevision } from '@/hooks/use-vault-mutation'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { DataTable } from '@/components/shared/data-table'
import { EmptyState, ErrorAlert, PageHeader } from '@/components/shared/page'
import { Skeleton } from '@/components/ui/badge'
import { formatDate } from '@/lib/utils'
import { TaskForm } from './TaskForm'

export function TasksPage() {
  const qc = useQueryClient()
  const [dialog, setDialog] = useState<'create' | 'edit' | 'delete' | 'clone' | null>(null)
  const [selected, setSelected] = useState<Task | null>(null)

  const { data, isLoading, isError, refetch } = useQuery({ queryKey: ['tasks'], queryFn: tasksApi.list })
  const hosts = useQuery({ queryKey: ['hosts'], queryFn: hostsApi.list })

  const save = useMutationWithRevision({
    mutationFn: (task: Task, etag) => tasksApi.save(task, etag),
    invalidateKeys: [['tasks']],
    onSuccess: () => { toast.success('Task saved'); setDialog(null); setSelected(null) },
  })

  const remove = useMutationWithRevision({
    mutationFn: (id: string, etag) => tasksApi.remove(id, etag),
    invalidateKeys: [['tasks']],
    onSuccess: () => { toast.success('Task deleted'); setDialog(null); setSelected(null) },
  })

  const toggle = useMutationWithRevision({
    mutationFn: (task: Task, etag) => tasksApi.save({ ...task, enabled: !task.enabled }, etag),
    invalidateKeys: [['tasks']],
    onSuccess: () => toast.success('Task updated'),
  })

  const run = useMutation({
    mutationFn: (id: string) => tasksApi.run(id),
    onSuccess: () => { toast.success('Task started'); void qc.invalidateQueries({ queryKey: ['operations'] }) },
    onError: (e: Error) => toast.error(e.message),
  })

  const items = data?.items ?? []

  const columns: ColumnDef<Task>[] = [
    {
      accessorKey: 'name',
      header: 'Name',
      cell: ({ row }) => <Link to={`/tasks/${row.original.id}`} className="font-medium hover:underline">{row.original.name}</Link>,
    },
    { accessorKey: 'trigger.type', header: 'Trigger', cell: ({ row }) => row.original.trigger.type },
    {
      accessorKey: 'enabled',
      header: 'Enabled',
      cell: ({ row }) => <Badge status={row.original.enabled ? 'succeeded' : 'canceled'} />,
    },
    {
      id: 'next',
      header: 'Next run',
      cell: ({ row }) => formatDate(row.original.state?.next_run_at),
    },
    {
      id: 'actions',
      header: 'Actions',
      cell: ({ row }) => (
        <div className="flex flex-wrap gap-2">
          <Button size="sm" variant="outline" disabled={run.isPending} onClick={() => run.mutate(row.original.id)}>Run</Button>
          <Button size="sm" variant="outline" onClick={() => toggle.mutate(row.original)}>{row.original.enabled ? 'Disable' : 'Enable'}</Button>
          <Button size="sm" variant="outline" onClick={() => { setSelected(row.original); setDialog('edit') }}>Edit</Button>
          <Button size="sm" variant="outline" onClick={() => { setSelected({ ...row.original, id: '', name: `${row.original.name} (copy)` }); setDialog('clone') }}>Clone</Button>
          <Button size="sm" variant="destructive" onClick={() => { setSelected(row.original); setDialog('delete') }}>Delete</Button>
        </div>
      ),
    },
  ]

  if (isLoading) return <Skeleton className="h-40 w-full" />
  if (isError) return <ErrorAlert message="Could not load tasks" onRetry={() => void refetch()} />

  return (
    <div>
      <PageHeader title="Tasks" description="Scheduled backup workflows" actions={<Button onClick={() => { setSelected(null); setDialog('create') }}>New task</Button>} />
      {items.length === 0 ? (
        <EmptyState title="No tasks" description="Schedule automated backup workflows." action={<Button onClick={() => setDialog('create')}>Create task</Button>} />
      ) : (
        <DataTable columns={columns} data={items} />
      )}

      <Dialog open={dialog === 'create' || dialog === 'edit' || dialog === 'clone'} onOpenChange={(o) => !o && setDialog(null)}>
        <DialogContent className="max-h-[90vh] max-w-2xl overflow-y-auto">
          <DialogHeader><DialogTitle>{dialog === 'edit' ? 'Edit task' : dialog === 'clone' ? 'Clone task' : 'New task'}</DialogTitle></DialogHeader>
          <TaskForm
            task={selected ?? undefined}
            hosts={hosts.data?.items ?? []}
            pending={save.isPending}
            onCancel={() => setDialog(null)}
            onSubmit={(v) => save.mutate(v)}
          />
        </DialogContent>
      </Dialog>

      <Dialog open={dialog === 'delete'} onOpenChange={(o) => !o && setDialog(null)}>
        <DialogContent>
          <DialogHeader><DialogTitle>Delete task</DialogTitle></DialogHeader>
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

export function TaskDetailPage({ id }: { id: string }) {
  const { data, isLoading, isError } = useQuery({ queryKey: ['tasks', id], queryFn: () => tasksApi.get(id) })
  const runs = useQuery({ queryKey: ['tasks', id, 'runs'], queryFn: () => tasksApi.runs(id) })

  if (isLoading) return <Skeleton className="h-40 w-full" />
  if (isError || !data) return <ErrorAlert message="Task not found" />

  const runItems = runs.data?.items ?? []

  return (
    <div>
      <PageHeader title={data.name} description={`Trigger: ${data.trigger.type}`} />
      <div className="mb-4">
        <Badge status={data.enabled ? 'succeeded' : 'canceled'} />
        <span className="ml-2 text-sm">{data.enabled ? 'Enabled' : 'Disabled'}</span>
      </div>
      <h2 className="mb-2 font-medium">Workflow</h2>
      <ol className="list-decimal space-y-2 pl-5 text-sm">
        {data.actions.map((a, i) => <li key={i}>{a.operation}</li>)}
      </ol>
      <p className="mt-4 text-sm text-[hsl(var(--muted-foreground))]">Profiles: {data.profile_ids.join(', ')}</p>
      {data.state?.next_run_at ? <p className="text-sm text-[hsl(var(--muted-foreground))]">Next run: {formatDate(data.state.next_run_at)}</p> : null}

      <h2 className="mb-3 mt-8 text-lg font-medium">Run history</h2>
      {runItems.length === 0 ? (
        <p className="text-sm text-[hsl(var(--muted-foreground))]">No runs yet</p>
      ) : (
        <div className="space-y-2">
          {runItems.map((r) => (
            <div key={r.id} className="flex items-center justify-between rounded border border-[hsl(var(--border))] px-3 py-2 text-sm">
              <span>{formatDate(r.started_at)} · {r.profile_id}</span>
              <Badge status={r.status} />
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
