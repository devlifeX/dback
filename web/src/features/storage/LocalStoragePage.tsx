import { useMemo, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { toast } from 'sonner'
import { hostsApi } from '@/api/hosts'
import { storageApi } from '@/api/storage'
import { ErrorAlert } from '@/components/shared/page'
import { downloadBlob, StorageExplorer, StorageHint } from './StorageExplorer'

export function LocalStoragePage() {
  const [path, setPath] = useState('')
  const [downloadingPath, setDownloadingPath] = useState('')

  const hosts = useQuery({ queryKey: ['hosts'], queryFn: hostsApi.list })

  const profileNames = useMemo(() => {
    const map: Record<string, string> = {}
    for (const host of hosts.data?.items ?? []) {
      map[host.id] = host.name
    }
    return map
  }, [hosts.data?.items])

  const listing = useQuery({
    queryKey: ['storage', 'local', path],
    queryFn: () => storageApi.listLocal(path || undefined),
  })

  const download = useMutation({
    mutationFn: async (filePath: string) => {
      setDownloadingPath(filePath)
      return storageApi.downloadLocal(filePath)
    },
    onSuccess: (blob, filePath) => {
      const name = filePath.split(/[/\\]/).pop() ?? 'download'
      downloadBlob(blob, name)
      toast.success('Download started')
    },
    onError: (e: Error) => toast.error(e.message),
    onSettled: () => setDownloadingPath(''),
  })

  if (listing.isError) {
    return <ErrorAlert message="Could not load local storage" onRetry={() => void listing.refetch()} />
  }

  const data = listing.data
  const entries = data?.entries ?? []

  return (
    <div className="space-y-3">
      <StorageHint>
        Backup folders configured on your hosts. Open a folder to browse files and download backups.
      </StorageHint>
      <StorageExplorer
        path={path}
        parent={data?.parent}
        entries={entries}
        loading={listing.isLoading}
        emptyMessage={path ? 'This folder is empty' : 'No backup destinations configured'}
        onNavigate={setPath}
        onOpen={(entry) => {
          if (entry.is_dir) setPath(entry.path)
        }}
        onDownload={(entry) => download.mutate(entry.path)}
        downloadingPath={downloadingPath}
        breadcrumbLabel={(segment) => {
          if (!path && data?.roots) {
            const root = data.roots.find((r) => r.path === segment || r.path.endsWith(segment))
            if (root?.label) return root.label
          }
          return segment
        }}
        profileNames={profileNames}
      />
    </div>
  )
}
