import { useRef, useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { toast } from 'sonner'
import { vaultApi } from '@/api/vault'
import { useMutationWithRevision } from '@/hooks/use-vault-mutation'
import { Button } from '@/components/ui/button'
import { ApiClientError } from '@/api/client'

function importErrorMessage(error: Error) {
  if (!(error instanceof ApiClientError)) return error.message
  if (error.message.includes('encrypted bundle requires')) {
    return 'This file is encrypted. Use the desktop app export password, or export a new plaintext bundle from this panel.'
  }
  return error.message
}

export function VaultTab() {
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const fileRef = useRef<HTMLInputElement>(null)

  const exportVault = useMutation({
    mutationFn: () => vaultApi.exportAppData({ includeSecrets: true }),
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
      const opts = { file, includeSecrets: true }
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
    invalidateKeys: [['hosts'], ['templates'], ['backups'], ['tasks'], ['notifications'], ['destinations'], ['squid-proxies'], ['squid-settings']],
    onSuccess: () => {
      setSelectedFile(null)
      if (fileRef.current) fileRef.current.value = ''
      toast.success('Import applied')
    },
  })

  return (
    <div className="space-y-6">
      <p className="max-w-2xl text-sm text-[hsl(var(--muted-foreground))]">
        Export or import all panel data (hosts, tasks, notifications, users, auth, Squid, destinations, settings) as plaintext JSON for easy migration between instances.
      </p>

      <div>
        <h3 className="mb-3 text-sm font-medium">Export</h3>
        <Button size="sm" variant="outline" disabled={exportVault.isPending} onClick={() => exportVault.mutate()}>
          Export app data
        </Button>
      </div>

      <div>
        <h3 className="mb-3 text-sm font-medium">Import</h3>
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
        <div className="flex flex-wrap items-center gap-2">
          <Button size="sm" variant="outline" onClick={() => fileRef.current?.click()}>Choose file</Button>
          {selectedFile ? <span className="text-sm text-[hsl(var(--muted-foreground))]">{selectedFile.name}</span> : null}
          <Button
            size="sm"
            disabled={importVault.isPending || !selectedFile}
            onClick={() => selectedFile && importVault.mutate(selectedFile)}
          >
            Import app data
          </Button>
        </div>
      </div>
    </div>
  )
}
