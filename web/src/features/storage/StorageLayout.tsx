import { NavLink, Outlet, useLocation } from 'react-router-dom'
import { PageHeader } from '@/components/shared/page'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { STORAGE_TABS, storagePath, storageSegment } from './storage-nav'

export function StorageLayout() {
  const { pathname } = useLocation()
  const active = storageSegment(pathname)

  return (
    <div>
      <PageHeader
        title="Storage"
        description="Browse backup files on disk and in remote destinations"
      />
      <Tabs value={active} className="mt-2">
        <TabsList className="mb-2 h-auto flex-wrap justify-start gap-1">
          {STORAGE_TABS.map(({ segment, label }) => (
            <TabsTrigger key={segment} value={segment} asChild>
              <NavLink to={storagePath(segment)}>{label}</NavLink>
            </TabsTrigger>
          ))}
        </TabsList>
      </Tabs>
      <Outlet />
    </div>
  )
}
