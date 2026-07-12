import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export type CalendarPref = 'jalali' | 'gregorian'

type UIState = {
  sidebarCollapsed: boolean
  theme: 'dark' | 'light'
  calendar: CalendarPref
  setSidebarCollapsed: (v: boolean) => void
  toggleSidebar: () => void
  setTheme: (t: 'dark' | 'light') => void
  setCalendar: (c: CalendarPref) => void
}

export const useUIStore = create<UIState>()(
  persist(
    (set) => ({
      sidebarCollapsed: false,
      theme: 'dark',
      calendar: 'jalali',
      setSidebarCollapsed: (sidebarCollapsed) => set({ sidebarCollapsed }),
      toggleSidebar: () => set((s) => ({ sidebarCollapsed: !s.sidebarCollapsed })),
      setTheme: (theme) => set({ theme }),
      setCalendar: (calendar) => set({ calendar }),
    }),
    {
      name: 'dback-ui',
      partialize: (s) => ({ sidebarCollapsed: s.sidebarCollapsed, theme: s.theme, calendar: s.calendar }),
    },
  ),
)
