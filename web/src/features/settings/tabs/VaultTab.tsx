import { useRef, useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { toast } from 'sonner'
import { vaultApi } from '@/api/vault'
import { useMutationWithRevision } from '@/hooks/use-vault-mutation'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Checkbox } from '@/components/ui/checkbox'
import { FormField } from '@/components/forms/FormField'
import { ApiClientError } from '@/api/client'

function importErrorMessage(error: Error) {
  if (!(error instanceof ApiClientError)) return error.message
  if (error.message.includes('encrypted bundle requires')) {
    return 'This file is encrypted. Enter the same export password you used in the desktop app (Settings → Export JSON).'
  }
  return error.message
}

export function VaultTab() {
  const [exportPass, setExportPass] = useState('')
  const [exportSecrets, setExportSecrets] = useState(false)
  const [importPass, setImportPass] = useState('')
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const fileRef = useRef<HTMLInputElement>(null)

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
    mutationFn: async (file: File, etag) => {
      const passphrase = importPass.trim()
      // Same rule as the desktop app: encrypted bundles need the export password.
      const includeSecrets = passphrase !== ''
      if (!includeSecrets) {
        throw new Error('Enter the export password from the desktop app (Settings → Export JSON).')
      }
      const opts = { file, passphrase, includeSecrets }
      try {
        const preview = await vaultApi.importFilePreview(opts)
        const profileConflicts = preview.profile_conflicts ?? []
        const templateConflicts = preview.template_conflicts ?? []
        if (profileConflicts.length > 0 || templateConflicts.length > 0) {
          toast.message(`Import merge: ${profileConflicts.length} host conflicts, ${templateConflicts.length} template conflicts`)
        }
        return await vaultApi.importFileApply(opts, etag)
      } catch (e) {
        throw new Error(importErrorMessage(e instanceof Error ? e : new Error(String(e))))
      }
    },
    invalidateKeys: [['hosts'], ['templates'], ['backups'], ['tasks'], ['notifications'], ['destinations']],
    onSuccess: () => {
      setSelectedFile(null)
      if (fileRef.current) fileRef.current.value = ''
      toast.success('Import applied')
    },
  })

  return (
    <div className="space-y-6">
      <div>
        <h3 className="mb-3 text-sm font-medium">Export</h3>
        <div className="grid max-w-2xl gap-4 sm:grid-cols-2">
          <FormField label="Export password (optional)">
            <Input type="password" value={exportPass} onChange={(e) => setExportPass(e.target.value)} autoComplete="off" />
          </FormField>
          <label className="flex items-center gap-2 text-sm pt-6">
            <Checkbox checked={exportSecrets} onCheckedChange={setExportSecrets} />
            Encrypt export (passwords &amp; keys)
          </label>
        </div>
        <Button size="sm" variant="outline" className="mt-3" disabled={exportVault.isPending} onClick={() => exportVault.mutate()}>Export app data</Button>
      </div>

      <div>
        <h3 className="mb-3 text-sm font-medium">Import</h3>
        <p className="mb-3 max-w-2xl text-sm text-[hsl(var(--muted-foreground))]">
          Desktop exports are always encrypted. Use the same <strong>export password</strong> you entered in the desktop app under
          {' '}Settings → Export JSON → Export App Data.
        </p>
        <div className="grid max-w-2xl gap-4 sm:grid-cols-2">
          <FormField label="Export password">
            <Input type="password" value={importPass} onChange={(e) => setImportPass(e.target.value)} autoComplete="off" placeholder="Same password as desktop export" />
          </FormField>
        </div>
        <input
          ref={fileRef}
          type="file"
          accept=".json,application/json"
          className="hidden"
          onChange={(e) => {
            const file = e.target.files?.[0] ?? null
            setSelectedFile(file)
          }}
        />
        <div className="mt-3 flex flex-wrap items-center gap-2">
          <Button size="sm" variant="outline" onClick={() => fileRef.current?.click()}>Choose file</Button>
          {selectedFile ? <span className="text-sm text-[hsl(var(--muted-foreground))]">{selectedFile.name}</span> : null}
          <Button
            size="sm"
            disabled={importVault.isPending || !selectedFile || !importPass.trim()}
            onClick={() => selectedFile && importVault.mutate(selectedFile)}
          >
            Import app data
          </Button>
        </div>
      </div>
    </div>
  )
}
