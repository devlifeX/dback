import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import type { ColumnDef } from '@tanstack/react-table'
import { usersApi } from '@/api/users'
import type { User } from '@/api/types'
import { useMutationWithRevision } from '@/hooks/use-vault-mutation'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { DataTable } from '@/components/shared/data-table'
import { EmptyState, ErrorAlert, PageHeader } from '@/components/shared/page'
import { Skeleton } from '@/components/ui/badge'
import { UserForm } from './UserForm'
import { usersPath } from './users-nav'

export function UsersListPage() {
  const [dialog, setDialog] = useState<'edit' | 'delete' | null>(null)
  const [selected, setSelected] = useState<User | null>(null)

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['users'],
    queryFn: usersApi.list,
  })

  const update = useMutationWithRevision({
    mutationFn: (input: { id: string; values: Parameters<typeof usersApi.update>[1] }, etag) =>
      usersApi.update(input.id, input.values, etag),
    invalidateKeys: [['users']],
    onSuccess: () => {
      toast.success('User updated')
      setDialog(null)
      setSelected(null)
    },
  })

  const remove = useMutationWithRevision({
    mutationFn: (id: string, etag) => usersApi.remove(id, etag),
    invalidateKeys: [['users']],
    onSuccess: () => {
      toast.success('User deleted')
      setDialog(null)
      setSelected(null)
    },
  })

  const items = data?.items ?? []

  const columns: ColumnDef<User>[] = [
    { accessorKey: 'phone', header: 'Phone' },
    { accessorKey: 'name', header: 'Name' },
    {
      accessorKey: 'enabled',
      header: 'Status',
      cell: ({ row }) => (row.original.enabled ? 'Enabled' : 'Disabled'),
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }) => (
        <div className="flex gap-2">
          <Button
            size="sm"
            variant="outline"
            onClick={() => {
              setSelected(row.original)
              setDialog('edit')
            }}
          >
            Edit
          </Button>
          <Button
            size="sm"
            variant="destructive"
            onClick={() => {
              setSelected(row.original)
              setDialog('delete')
            }}
          >
            Delete
          </Button>
        </div>
      ),
    },
  ]

  if (isLoading) return <Skeleton className="h-40 w-full" />
  if (isError) return <ErrorAlert message="Could not load users" onRetry={() => void refetch()} />

  return (
    <div>
      <PageHeader title="All users" description="Admin accounts with mobile login" />
      {items.length === 0 ? (
        <EmptyState
          title="No users yet"
          description="Create the first admin account to enable web login."
          action={<Button onClick={() => window.location.assign(usersPath('new'))}>Add user</Button>}
        />
      ) : (
        <DataTable columns={columns} data={items} />
      )}

      <Dialog open={dialog === 'edit'} onOpenChange={(o) => !o && setDialog(null)}>
        <DialogContent className="max-h-[90vh] max-w-lg overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Edit user</DialogTitle>
          </DialogHeader>
          {selected ? (
            <UserForm
              mode="edit"
              initial={selected}
              pending={update.isPending}
              onCancel={() => setDialog(null)}
              onSubmit={(values) => update.mutate({ id: selected.id, values })}
            />
          ) : null}
        </DialogContent>
      </Dialog>

      <Dialog open={dialog === 'delete'} onOpenChange={(o) => !o && setDialog(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete user</DialogTitle>
          </DialogHeader>
          <p className="text-sm">
            Delete user <strong>{selected?.phone}</strong>? This cannot be undone.
          </p>
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setDialog(null)}>
              Cancel
            </Button>
            <Button variant="destructive" disabled={remove.isPending} onClick={() => selected && remove.mutate(selected.id)}>
              Delete
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}

export function UsersCreatePage() {
  const navigate = useNavigate()

  const create = useMutationWithRevision({
    mutationFn: (values: Parameters<typeof usersApi.create>[0], etag) => usersApi.create(values, etag),
    invalidateKeys: [['users']],
    onSuccess: () => {
      toast.success('User created')
      navigate(usersPath('list'))
    },
  })

  return (
    <div>
      <PageHeader title="Add user" description="Create a new admin account" />
      <UserForm
        mode="create"
        pending={create.isPending}
        onCancel={() => navigate(usersPath('list'))}
        onSubmit={(values) => create.mutate(values as Parameters<typeof usersApi.create>[0])}
      />
    </div>
  )
}
