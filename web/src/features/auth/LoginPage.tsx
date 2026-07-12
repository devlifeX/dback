import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { authApi } from '@/api/auth'
import { setSessionToken } from '@/api/client'
import { ApiClientError } from '@/api/client'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

export function LoginPage() {
  const navigate = useNavigate()
  const [phone, setPhone] = useState('')
  const [password, setPassword] = useState('')
  const [otp, setOtp] = useState('')
  const [challengeId, setChallengeId] = useState('')
  const [step, setStep] = useState<'login' | 'otp'>('login')
  const [error, setError] = useState('')
  const [pending, setPending] = useState(false)

  async function completeLogin(token: string) {
    setSessionToken(token)
    navigate('/dashboard', { replace: true })
  }

  async function submitLogin(e: FormEvent) {
    e.preventDefault()
    setError('')
    setPending(true)
    try {
      const res = await authApi.login(phone, password)
      if (res.otp_required && res.challenge_id) {
        setChallengeId(res.challenge_id)
        setStep('otp')
        return
      }
      if (!res.token) {
        setError('Login succeeded but no session token was returned')
        return
      }
      await completeLogin(res.token)
    } catch (err) {
      if (err instanceof ApiClientError) {
        if (err.status === 401) {
          setError('Invalid phone or password')
        } else if (err.status === 403) {
          setError('This account is disabled')
        } else if (err.status === 503) {
          setError('Two-factor is enabled but SMS is not configured')
        } else {
          setError(err.message)
        }
      } else if (err instanceof TypeError) {
        setError('API unavailable — start the server with ./run-web.sh')
      } else {
        setError('Login failed')
      }
    } finally {
      setPending(false)
    }
  }

  async function submitOtp(e: FormEvent) {
    e.preventDefault()
    setError('')
    setPending(true)
    try {
      const res = await authApi.verifyOtp(challengeId, otp)
      if (!res.token) {
        setError('Verification succeeded but no session token was returned')
        return
      }
      await completeLogin(res.token)
    } catch (err) {
      if (err instanceof ApiClientError && err.status === 401) {
        setError('Invalid or expired verification code')
      } else {
        setError(err instanceof Error ? err.message : 'Verification failed')
      }
    } finally {
      setPending(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>DBack Control Plane</CardTitle>
        </CardHeader>
        <CardContent>
          {step === 'login' ? (
            <form onSubmit={submitLogin} className="space-y-4">
              <p className="text-sm text-[hsl(var(--muted-foreground))]">
                Sign in with your mobile number and password.
              </p>
              <Input
                type="tel"
                placeholder="09XXXXXXXXX"
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
                autoComplete="username"
                required
              />
              <Input
                type="password"
                placeholder="Password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                autoComplete="current-password"
                required
              />
              {error ? <p className="text-sm text-[hsl(var(--destructive))]">{error}</p> : null}
              <Button type="submit" className="w-full" disabled={pending}>
                {pending ? 'Signing in…' : 'Sign in'}
              </Button>
            </form>
          ) : (
            <form onSubmit={submitOtp} className="space-y-4">
              <p className="text-sm text-[hsl(var(--muted-foreground))]">
                Enter the verification code sent to your phone.
              </p>
              <Input
                type="text"
                inputMode="numeric"
                placeholder="OTP code"
                value={otp}
                onChange={(e) => setOtp(e.target.value)}
                autoComplete="one-time-code"
                required
              />
              {error ? <p className="text-sm text-[hsl(var(--destructive))]">{error}</p> : null}
              <div className="flex gap-2">
                <Button type="button" variant="outline" className="flex-1" onClick={() => { setStep('login'); setOtp(''); setError('') }}>
                  Back
                </Button>
                <Button type="submit" className="flex-1" disabled={pending}>
                  {pending ? 'Verifying…' : 'Verify'}
                </Button>
              </div>
            </form>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
