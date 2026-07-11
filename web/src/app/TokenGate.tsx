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
    } catch (err) {
      clearApiToken()
      if (err instanceof TypeError) {
        setError('API unavailable — start the server with ./run-web.sh (dev: http://127.0.0.1:5173)')
        return
      }
      if (err && typeof err === 'object' && 'status' in err) {
        const status = (err as { status: number }).status
        if (status === 401) {
          setError('Invalid token — default dev token is dev-token (see terminal from ./run-web.sh)')
          return
        }
        if (status === 503) {
          setError('API token not configured on server — set DBACK_API_TOKEN when running dback serve')
          return
        }
      }
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
              {import.meta.env.DEV ? (
                <>
                  {' '}
                  Dev default: <code className="text-xs">dev-token</code> (from{' '}
                  <code className="text-xs">./run-web.sh</code>).
                </>
              ) : null}
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
