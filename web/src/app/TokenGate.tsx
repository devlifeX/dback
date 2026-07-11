import { useEffect, useState, type FormEvent, type ReactNode } from 'react'
import { clearApiToken, hasApiToken, setApiToken } from '@/api/client'
import { systemApi } from '@/api/system'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

export function TokenGate({ children }: { children: ReactNode }) {
  const [ready, setReady] = useState(hasApiToken())
  const [token, setToken] = useState('')
  const [error, setError] = useState('')

  async function submit(e: FormEvent) {
    e.preventDefault()
    setError('')
    setApiToken(token)
    try {
      await systemApi.revision()
      setReady(true)
    } catch {
      clearApiToken()
      setError('Invalid token or API unavailable')
    }
  }

  useEffect(() => {
    document.documentElement.classList.toggle('light', false)
  }, [])

  if (ready) return <>{children}</>

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>DBack Control Plane</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={submit} className="space-y-4">
            <p className="text-sm text-[hsl(var(--muted-foreground))]">
              Enter your API bearer token. It is kept in memory only for this session.
            </p>
            <Input
              type="password"
              placeholder="DBACK_API_TOKEN"
              value={token}
              onChange={(e) => setToken(e.target.value)}
              autoComplete="off"
              required
            />
            {error ? <p className="text-sm text-[hsl(var(--destructive))]">{error}</p> : null}
            <Button type="submit" className="w-full">
              Connect
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
