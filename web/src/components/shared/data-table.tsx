import {
  flexRender,
  getCoreRowModel,
  getSortedRowModel,
  useReactTable,
  type ColumnDef,
  type SortingState,
} from '@tanstack/react-table'
import { useState } from 'react'
import { cn } from '@/lib/utils'

type DataTableProps<T> = {
  columns: ColumnDef<T, unknown>[]
  data: T[]
  emptyMessage?: string
  className?: string
}

export function DataTable<T>({ columns, data, emptyMessage = 'No rows', className }: DataTableProps<T>) {
  const [sorting, setSorting] = useState<SortingState>([])
  const table = useReactTable({
    data,
    columns,
    state: { sorting },
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
  })

  if (data.length === 0) {
    return (
      <div className="rounded-lg border border-dashed border-[hsl(var(--border))] px-6 py-12 text-center text-sm text-[hsl(var(--muted-foreground))]">
        {emptyMessage}
      </div>
    )
  }

  return (
    <div className={cn('overflow-x-auto rounded-lg border border-[hsl(var(--border))]', className)}>
      <table className="w-full min-w-[640px] text-left text-sm">
        <thead className="bg-[hsl(var(--muted))]">
          {table.getHeaderGroups().map((hg) => (
            <tr key={hg.id}>
              {hg.headers.map((header) => (
                <th key={header.id} className="px-4 py-2 font-medium">
                  {header.isPlaceholder ? null : (
                    <button
                      type="button"
                      className={cn(header.column.getCanSort() && 'cursor-pointer select-none hover:underline')}
                      onClick={header.column.getToggleSortingHandler()}
                    >
                      {flexRender(header.column.columnDef.header, header.getContext())}
                      {{ asc: ' ↑', desc: ' ↓' }[header.column.getIsSorted() as string] ?? null}
                    </button>
                  )}
                </th>
              ))}
            </tr>
          ))}
        </thead>
        <tbody>
          {table.getRowModel().rows.map((row) => (
            <tr key={row.id} className="border-t border-[hsl(var(--border))] hover:bg-[hsl(var(--muted)/0.5)]">
              {row.getVisibleCells().map((cell) => (
                <td key={cell.id} className="px-4 py-2 align-middle">
                  {flexRender(cell.column.columnDef.cell, cell.getContext())}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
