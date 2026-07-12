import { useEffect, type ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { hasSession } from '@/api/client'

export function TokenGate({ children }: { children: ReactNode }) {
  const location = useLocation()

  useEffect(() => {
    document.documentElement.classList.toggle('light', false)
  }, [])

  if (!hasSession()) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />
  }

  return <>{children}</>
}
