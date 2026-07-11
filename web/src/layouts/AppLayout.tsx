import { NavLink, Outlet, useLocation } from 'react-router-dom'
import {
  Bell,
  Database,
  FolderOpen,
  HardDrive,
  LayoutDashboard,
  Menu,
  Moon,
  PlayCircle,
  Server,
  Settings,
  Sun,
  Workflow,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Sheet } from '@/components/ui/sheet'
import { useUIStore } from '@/store/ui-store'
import { useState } from 'react'
import { cn } from '@/lib/utils'
import { clearApiToken } from '@/api/client'
import { useNavigate } from 'react-router-dom'
import { SETTINGS_TABS, settingsPath } from '@/features/settings/settings-nav'
import { STORAGE_TABS, storagePath } from '@/features/storage/storage-nav'

type NavItem = {
  to: string
  label: string
  icon: LucideIcon
  children?: { to: string; label: string }[]
}

const nav: NavItem[] = [
  { to: '/dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { to: '/hosts', label: 'Hosts', icon: Server },
  { to: '/backups', label: 'Backups', icon: HardDrive },
  {
    to: storagePath('local'),
    label: 'Storage',
    icon: FolderOpen,
    children: STORAGE_TABS.map(({ segment, label }) => ({ to: storagePath(segment), label })),
  },
  { to: '/operations', label: 'Operations', icon: PlayCircle },
  { to: '/tasks', label: 'Tasks', icon: Workflow },
  { to: '/notifications', label: 'Notifications', icon: Bell },
  { to: '/templates', label: 'Templates', icon: Database },
  {
    to: settingsPath('general'),
    label: 'Settings',
    icon: Settings,
    children: SETTINGS_TABS.map(({ segment, label }) => ({ to: settingsPath(segment), label })),
  },
]

function NavItems({ onNavigate }: { onNavigate?: () => void }) {
  const { pathname } = useLocation()
  const inSettings = pathname.startsWith('/settings')
  const inStorage = pathname.startsWith('/storage')

  return (
    <nav className="flex flex-col gap-1 p-2">
      {nav.map(({ to, label, icon: Icon, children }) => (
        <div key={to}>
          <NavLink
            to={to}
            end={!children}
            onClick={onNavigate}
            className={({ isActive }) =>
              cn(
                'flex items-center gap-2 rounded-md px-3 py-2 text-sm transition-colors',
                (children ? (to.startsWith('/settings') ? inSettings : inStorage) : isActive)
                  ? 'bg-[hsl(var(--primary)/0.15)] text-[hsl(var(--primary))]'
                  : 'hover:bg-[hsl(var(--muted))]',
              )
            }
          >
            <Icon className="h-4 w-4 shrink-0" />
            {label}
          </NavLink>
          {children ? (
            <div className="ml-3 mt-0.5 flex flex-col gap-0.5 border-l border-[hsl(var(--border))] pl-2">
              {children.map((child) => (
                <NavLink
                  key={child.to}
                  to={child.to}
                  end
                  onClick={onNavigate}
                  className={({ isActive }) =>
                    cn(
                      'rounded-md px-3 py-1.5 text-sm transition-colors',
                      isActive
                        ? 'bg-[hsl(var(--primary)/0.15)] font-medium text-[hsl(var(--primary))]'
                        : 'text-[hsl(var(--muted-foreground))] hover:bg-[hsl(var(--muted))] hover:text-[hsl(var(--foreground))]',
                    )
                  }
                >
                  {child.label}
                </NavLink>
              ))}
            </div>
          ) : null}
        </div>
      ))}
    </nav>
  )
}

export function AppLayout() {
  const { sidebarCollapsed, toggleSidebar, theme, setTheme } = useUIStore()
  const [mobileOpen, setMobileOpen] = useState(false)
  const navigate = useNavigate()

  function logout() {
    clearApiToken()
    navigate(0)
  }

  return (
    <div className="flex min-h-screen">
      <aside
        className={cn(
          'hidden border-r border-[hsl(var(--border))] bg-[hsl(var(--card))] md:block',
          sidebarCollapsed ? 'w-16' : 'w-56',
        )}
      >
        <div className="flex h-14 items-center border-b border-[hsl(var(--border))] px-4 font-semibold">
          {sidebarCollapsed ? 'DB' : 'DBack'}
        </div>
        {!sidebarCollapsed ? <NavItems /> : null}
      </aside>

      <Sheet open={mobileOpen} onOpenChange={setMobileOpen} title="Menu">
        <NavItems onNavigate={() => setMobileOpen(false)} />
      </Sheet>

      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex h-14 items-center justify-between border-b border-[hsl(var(--border))] px-4">
          <div className="flex items-center gap-2">
            <Button variant="ghost" size="icon" className="md:hidden" onClick={() => setMobileOpen(true)}>
              <Menu className="h-4 w-4" />
            </Button>
            <Button variant="ghost" size="icon" className="hidden md:inline-flex" onClick={toggleSidebar}>
              <Menu className="h-4 w-4" />
            </Button>
            <span className="text-sm text-[hsl(var(--muted-foreground))]">Control Plane</span>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              size="icon"
              onClick={() => {
                const next = theme === 'dark' ? 'light' : 'dark'
                setTheme(next)
                document.documentElement.classList.toggle('light', next === 'light')
              }}
            >
              {theme === 'dark' ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
            </Button>
            <Button variant="outline" size="sm" onClick={logout}>
              Sign out
            </Button>
          </div>
        </header>
        <main className="flex-1 overflow-auto p-4 md:p-6">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
