import { useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { toast } from 'sonner'
import type { ColumnDef } from '@tanstack/react-table'
import { squidApi } from '@/api/squid'
import type { SquidProxy, SquidSettings } from '@/api/types'
import { useMutationWithRevision } from '@/hooks/use-vault-mutation'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { DataTable } from '@/components/shared/data-table'
import { ErrorAlert } from '@/components/shared/page'
import { Skeleton } from '@/components/ui/badge'
import { FormField } from '@/components/forms/FormField'
import { FormSection } from '@/components/forms/FormSection'
import { countryLabel } from '@/lib/country-flag'
import { SquidProxyForm } from '../SquidProxyForm'

export function SquidTab() {
  const [proxyDialog, setProxyDialog] = useState<'create' | 'edit' | 'delete' | null>(null)
  const [selectedProxy, setSelectedProxy] = useState<SquidProxy | null>(null)

  const proxies = useQuery({ queryKey: ['squid-proxies'], queryFn: squidApi.listProxies })
  const settings = useQuery({ queryKey: ['squid-settings'], queryFn: squidApi.getSettings })

  const editProxy = useQuery({
    queryKey: ['squid-proxies', selectedProxy?.id],
    queryFn: () => squidApi.getProxy(selectedProxy!.id),
    enabled: proxyDialog === 'edit' && !!selectedProxy?.id,
  })

  const saveSettings = useMutationWithRevision({
    mutationFn: (s: SquidSettings, etag) => squidApi.saveSettings(s, etag),
    invalidateKeys: [['squid-settings']],
    onSuccess: () => toast.success('Primary host country saved'),
  })

  const saveProxy = useMutationWithRevision({
    mutationFn: (p: SquidProxy, etag) => squidApi.saveProxy(p, etag),
    invalidateKeys: [['squid-proxies']],
    onSuccess: () => {
      toast.success('Squid proxy saved')
      setProxyDialog(null)
      setSelectedProxy(null)
    },
  })

  const removeProxy = useMutationWithRevision({
    mutationFn: (id: string, etag) => squidApi.removeProxy(id, etag),
    invalidateKeys: [['squid-proxies']],
    onSuccess: () => {
      toast.success('Squid proxy deleted')
      setProxyDialog(null)
    },
  })

  const proxyItems = proxies.data?.items ?? []
  const squidSettings = settings.data

  const columns: ColumnDef<SquidProxy>[] = [
    { accessorKey: 'name', header: 'Name' },
    {
      id: 'country',
      header: 'Country',
      cell: ({ row }) => countryLabel(row.original.country, row.original.country_code),
    },
    { accessorKey: 'url', header: 'URL' },
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
              setSelectedProxy(row.original)
              setProxyDialog('edit')
            }}
          >
            Edit
          </Button>
          <Button
            size="sm"
            variant="destructive"
            onClick={() => {
              setSelectedProxy(row.original)
              setProxyDialog('delete')
            }}
          >
            Delete
          </Button>
        </div>
      ),
    },
  ]

  return (
    <div className="space-y-8">
      <FormSection
        title="Primary host country"
        description="Direct URL checks (without Squid) are labeled with this country on charts"
      >
        {settings.isLoading ? (
          <Skeleton className="h-24 w-full" />
        ) : settings.isError ? (
          <ErrorAlert message="Could not load Squid settings" />
        ) : (
          <PrimaryHostCountryForm
            initial={squidSettings ?? { primary_host_country: 'Local server' }}
            pending={saveSettings.isPending}
            onSave={(s) => saveSettings.mutate(s)}
          />
        )}
      </FormSection>

      <div className="space-y-4">
        <div className="flex items-center justify-between gap-2">
          <div>
            <h3 className="text-sm font-medium">Squid proxies</h3>
            <p className="text-xs text-[hsl(var(--muted-foreground))]">
              Add regional HTTP proxies; hosts can select which ones to use for url_checker.
            </p>
          </div>
          <Button size="sm" onClick={() => { setSelectedProxy(null); setProxyDialog('create') }}>
            Add proxy
          </Button>
        </div>
        {proxies.isLoading ? (
          <Skeleton className="h-24 w-full" />
        ) : proxies.isError ? (
          <ErrorAlert message="Could not load Squid proxies" />
        ) : (
          <DataTable columns={columns} data={proxyItems} emptyMessage="No Squid proxies configured" />
        )}
      </div>

      <Dialog open={proxyDialog === 'create' || proxyDialog === 'edit'} onOpenChange={(o) => !o && setProxyDialog(null)}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>{proxyDialog === 'edit' ? 'Edit Squid proxy' : 'New Squid proxy'}</DialogTitle>
          </DialogHeader>
          {proxyDialog === 'edit' && editProxy.isLoading ? (
            <Skeleton className="h-48 w-full" />
          ) : proxyDialog === 'edit' && editProxy.isError ? (
            <ErrorAlert message="Could not load proxy" />
          ) : (
            <SquidProxyForm
              proxy={proxyDialog === 'edit' ? editProxy.data : undefined}
              pending={saveProxy.isPending}
              onCancel={() => setProxyDialog(null)}
              onSubmit={(p) => saveProxy.mutate(p)}
            />
          )}
        </DialogContent>
      </Dialog>

      <Dialog open={proxyDialog === 'delete'} onOpenChange={(o) => !o && setProxyDialog(null)}>
        <DialogContent>
          <DialogHeader><DialogTitle>Delete Squid proxy?</DialogTitle></DialogHeader>
          <p className="text-sm text-[hsl(var(--muted-foreground))]">
            Remove {selectedProxy?.name}? Hosts referencing this proxy will fail validation until updated.
          </p>
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setProxyDialog(null)}>Cancel</Button>
            <Button
              variant="destructive"
              disabled={removeProxy.isPending || !selectedProxy?.id}
              onClick={() => selectedProxy?.id && removeProxy.mutate(selectedProxy.id)}
            >
              Delete
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}

function PrimaryHostCountryForm({
  initial,
  pending,
  onSave,
}: {
  initial: SquidSettings
  pending?: boolean
  onSave: (s: SquidSettings) => void
}) {
  const [country, setCountry] = useState(initial.primary_host_country)
  const [code, setCode] = useState(initial.primary_host_country_code ?? '')

  useEffect(() => {
    setCountry(initial.primary_host_country)
    setCode(initial.primary_host_country_code ?? '')
  }, [initial.primary_host_country, initial.primary_host_country_code])

  return (
    <div className="grid gap-4 sm:grid-cols-2 max-w-xl">
      <FormField label="Country name" htmlFor="primary-country">
        <Input
          id="primary-country"
          value={country}
          onChange={(e) => setCountry(e.target.value)}
          placeholder="Germany"
        />
      </FormField>
      <FormField label="Country code (ISO 2)" htmlFor="primary-country-code">
        <Input
          id="primary-country-code"
          value={code}
          onChange={(e) => setCode(e.target.value.toUpperCase())}
          placeholder="DE"
          maxLength={2}
        />
      </FormField>
      {(country || code) ? (
        <p className="sm:col-span-2 text-sm text-[hsl(var(--muted-foreground))]">
          Chart label: {countryLabel(country || '—', code)} (direct, no Squid)
        </p>
      ) : null}
      <div className="sm:col-span-2">
        <Button
          size="sm"
          disabled={pending || !country.trim()}
          onClick={() =>
            onSave({
              primary_host_country: country.trim(),
              primary_host_country_code: code.trim() || undefined,
            })
          }
        >
          Save primary host country
        </Button>
      </div>
    </div>
  )
}
