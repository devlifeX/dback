import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import type { ColumnDef } from '@tanstack/react-table'
import { systemApi } from '@/api/system'
import { DataTable } from '@/components/shared/data-table'
import { ErrorAlert } from '@/components/shared/page'
import { Skeleton } from '@/components/ui/badge'
import { useFormatDate } from '@/lib/datetime'
import type { LogEntry } from '@/api/types'

export function LogsTab() {
  const formatDate = useFormatDate()
  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['logs'],
    queryFn: () => systemApi.logs({ limit: 200 }),
  })

  const items = data?.items ?? []

  const columns: ColumnDef<LogEntry>[] = useMemo(() => [
    { accessorKey: 'timestamp', header: 'Time', cell: ({ row }) => formatDate(row.original.timestamp) },
    { accessorKey: 'profile_name', header: 'Host', cell: ({ row }) => row.original.profile_name ?? row.original.profile_id ?? '—' },
    { accessorKey: 'action', header: 'Action' },
    { accessorKey: 'status', header: 'Status' },
    { accessorKey: 'details', header: 'Details', cell: ({ row }) => row.original.details?.slice(0, 120) },
  ], [formatDate])

  if (isLoading) return <Skeleton className="h-40 w-full" />
  if (isError) return <ErrorAlert message="Could not load logs" onRetry={() => void refetch()} />

  return <DataTable columns={columns} data={items} emptyMessage="No log entries" />
}
