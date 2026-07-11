import { useEffect } from 'react'
import { useQueryClient } from '@tanstack/react-query'

import { getApiToken } from '@/api/client'

export function useOperationsSSE(enabled: boolean) {
  const qc = useQueryClient()

  useEffect(() => {
    if (!enabled) return
    const token = getApiToken()
    const url = token
      ? `/api/v1/operations/stream?access_token=${encodeURIComponent(token)}`
      : '/api/v1/operations/stream'
    const es = new EventSource(url)
    const invalidate = () => void qc.invalidateQueries({ queryKey: ['operations'] })

    es.addEventListener('operation.started', invalidate)
    es.addEventListener('operation.progress', invalidate)
    es.addEventListener('operation.completed', invalidate)
    es.addEventListener('operation.failed', invalidate)
    es.addEventListener('operation.canceled', invalidate)
    es.addEventListener('task.skipped', invalidate)

    return () => es.close()
  }, [enabled, qc])
}
