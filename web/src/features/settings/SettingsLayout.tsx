import { NavLink, Outlet, useLocation } from 'react-router-dom'
import { PageHeader } from '@/components/shared/page'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { SETTINGS_TABS, settingsPath, settingsSegment } from './settings-nav'

export function SettingsLayout() {
  const { pathname } = useLocation()
  const active = settingsSegment(pathname)

  return (
    <div>
      <PageHeader title="Settings" description="Destinations, sync, vault, and system info" />
      <Tabs value={active} className="mt-2">
        <TabsList className="mb-2 h-auto flex-wrap justify-start gap-1">
          {SETTINGS_TABS.map(({ segment, label }) => (
            <TabsTrigger key={segment} value={segment} asChild>
              <NavLink to={settingsPath(segment)}>{label}</NavLink>
            </TabsTrigger>
          ))}
        </TabsList>
      </Tabs>
      <Outlet />
    </div>
  )
}
