import { Controller, useForm } from 'react-hook-form'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Checkbox } from '@/components/ui/checkbox'
import { FormField } from '@/components/forms/FormField'
import { FormSection } from '@/components/forms/FormSection'
import { SecretField } from '@/components/forms/SecretField'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { DEFAULT_NOTIFY_EVENTS, NOTIFY_EVENTS, type BaleConfig, type NotifyChannel, type NotifyProvider, type SlackConfig, type TelegramConfig, type WebhookConfig } from '@/api/types'

const PROVIDERS: { value: NotifyProvider; label: string }[] = [
  { value: 'telegram', label: 'Telegram' },
  { value: 'slack', label: 'Slack' },
  { value: 'bale', label: 'Bale' },
  { value: 'webhook', label: 'Webhook' },
]

type FormValues = NotifyChannel

export function NotificationForm({
  channel,
  onSubmit,
  onCancel,
  pending,
}: {
  channel?: NotifyChannel
  onSubmit: (values: FormValues) => void
  onCancel: () => void
  pending?: boolean
}) {
  const { register, handleSubmit, control, watch, setValue } = useForm<FormValues>({
    defaultValues: channel ?? {
      id: '',
      name: '',
      provider: 'telegram',
      enabled: true,
      events: [...DEFAULT_NOTIFY_EVENTS],
      config: { token: '', chat_id: '' } as TelegramConfig,
    },
  })

  const provider = watch('provider')
  const events = watch('events') ?? []

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <FormSection title="Channel">
        <FormField label="Name" htmlFor="ch-name">
          <Input id="ch-name" {...register('name', { required: true })} />
        </FormField>
        <FormField label="Provider">
          <Controller
            control={control}
            name="provider"
            render={({ field }) => (
              <Select value={field.value} onValueChange={(v) => {
                field.onChange(v)
                if (v === 'telegram') setValue('config', { token: '', chat_id: '' })
                if (v === 'slack') setValue('config', { webhook_url: '' })
                if (v === 'bale') setValue('config', { token: '', chat_id: '' })
                if (v === 'webhook') setValue('config', { url: '' })
              }}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {PROVIDERS.map((p) => <SelectItem key={p.value} value={p.value}>{p.label}</SelectItem>)}
                </SelectContent>
              </Select>
            )}
          />
        </FormField>
        <div className="flex items-center gap-2">
          <Controller control={control} name="enabled" render={({ field }) => (
            <Checkbox checked={field.value} onCheckedChange={field.onChange} />
          )} />
          <span className="text-sm">Enabled</span>
        </div>
      </FormSection>

      <div>
        <p className="mb-2 text-sm font-medium">Events</p>
        <p className="mb-2 text-xs text-[hsl(var(--muted-foreground))]">
          Successful backups send <code className="text-xs">operation.completed</code>. Leave all checked unless you want to filter.
        </p>
        <div className="grid gap-2 sm:grid-cols-2">
          {NOTIFY_EVENTS.map((ev) => (
            <label key={ev} className="flex items-center gap-2 text-sm">
              <Checkbox
                checked={events.includes(ev)}
                onCheckedChange={(c) => {
                  const next = c ? [...events, ev] : events.filter((x) => x !== ev)
                  setValue('events', next)
                }}
              />
              {ev}
            </label>
          ))}
        </div>
      </div>

      <FormSection title="Provider config">
        {provider === 'telegram' ? (
          <>
            <SecretField label="Bot token" value={(watch('config') as TelegramConfig)?.token ?? ''} onChange={(v) => setValue('config', { ...(watch('config') as TelegramConfig), token: v })} />
            <FormField label="Chat ID" htmlFor="tg-chat">
              <Input id="tg-chat" value={(watch('config') as TelegramConfig)?.chat_id ?? ''} onChange={(e) => setValue('config', { ...(watch('config') as TelegramConfig), chat_id: e.target.value })} />
            </FormField>
          </>
        ) : null}
        {provider === 'slack' ? (
          <div className="sm:col-span-2">
            <SecretField label="Webhook URL" value={(watch('config') as SlackConfig)?.webhook_url ?? ''} onChange={(v) => setValue('config', { webhook_url: v })} />
          </div>
        ) : null}
        {provider === 'bale' ? (
          <>
            <SecretField label="Token" value={(watch('config') as BaleConfig)?.token ?? ''} onChange={(v) => setValue('config', { ...(watch('config') as BaleConfig), token: v })} />
            <FormField label="Chat ID" htmlFor="bale-chat">
              <Input id="bale-chat" value={(watch('config') as BaleConfig)?.chat_id ?? ''} onChange={(e) => setValue('config', { ...(watch('config') as BaleConfig), chat_id: e.target.value })} />
            </FormField>
          </>
        ) : null}
        {provider === 'webhook' ? (
          <>
            <FormField label="URL" htmlFor="wh-url" className="sm:col-span-2">
              <Input id="wh-url" value={(watch('config') as WebhookConfig)?.url ?? ''} onChange={(e) => setValue('config', { ...(watch('config') as WebhookConfig), url: e.target.value })} />
            </FormField>
            <FormField label="Method" htmlFor="wh-method">
              <Input id="wh-method" placeholder="POST" value={(watch('config') as WebhookConfig)?.method ?? ''} onChange={(e) => setValue('config', { ...(watch('config') as WebhookConfig), method: e.target.value })} />
            </FormField>
          </>
        ) : null}
      </FormSection>

      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={onCancel}>Cancel</Button>
        <Button type="submit" disabled={pending}>{channel?.id ? 'Save' : 'Create'}</Button>
      </div>
    </form>
  )
}
