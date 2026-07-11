import { useQuery } from '@tanstack/react-query'
import { systemApi } from '@/api/system'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/badge'

export function GeneralTab() {
  const version = useQuery({ queryKey: ['version'], queryFn: systemApi.version })
  const revision = useQuery({ queryKey: ['revision'], queryFn: systemApi.revision })

  return (
    <div className="grid gap-4 lg:grid-cols-2">
      <Card>
        <CardHeader><CardTitle>System</CardTitle></CardHeader>
        <CardContent className="space-y-2 text-sm">
          {version.isLoading ? <Skeleton className="h-4 w-32" /> : <p>Version: {version.data?.version}</p>}
          {revision.isLoading ? <Skeleton className="h-4 w-32" /> : <p>Vault revision: {revision.data?.revision}</p>}
        </CardContent>
      </Card>
    </div>
  )
}
