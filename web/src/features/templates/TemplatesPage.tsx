import { useQuery } from '@tanstack/react-query'
import { templatesApi } from '@/api/templates'
import { DataTable, EmptyState, ErrorAlert, PageHeader } from '@/components/shared/page'
import { Skeleton } from '@/components/ui/badge'

export function TemplatesPage() {
  const { data, isLoading, isError, refetch } = useQuery({ queryKey: ['templates'], queryFn: templatesApi.list })

  if (isLoading) return <Skeleton className="h-40 w-full" />
  if (isError) return <ErrorAlert message="Could not load templates" onRetry={() => void refetch()} />

  const items = data?.items ?? []
  return (
    <div>
      <PageHeader title="Templates" description="SQL templates for backup queries" />
      {items.length === 0 ? (
        <EmptyState title="No templates" description="Templates are managed in the vault." />
      ) : (
        <DataTable
          headers={['Name', 'Preview']}
          rows={items.map((t) => [t.name, t.body.slice(0, 80) + (t.body.length > 80 ? '…' : '')])}
        />
      )}
    </div>
  )
}
