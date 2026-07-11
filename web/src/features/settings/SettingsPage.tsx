import { useMutation, useQuery } from '@tanstack/react-query'
import { toast } from 'sonner'
import { destinationsApi, syncApi } from '@/api/settings'
import { systemApi } from '@/api/system'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { DataTable, ErrorAlert, PageHeader } from '@/components/shared/page'
import { Skeleton } from '@/components/ui/badge'

export function SettingsPage() {
  const version = useQuery({ queryKey: ['version'], queryFn: systemApi.version })
  const revision = useQuery({ queryKey: ['revision'], queryFn: systemApi.revision })
  const destinations = useQuery({ queryKey: ['destinations'], queryFn: destinationsApi.list })

  const push = useMutation({
    mutationFn: syncApi.push,
    onSuccess: () => toast.success('Sync push completed'),
    onError: (e: Error) => toast.error(e.message),
  })
  const pull = useMutation({
    mutationFn: syncApi.pull,
    onSuccess: (d) => toast.success(`Pulled ${d.bytes} bytes`),
    onError: (e: Error) => toast.error(e.message),
  })

  return (
    <div>
      <PageHeader title="Settings" description="Destinations, sync, and system info" />
      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>System</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2 text-sm">
            {version.isLoading ? <Skeleton className="h-4 w-32" /> : <p>Version: {version.data?.version}</p>}
            {revision.isLoading ? <Skeleton className="h-4 w-32" /> : <p>Vault revision: {revision.data?.revision}</p>}
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Sync</CardTitle>
          </CardHeader>
          <CardContent className="flex gap-2">
            <Button size="sm" onClick={() => push.mutate()} disabled={push.isPending}>
              Push
            </Button>
            <Button size="sm" variant="outline" onClick={() => pull.mutate()} disabled={pull.isPending}>
              Pull
            </Button>
          </CardContent>
        </Card>
      </div>
      <div className="mt-6">
        <h2 className="mb-3 text-lg font-medium">Remote destinations</h2>
        {destinations.isLoading ? (
          <Skeleton className="h-24 w-full" />
        ) : destinations.isError ? (
          <ErrorAlert message="Could not load destinations" />
        ) : (
          <DataTable
            headers={['Name', 'Type']}
            rows={(destinations.data?.items ?? []).map((d) => [d.name, d.type])}
          />
        )}
      </div>
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
