import { useQuery } from '@tanstack/react-query'
import { systemApi } from '@/api/system'
import { PageHeader } from '@/components/shared/page'

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
