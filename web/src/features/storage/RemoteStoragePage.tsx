import { useEffect, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { toast } from 'sonner'
import { destinationsApi } from '@/api/settings'
import { storageApi } from '@/api/storage'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { ErrorAlert, EmptyState } from '@/components/shared/page'
import { downloadBlob, StorageExplorer, StorageHint } from './StorageExplorer'

export function RemoteStoragePage() {
  const [destinationId, setDestinationId] = useState('')
  const [prefix, setPrefix] = useState('')
  const [downloadingPath, setDownloadingPath] = useState('')

  const destinations = useQuery({ queryKey: ['destinations'], queryFn: destinationsApi.list })

  useEffect(() => {
    const items = destinations.data?.items ?? []
    if (!destinationId && items.length > 0) {
      setDestinationId(items[0].id)
    }
  }, [destinations.data, destinationId])

  const listing = useQuery({
    queryKey: ['storage', 'remote', destinationId, prefix],
    queryFn: () => storageApi.listRemote(destinationId, prefix || undefined),
    enabled: Boolean(destinationId),
  })

  const download = useMutation({
    mutationFn: async (key: string) => {
      setDownloadingPath(key)
      return storageApi.downloadRemote(destinationId, key)
    },
    onSuccess: (blob, key) => {
      const name = key.split('/').pop() ?? 'download'
      downloadBlob(blob, name)
      toast.success('Download started')
    },
    onError: (e: Error) => toast.error(e.message),
    onSettled: () => setDownloadingPath(''),
  })

  const destItems = destinations.data?.items ?? []

  if (destinations.isError) {
    return <ErrorAlert message="Could not load destinations" onRetry={() => void destinations.refetch()} />
  }

  if (!destinations.isLoading && destItems.length === 0) {
    return (
      <EmptyState
        title="No remote destinations"
        description="Add an S3 destination in Settings to browse remote backups."
        action={<Button asChild><Link to="/settings/destinations">Destinations</Link></Button>}
      />
    )
  }

  if (listing.isError) {
    return <ErrorAlert message="Could not load remote storage" onRetry={() => void listing.refetch()} />
  }

  const data = listing.data

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-3">
        <Select value={destinationId} onValueChange={(id) => { setDestinationId(id); setPrefix('') }}>
          <SelectTrigger className="w-64"><SelectValue placeholder="Select destination" /></SelectTrigger>
          <SelectContent>
            {destItems.map((d) => (
              <SelectItem key={d.id} value={d.id}>{d.name}</SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <StorageHint>
        Objects under <code className="text-xs">dback/backups/</code> on the selected destination.
      </StorageHint>
      <StorageExplorer
        path={prefix || data?.path || 'dback/backups/'}
        parent={data?.parent}
        entries={data?.entries ?? []}
        loading={listing.isLoading || destinations.isLoading}
        emptyMessage="No backup objects in this folder"
        onNavigate={setPrefix}
        onOpen={(entry) => {
          if (entry.is_dir) setPrefix(entry.path)
        }}
        onDownload={(entry) => download.mutate(entry.path)}
        downloadingPath={downloadingPath}
        breadcrumbLabel={(segment) => segment}
      />
    </div>
  )
}
