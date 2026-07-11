import { useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { toast } from 'sonner'
import { syncApi } from '@/api/settings'
import { useMutationWithRevision } from '@/hooks/use-vault-mutation'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Checkbox } from '@/components/ui/checkbox'
import { FormField } from '@/components/forms/FormField'
import { SecretField } from '@/components/forms/SecretField'
import type { SyncSettings } from '@/api/types'

export function SyncTab() {
  const syncSettings = useQuery({ queryKey: ['sync-settings'], queryFn: syncApi.getSettings })

  const [syncForm, setSyncForm] = useState<SyncSettings | null>(null)
  const sync = syncForm ?? (syncSettings.data as SyncSettings | undefined) ?? {
    endpoint: '', bucket: '', access_key_id: '', secret_key: '', use_ssl: true,
  }

  const saveSync = useMutationWithRevision({
    mutationFn: (s: SyncSettings, etag) => syncApi.saveSettings(s, etag),
    invalidateKeys: [['sync-settings']],
    onSuccess: () => toast.success('Sync settings saved'),
  })

  const push = useMutation({ mutationFn: syncApi.push, onSuccess: () => toast.success('Sync push completed'), onError: (e: Error) => toast.error(e.message) })
  const pull = useMutation({ mutationFn: syncApi.pull, onSuccess: (d) => toast.success(`Pulled ${d.bytes} bytes`), onError: (e: Error) => toast.error(e.message) })

  return (
    <div className="space-y-6">
      <Card className="max-w-2xl">
        <CardHeader><CardTitle>Sync actions</CardTitle></CardHeader>
        <CardContent className="flex gap-2">
          <Button size="sm" onClick={() => push.mutate()} disabled={push.isPending}>Push</Button>
          <Button size="sm" variant="outline" onClick={() => pull.mutate()} disabled={pull.isPending}>Pull</Button>
        </CardContent>
      </Card>

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
  )
}
