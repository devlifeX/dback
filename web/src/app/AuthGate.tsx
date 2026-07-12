import { useEffect, useState } from 'react'
import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { authApi } from '@/api/auth'
import { clearSessionToken, hasSession, initAuth } from '@/api/client'
import { Skeleton } from '@/components/ui/badge'

export function AuthGate() {
  const location = useLocation()
  const [ready, setReady] = useState(false)
  const [checking, setChecking] = useState(true)

  useEffect(() => {
    initAuth()
    if (!hasSession()) {
      setChecking(false)
      return
    }
    authApi
      .me()
      .then(() => setReady(true))
      .catch(() => clearSessionToken())
      .finally(() => setChecking(false))
  }, [])

  if (checking) {
    return (
      <div className="flex min-h-screen items-center justify-center p-4">
        <Skeleton className="h-10 w-48" />
      </div>
    )
  }

  if (!hasSession() || !ready) {
    return <Navigate to="/login" replace state={{ from: location }} />
  }

  return <Outlet />
}
