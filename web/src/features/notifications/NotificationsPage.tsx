import { useMutation, useQuery } from '@tanstack/react-query'
import { toast } from 'sonner'
import { notificationsApi } from '@/api/notifications'
import { Button } from '@/components/ui/button'
import { DataTable, EmptyState, ErrorAlert, PageHeader } from '@/components/shared/page'
import { Skeleton } from '@/components/ui/badge'

export function NotificationsPage() {
  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['notifications'],
    queryFn: notificationsApi.list,
  })

  const test = useMutation({
    mutationFn: (id: string) => notificationsApi.test(id),
    onSuccess: () => toast.success('Test notification sent'),
    onError: (e: Error) => toast.error(e.message),
  })

  if (isLoading) return <Skeleton className="h-40 w-full" />
  if (isError) return <ErrorAlert message="Could not load channels" onRetry={() => void refetch()} />

  const items = data?.items ?? []
  return (
    <div>
      <PageHeader title="Notifications" description="Alert channels for operation events" />
      {items.length === 0 ? (
        <EmptyState title="No channels" description="Configure Telegram, Slack, Bale, or Webhook via API." />
      ) : (
        <DataTable
          headers={['Name', 'Provider', 'Status', 'Events', '']}
          rows={items.map((c) => [
            c.name,
            c.provider,
            c.enabled ? 'Enabled' : 'Disabled',
            (c.events ?? ['all']).join(', '),
            <Button key={c.id} size="sm" variant="outline" disabled={test.isPending} onClick={() => test.mutate(c.id)}>
              Test
            </Button>,
          ])}
        />
      )}
    </div>
  )
}
