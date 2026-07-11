import { useRef, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { toast } from 'sonner'
import { destinationsApi, syncApi } from '@/api/settings'
import { systemApi } from '@/api/system'
import { vaultApi } from '@/api/vault'
import { useMutationWithRevision } from '@/hooks/use-vault-mutation'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Checkbox } from '@/components/ui/checkbox'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { FormField } from '@/components/forms/FormField'
import { SecretField } from '@/components/forms/SecretField'
import { DataTable } from '@/components/shared/data-table'
import { ErrorAlert, PageHeader } from '@/components/shared/page'
import { Skeleton } from '@/components/ui/badge'
import { DestinationForm } from './DestinationForm'
import { LogsPage } from './LogsPage'
import type { ColumnDef } from '@tanstack/react-table'
import type { RemoteDestination, SyncSettings } from '@/api/types'
import { formatDate } from '@/lib/utils'

export function SettingsPage() {
  const [destDialog, setDestDialog] = useState<'create' | 'edit' | 'delete' | null>(null)
  const [selectedDest, setSelectedDest] = useState<RemoteDestination | null>(null)
  const [exportPass, setExportPass] = useState('')
  const [exportSecrets, setExportSecrets] = useState(false)
  const [importPass, setImportPass] = useState('')
  const [importSecrets, setImportSecrets] = useState(false)
  const fileRef = useRef<HTMLInputElement>(null)

  const version = useQuery({ queryKey: ['version'], queryFn: systemApi.version })
  const revision = useQuery({ queryKey: ['revision'], queryFn: systemApi.revision })
  const destinations = useQuery({ queryKey: ['destinations'], queryFn: destinationsApi.list })
  const syncSettings = useQuery({ queryKey: ['sync-settings'], queryFn: syncApi.getSettings })
  const audit = useQuery({ queryKey: ['audit'], queryFn: () => systemApi.audit({ limit: 50 }) })

  const [syncForm, setSyncForm] = useState<SyncSettings | null>(null)
  const sync = syncForm ?? (syncSettings.data as SyncSettings | undefined) ?? {
    endpoint: '', bucket: '', access_key_id: '', secret_key: '', use_ssl: true,
  }

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

  const saveSync = useMutationWithRevision({
    mutationFn: (s: SyncSettings, etag) => syncApi.saveSettings(s, etag),
    invalidateKeys: [['sync-settings']],
    onSuccess: () => toast.success('Sync settings saved'),
  })

  const push = useMutation({ mutationFn: syncApi.push, onSuccess: () => toast.success('Sync push completed'), onError: (e: Error) => toast.error(e.message) })
  const pull = useMutation({ mutationFn: syncApi.pull, onSuccess: (d) => toast.success(`Pulled ${d.bytes} bytes`), onError: (e: Error) => toast.error(e.message) })

  const exportVault = useMutation({
    mutationFn: () => vaultApi.exportAppData({ passphrase: exportPass, includeSecrets: exportSecrets }),
    onSuccess: (blob) => {
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = 'dback-app-data.json'
      a.click()
      URL.revokeObjectURL(url)
      toast.success('Export downloaded')
    },
    onError: (e: Error) => toast.error(e.message),
  })

  const importVault = useMutationWithRevision({
    mutationFn: async (content: string, etag) => {
      const preview = await vaultApi.importPreview({ passphrase: importPass, include_secrets: importSecrets, content_base64: content })
      if ((preview.profile_conflicts as unknown[]).length > 0) {
        toast.message('Import has profile conflicts — applying merge')
      }
      return vaultApi.importApply({ passphrase: importPass, include_secrets: importSecrets, content_base64: content }, etag)
    },
    invalidateKeys: [['hosts'], ['templates'], ['backups'], ['tasks'], ['notifications'], ['destinations']],
    onSuccess: () => toast.success('Import applied'),
  })

  const destItems = destinations.data?.items ?? []
  const auditItems = audit.data?.items ?? []

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
    <div>
      <PageHeader title="Settings" description="Destinations, sync, vault, and system info" />
      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader><CardTitle>System</CardTitle></CardHeader>
          <CardContent className="space-y-2 text-sm">
            {version.isLoading ? <Skeleton className="h-4 w-32" /> : <p>Version: {version.data?.version}</p>}
            {revision.isLoading ? <Skeleton className="h-4 w-32" /> : <p>Vault revision: {revision.data?.revision}</p>}
          </CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle>Sync</CardTitle></CardHeader>
          <CardContent className="flex gap-2">
            <Button size="sm" onClick={() => push.mutate()} disabled={push.isPending}>Push</Button>
            <Button size="sm" variant="outline" onClick={() => pull.mutate()} disabled={pull.isPending}>Pull</Button>
          </CardContent>
        </Card>
      </div>

      <div className="mt-6 space-y-4">
        <h2 className="text-lg font-medium">Remote destinations</h2>
        <Button size="sm" onClick={() => { setSelectedDest(null); setDestDialog('create') }}>Add destination</Button>
        {destinations.isLoading ? <Skeleton className="h-24 w-full" /> : destinations.isError ? (
          <ErrorAlert message="Could not load destinations" />
        ) : (
          <DataTable columns={destColumns} data={destItems} emptyMessage="No destinations" />
        )}
      </div>

      <div className="mt-8 space-y-4">
        <h2 className="text-lg font-medium">Sync settings</h2>
        <div className="grid max-w-2xl gap-4 sm:grid-cols-2">
          <FormField label="Endpoint"><Input value={sync.endpoint} onChange={(e) => setSyncForm({ ...sync, endpoint: e.target.value })} /></FormField>
          <FormField label="Bucket"><Input value={sync.bucket} onChange={(e) => setSyncForm({ ...sync, bucket: e.target.value })} /></FormField>
          <FormField label="Access key"><Input value={sync.access_key_id} onChange={(e) => setSyncForm({ ...sync, access_key_id: e.target.value })} /></FormField>
          <SecretField label="Secret key" value={sync.secret_key ?? ''} onChange={(v) => setSyncForm({ ...sync, secret_key: v })} />
          <label className="flex items-center gap-2 text-sm sm:col-span-2">
            <Checkbox checked={sync.use_ssl} onCheckedChange={(c) => setSyncForm({ ...sync, use_ssl: c })} />
            Use SSL
          </label>
        </div>
        <Button size="sm" disabled={saveSync.isPending} onClick={() => saveSync.mutate(sync)}>Save sync settings</Button>
      </div>

      <div className="mt-8 space-y-4">
        <h2 className="text-lg font-medium">Vault export / import</h2>
        <div className="grid max-w-2xl gap-4 sm:grid-cols-2">
          <FormField label="Passphrase (optional)"><Input type="password" value={exportPass} onChange={(e) => setExportPass(e.target.value)} /></FormField>
          <label className="flex items-center gap-2 text-sm pt-6">
            <Checkbox checked={exportSecrets} onCheckedChange={setExportSecrets} />
            Include secrets
          </label>
        </div>
        <Button size="sm" variant="outline" disabled={exportVault.isPending} onClick={() => exportVault.mutate()}>Export app data</Button>

        <div className="grid max-w-2xl gap-4 sm:grid-cols-2 pt-4">
          <FormField label="Import passphrase"><Input type="password" value={importPass} onChange={(e) => setImportPass(e.target.value)} /></FormField>
          <label className="flex items-center gap-2 text-sm pt-6">
            <Checkbox checked={importSecrets} onCheckedChange={setImportSecrets} />
            Include secrets
          </label>
        </div>
        <input ref={fileRef} type="file" accept=".json" className="hidden" onChange={async (e) => {
          const file = e.target.files?.[0]
          if (!file) return
          const text = await file.text()
          importVault.mutate(btoa(unescape(encodeURIComponent(text))))
        }} />
        <Button size="sm" variant="outline" disabled={importVault.isPending} onClick={() => fileRef.current?.click()}>Import app data</Button>
      </div>

      <div className="mt-8">
        <h2 className="mb-3 text-lg font-medium">Audit log</h2>
        {audit.isLoading ? <Skeleton className="h-24" /> : (
          <div className="max-h-64 overflow-auto rounded border border-[hsl(var(--border))] text-xs font-mono">
            {auditItems.length === 0 ? <p className="p-4 text-[hsl(var(--muted-foreground))]">No audit entries</p> : null}
            {auditItems.map((entry, i) => (
              <div key={i} className="border-b border-[hsl(var(--border))] px-3 py-1">
                {formatDate(entry.timestamp)} {entry.method} {entry.path} → {entry.status}
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="mt-8">
        <LogsPage />
      </div>

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

export function AboutPage() {
  const version = useQuery({ queryKey: ['version'], queryFn: systemApi.version })
  return (
    <div>
      <PageHeader title="About" description="DBack Control Plane Web UI" />
      <p className="text-sm text-[hsl(var(--muted-foreground))]">
        Primary management interface for server deployments. API version {version.data?.api ?? 'v1'}.
      </p>
    </div>
  )
}
