import type { SortingState } from '@tanstack/react-table'

/** Default table sort: newest / largest values first. */
export function newestFirst(columnId: string): { sorting: SortingState } {
  return { sorting: [{ id: columnId, desc: true }] }
}
