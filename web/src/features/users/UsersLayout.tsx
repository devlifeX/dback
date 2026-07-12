import { NavLink, Outlet, useLocation } from 'react-router-dom'
import { PageHeader } from '@/components/shared/page'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { USERS_TABS, usersPath, usersSegment } from './users-nav'

export function UsersLayout() {
  const { pathname } = useLocation()
  const active = usersSegment(pathname)

  return (
    <div>
      <PageHeader title="User Management" description="Admin accounts, login security, and SMS OTP" />
      <Tabs value={active} className="mt-2">
        <TabsList className="mb-2 h-auto flex-wrap justify-start gap-1">
          {USERS_TABS.map(({ segment, label }) => (
            <TabsTrigger key={segment} value={segment} asChild>
              <NavLink to={usersPath(segment)}>{label}</NavLink>
            </TabsTrigger>
          ))}
        </TabsList>
      </Tabs>
      <Outlet />
    </div>
  )
}
