import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { toast } from 'sonner'
import type { ColumnDef } from '@tanstack/react-table'
import { destinationsApi } from '@/api/settings'
import { useMutationWithRevision } from '@/hooks/use-vault-mutation'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { DataTable } from '@/components/shared/data-table'
import { ErrorAlert } from '@/components/shared/page'
import { Skeleton } from '@/components/ui/badge'
import { DestinationForm } from '../DestinationForm'
import type { RemoteDestination } from '@/api/types'

export function DestinationsTab() {
  const [destDialog, setDestDialog] = useState<'create' | 'edit' | 'delete' | null>(null)
  const [selectedDest, setSelectedDest] = useState<RemoteDestination | null>(null)

  const destinations = useQuery({ queryKey: ['destinations'], queryFn: destinationsApi.list })

  const saveDest = useMutationWithRevision({
    mutationFn: (d: RemoteDestination, etag) => destinationsApi.save(d, etag),
    invalidateKeys: [['destinations']],
    onSuccess: () => { toast.success('Destination saved'); setDestDialog(null); setSelectedDest(null) },
  })

  const removeDest = useMutationWithRevision({
    mutationFn: (id: string, etag) => destinationsApi.remove(id, etag),
    invalidateKeys: [['destinations']],
    onSuccess: () => { toast.success('Destination deleted'); setDestDialog(null) },
  })

  const destItems = destinations.data?.items ?? []

  const destColumns: ColumnDef<RemoteDestination>[] = [
    { accessorKey: 'name', header: 'Name' },
    { accessorKey: 'type', header: 'Type' },
    {
      id: 'actions',
      header: '',
      cell: ({ row }) => (
        <div className="flex gap-2">
          <Button size="sm" variant="outline" onClick={() => { setSelectedDest(row.original); setDestDialog('edit') }}>Edit</Button>
          <Button size="sm" variant="destructive" onClick={() => { setSelectedDest(row.original); setDestDialog('delete') }}>Delete</Button>
        </div>
      ),
    },
  ]

  return (
    <div className="space-y-4">
      <Button size="sm" onClick={() => { setSelectedDest(null); setDestDialog('create') }}>Add destination</Button>
      {destinations.isLoading ? <Skeleton className="h-24 w-full" /> : destinations.isError ? (
        <ErrorAlert message="Could not load destinations" />
      ) : (
        <DataTable columns={destColumns} data={destItems} emptyMessage="No destinations" />
      )}

      <Dialog open={destDialog === 'create' || destDialog === 'edit'} onOpenChange={(o) => !o && setDestDialog(null)}>
        <DialogContent className="max-w-lg">
          <DialogHeader><DialogTitle>{destDialog === 'edit' ? 'Edit destination' : 'New destination'}</DialogTitle></DialogHeader>
          <DestinationForm destination={selectedDest ?? undefined} pending={saveDest.isPending} onCancel={() => setDestDialog(null)} onSubmit={(v) => saveDest.mutate(v)} />
        </DialogContent>
      </Dialog>

      <Dialog open={destDialog === 'delete'} onOpenChange={(o) => !o && setDestDialog(null)}>
        <DialogContent>
          <DialogHeader><DialogTitle>Delete destination</DialogTitle></DialogHeader>
          <p className="text-sm">Delete &quot;{selectedDest?.name}&quot;?</p>
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setDestDialog(null)}>Cancel</Button>
            <Button variant="destructive" disabled={removeDest.isPending} onClick={() => selectedDest && removeDest.mutate(selectedDest.id)}>Delete</Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
