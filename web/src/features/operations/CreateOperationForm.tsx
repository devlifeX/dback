import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { toast } from 'sonner'
import { operationsApi } from '@/api/operations'
import { OPERATION_KINDS, type Host, type OperationKind } from '@/api/types'
import { Button } from '@/components/ui/button'
import { FormField } from '@/components/forms/FormField'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Input } from '@/components/ui/input'
import { Checkbox } from '@/components/ui/checkbox'

export function CreateOperationForm({
  hosts,
  defaultProfileId,
  onSuccess,
  onCancel,
}: {
  hosts: Host[]
  defaultProfileId?: string
  onSuccess?: () => void
  onCancel?: () => void
}) {
  const [kind, setKind] = useState<OperationKind>('backup_db')
  const [profileId, setProfileId] = useState(defaultProfileId ?? hosts[0]?.id ?? '')
  const [recordId, setRecordId] = useState('')
  const [destId, setDestId] = useState('')
  const [urlIndex, setUrlIndex] = useState('0')
  const [useProxy, setUseProxy] = useState(false)
  const [timeoutSec, setTimeoutSec] = useState('')

  const needsRecord = kind === 'restore' || kind === 'deep_verify'
  const isUrlChecker = kind === 'url_checker'
  const effectiveProfile = needsRecord ? destId : profileId

  const create = useMutation({
    mutationFn: () => {
      let params: Record<string, unknown> | undefined
      if (needsRecord) {
        params = { record_id: recordId, destination_profile_id: destId }
      } else if (isUrlChecker) {
        params = {
          url_index: Number(urlIndex),
          use_proxy: useProxy,
        }
        if (timeoutSec.trim()) {
          params.timeout_seconds = Number(timeoutSec)
        }
      }
      return operationsApi.create(kind, effectiveProfile, params)
    },
    onSuccess: () => {
      toast.success('Operation queued')
      onSuccess?.()
    },
    onError: (e: Error) => toast.error(e.message),
  })

  return (
    <div className="space-y-4">
      <FormField label="Operation kind">
        <Select value={kind} onValueChange={(v) => setKind(v as OperationKind)}>
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent>
            {OPERATION_KINDS.map((k) => (
              <SelectItem key={k.value} value={k.value}>{k.label}</SelectItem>
            ))}
          </SelectContent>
        </Select>
      </FormField>

      {needsRecord ? (
        <>
          <FormField label="Backup record ID" htmlFor="record-id">
            <Input id="record-id" value={recordId} onChange={(e) => setRecordId(e.target.value)} placeholder="From Backups page" />
          </FormField>
          <FormField label="Destination host">
            <Select value={destId} onValueChange={setDestId}>
              <SelectTrigger><SelectValue placeholder="Select host" /></SelectTrigger>
              <SelectContent>
                {hosts.map((h) => <SelectItem key={h.id} value={h.id}>{h.name}</SelectItem>)}
              </SelectContent>
            </Select>
          </FormField>
        </>
      ) : (
        <FormField label="Host">
          <Select value={profileId} onValueChange={setProfileId}>
            <SelectTrigger><SelectValue placeholder="Select host" /></SelectTrigger>
            <SelectContent>
              {hosts.map((h) => <SelectItem key={h.id} value={h.id}>{h.name}</SelectItem>)}
            </SelectContent>
          </Select>
        </FormField>
      )}

      {isUrlChecker ? (
        <>
          <FormField label="URL target">
            <Select value={urlIndex} onValueChange={setUrlIndex}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="0">Primary URL</SelectItem>
                <SelectItem value="-1">All URLs</SelectItem>
                {[1, 2, 3, 4, 5].map((n) => (
                  <SelectItem key={n} value={String(n)}>Secondary URL {n}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FormField>
          <div className="flex items-center gap-2">
            <Checkbox checked={useProxy} onCheckedChange={(c) => setUseProxy(!!c)} id="use-proxy" />
            <label htmlFor="use-proxy" className="text-sm">Use global Squid proxy</label>
          </div>
          <FormField label="Timeout (seconds, optional)" htmlFor="url-timeout">
            <Input id="url-timeout" type="number" min={0} value={timeoutSec} onChange={(e) => setTimeoutSec(e.target.value)} placeholder="30" />
          </FormField>
        </>
      ) : null}

      <div className="flex justify-end gap-2">
        {onCancel ? <Button variant="outline" onClick={onCancel}>Cancel</Button> : null}
        <Button
          disabled={create.isPending || !effectiveProfile || (needsRecord && !recordId)}
          onClick={() => create.mutate()}
        >
          Start operation
        </Button>
      </div>
    </div>
  )
}
