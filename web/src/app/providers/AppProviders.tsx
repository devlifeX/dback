import { QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter } from 'react-router-dom'
import { Toaster } from 'sonner'
import { queryClient } from '@/app/query-client'
import { TokenGate } from '@/app/TokenGate'
import { AppRoutes } from '@/routes/AppRoutes'
import { useUIStore } from '@/store/ui-store'
import { useEffect } from 'react'

function ThemeSync() {
  const theme = useUIStore((s) => s.theme)
  useEffect(() => {
    document.documentElement.classList.toggle('light', theme === 'light')
  }, [theme])
  return null
}

export function AppProviders() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <ThemeSync />
        <TokenGate>
          <AppRoutes />
        </TokenGate>
        <Toaster richColors closeButton position="top-right" />
      </BrowserRouter>
    </QueryClientProvider>
  )
}
