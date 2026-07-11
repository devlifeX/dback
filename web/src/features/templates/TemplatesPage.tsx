import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { toast } from 'sonner'
import type { ColumnDef } from '@tanstack/react-table'
import { templatesApi } from '@/api/templates'
import type { SQLTemplate } from '@/api/types'
import { useMutationWithRevision } from '@/hooks/use-vault-mutation'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { DataTable } from '@/components/shared/data-table'
import { EmptyState, ErrorAlert, PageHeader } from '@/components/shared/page'
import { Skeleton } from '@/components/ui/badge'
import { TemplateForm } from './TemplateForm'

export function TemplatesPage() {
  const [dialog, setDialog] = useState<'create' | 'edit' | 'delete' | null>(null)
  const [selected, setSelected] = useState<SQLTemplate | null>(null)

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['templates'],
    queryFn: templatesApi.list,
  })

  const save = useMutationWithRevision({
    mutationFn: (tpl: SQLTemplate, etag) => templatesApi.save(tpl, etag),
    invalidateKeys: [['templates']],
    onSuccess: () => {
      toast.success('Template saved')
      setDialog(null)
      setSelected(null)
    },
  })

  const remove = useMutationWithRevision({
    mutationFn: (id: string, etag) => templatesApi.remove(id, etag),
    invalidateKeys: [['templates']],
    onSuccess: () => {
      toast.success('Template deleted')
      setDialog(null)
      setSelected(null)
    },
  })

  const items = data?.items ?? []

  const columns: ColumnDef<SQLTemplate>[] = [
    { accessorKey: 'name', header: 'Name' },
    {
      accessorKey: 'body',
      header: 'Preview',
      cell: ({ row }) => {
        const body = row.original.body
        return body.slice(0, 80) + (body.length > 80 ? '…' : '')
      },
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
  if (isError) return <ErrorAlert message="Could not load templates" onRetry={() => void refetch()} />

  return (
    <div>
      <PageHeader
        title="Templates"
        description="SQL templates for backup queries"
        actions={
          <Button onClick={() => { setSelected(null); setDialog('create') }}>New template</Button>
        }
      />
      {items.length === 0 ? (
        <EmptyState
          title="No templates"
          description="Create reusable SQL snippets with placeholder support."
          action={<Button onClick={() => setDialog('create')}>Create template</Button>}
        />
      ) : (
        <DataTable columns={columns} data={items} />
      )}

      <Dialog open={dialog === 'create' || dialog === 'edit'} onOpenChange={(o) => !o && setDialog(null)}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>{dialog === 'edit' ? 'Edit template' : 'New template'}</DialogTitle>
          </DialogHeader>
          <TemplateForm
            template={selected ?? undefined}
            pending={save.isPending}
            onCancel={() => setDialog(null)}
            onSubmit={(values) =>
              save.mutate({ id: selected?.id ?? '', ...values, body: values.body, name: values.name })
            }
          />
        </DialogContent>
      </Dialog>

      <Dialog open={dialog === 'delete'} onOpenChange={(o) => !o && setDialog(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete template</DialogTitle>
          </DialogHeader>
          <p className="text-sm">Delete &quot;{selected?.name}&quot;? This cannot be undone.</p>
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setDialog(null)}>Cancel</Button>
            <Button
              variant="destructive"
              disabled={remove.isPending}
              onClick={() => selected && remove.mutate(selected.id)}
            >
              Delete
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
