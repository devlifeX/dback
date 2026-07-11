import { NavLink, Outlet } from 'react-router-dom'
import {
  Bell,
  Database,
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
import { Button } from '@/components/ui/button'
import { Sheet } from '@/components/ui/sheet'
import { useUIStore } from '@/store/ui-store'
import { useState } from 'react'
import { cn } from '@/lib/utils'
import { clearApiToken } from '@/api/client'
import { useNavigate } from 'react-router-dom'

const nav = [
  { to: '/dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { to: '/hosts', label: 'Hosts', icon: Server },
  { to: '/backups', label: 'Backups', icon: HardDrive },
  { to: '/operations', label: 'Operations', icon: PlayCircle },
  { to: '/tasks', label: 'Tasks', icon: Workflow },
  { to: '/notifications', label: 'Notifications', icon: Bell },
  { to: '/templates', label: 'Templates', icon: Database },
  { to: '/settings', label: 'Settings', icon: Settings },
]

function NavItems({ onNavigate }: { onNavigate?: () => void }) {
  return (
    <nav className="flex flex-col gap-1 p-2">
      {nav.map(({ to, label, icon: Icon }) => (
        <NavLink
          key={to}
          to={to}
          onClick={onNavigate}
          className={({ isActive }) =>
            cn(
              'flex items-center gap-2 rounded-md px-3 py-2 text-sm transition-colors',
              isActive ? 'bg-[hsl(var(--primary)/0.15)] text-[hsl(var(--primary))]' : 'hover:bg-[hsl(var(--muted))]',
            )
          }
        >
          <Icon className="h-4 w-4" />
          {label}
        </NavLink>
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
