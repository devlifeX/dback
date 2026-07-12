import { apiRequest } from './client'
import type { LoginResponse, User } from './types'

export const authApi = {
  login: (phone: string, password: string) =>
    apiRequest<LoginResponse>('/api/v1/auth/login', {
      method: 'POST',
      body: { phone, password },
      skipAuthRedirect: true,
    }),

  verifyOtp: (challengeId: string, code: string) =>
    apiRequest<LoginResponse>('/api/v1/auth/verify-otp', {
      method: 'POST',
      body: { challenge_id: challengeId, code },
      skipAuthRedirect: true,
    }),

  logout: () => apiRequest<void>('/api/v1/auth/logout', { method: 'POST' }),

  me: () => apiRequest<User>('/api/v1/auth/me'),
}
