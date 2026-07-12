import { Controller, useForm } from 'react-hook-form'
import { useQuery } from '@tanstack/react-query'
import { toast } from 'sonner'
import { usersApi } from '@/api/users'
import type { AuthSettings, AuthSMSConfig, SMSProvider } from '@/api/types'
import { useMutationWithRevision } from '@/hooks/use-vault-mutation'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Checkbox } from '@/components/ui/checkbox'
import { FormField } from '@/components/forms/FormField'
import { FormSection } from '@/components/forms/FormSection'
import { SecretField } from '@/components/forms/SecretField'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { ErrorAlert, PageHeader } from '@/components/shared/page'
import { Skeleton } from '@/components/ui/badge'

const SMS_PROVIDERS: { value: SMSProvider; label: string }[] = [
  { value: 'kavenegar', label: 'Kavenegar' },
  { value: 'melipayamak', label: 'MeliPayamak' },
]

function defaultSettings(): AuthSettings {
  return {
    two_factor_enabled: false,
    sms_provider: undefined,
    sms_config: {},
    otp_ttl_seconds: 300,
    otp_length: 6,
  }
}

export function UserSettingsPage() {
  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['users', 'settings'],
    queryFn: usersApi.getSettings,
  })

  const save = useMutationWithRevision({
    mutationFn: (settings: AuthSettings, etag) => usersApi.saveSettings(settings, etag),
    invalidateKeys: [['users', 'settings']],
    onSuccess: () => toast.success('User settings saved'),
  })

  const { register, handleSubmit, control, watch, setValue } = useForm<AuthSettings>({
    values: data ?? defaultSettings(),
  })

  const twoFactor = watch('two_factor_enabled')
  const provider = watch('sms_provider')
  const smsConfig = watch('sms_config') ?? {}

  if (isLoading) return <Skeleton className="h-40 w-full" />
  if (isError) return <ErrorAlert message="Could not load user settings" onRetry={() => void refetch()} />

  return (
    <div>
      <PageHeader title="User settings" description="Global two-factor authentication and SMS OTP provider" />
      <form
        onSubmit={handleSubmit((values) => save.mutate(values))}
        className="max-w-xl space-y-4"
      >
        <FormSection title="Two-factor authentication">
          <div className="flex items-center gap-2 sm:col-span-2">
            <Controller
              control={control}
              name="two_factor_enabled"
              render={({ field }) => <Checkbox checked={!!field.value} onCheckedChange={field.onChange} />}
            />
            <span className="text-sm">Require OTP via SMS on every login</span>
          </div>
        </FormSection>

        {twoFactor ? (
          <FormSection title="SMS provider" description="Used to send login OTP codes">
            <FormField label="Provider">
              <Controller
                control={control}
                name="sms_provider"
                render={({ field }) => (
                  <Select
                    value={field.value ?? ''}
                    onValueChange={(v) => {
                      field.onChange(v)
                      if (v === 'kavenegar') setValue('sms_config', { api_key: '', line: '' })
                      if (v === 'melipayamak') setValue('sms_config', { username: '', password: '', from: '' })
                    }}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="Select provider" />
                    </SelectTrigger>
                    <SelectContent>
                      {SMS_PROVIDERS.map((p) => (
                        <SelectItem key={p.value} value={p.value}>
                          {p.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                )}
              />
            </FormField>

            {provider === 'kavenegar' ? (
              <>
                <SecretField
                  label="API key"
                  value={(smsConfig as AuthSMSConfig).api_key ?? ''}
                  onChange={(v) => setValue('sms_config', { ...smsConfig, api_key: v })}
                />
                <FormField label="Line number" htmlFor="sms-line">
                  <Input
                    id="sms-line"
                    value={(smsConfig as AuthSMSConfig).line ?? ''}
                    onChange={(e) => setValue('sms_config', { ...smsConfig, line: e.target.value })}
                  />
                </FormField>
              </>
            ) : null}

            {provider === 'melipayamak' ? (
              <>
                <FormField label="Username" htmlFor="sms-username">
                  <Input
                    id="sms-username"
                    value={(smsConfig as AuthSMSConfig).username ?? ''}
                    onChange={(e) => setValue('sms_config', { ...smsConfig, username: e.target.value })}
                  />
                </FormField>
                <SecretField
                  label="Password"
                  value={(smsConfig as AuthSMSConfig).password ?? ''}
                  onChange={(v) => setValue('sms_config', { ...smsConfig, password: v })}
                />
                <FormField label="Sender line" htmlFor="sms-from">
                  <Input
                    id="sms-from"
                    value={(smsConfig as AuthSMSConfig).from ?? ''}
                    onChange={(e) => setValue('sms_config', { ...smsConfig, from: e.target.value })}
                  />
                </FormField>
              </>
            ) : null}

            <FormField label="OTP TTL (seconds)" htmlFor="otp-ttl">
              <Input id="otp-ttl" type="number" min={60} {...register('otp_ttl_seconds', { valueAsNumber: true })} />
            </FormField>
            <FormField label="OTP length" htmlFor="otp-length">
              <Input id="otp-length" type="number" min={4} max={8} {...register('otp_length', { valueAsNumber: true })} />
            </FormField>
          </FormSection>
        ) : null}

        <div className="flex justify-end">
          <Button type="submit" disabled={save.isPending}>
            Save settings
          </Button>
        </div>
      </form>
    </div>
  )
}
