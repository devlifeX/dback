import { useEffect } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Checkbox } from '@/components/ui/checkbox'
import { FormField } from '@/components/forms/FormField'
import { FormSection } from '@/components/forms/FormSection'
import { countryLabel } from '@/lib/country-flag'
import type { SquidProxy } from '@/api/types'

const emptyProxy: SquidProxy = {
  id: '',
  name: '',
  url: 'http://',
  country: '',
  country_code: '',
  enabled: true,
}

export function SquidProxyForm({
  proxy,
  onSubmit,
  onCancel,
  pending,
}: {
  proxy?: SquidProxy
  onSubmit: (values: SquidProxy) => void
  onCancel: () => void
  pending?: boolean
}) {
  const { register, handleSubmit, control, watch, reset } = useForm<SquidProxy>({
    defaultValues: proxy ?? emptyProxy,
  })

  useEffect(() => {
    reset(proxy ?? emptyProxy)
  }, [proxy, reset])

  const country = watch('country')
  const countryCode = watch('country_code')

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <FormSection title="Squid proxy" description="HTTP proxy used for regional URL checks">
        <FormField label="Name" htmlFor="proxy-name">
          <Input id="proxy-name" {...register('name', { required: true })} placeholder="EU Squid" />
        </FormField>
        <FormField label="Proxy URL" htmlFor="proxy-url" className="sm:col-span-2">
          <Input id="proxy-url" {...register('url', { required: true })} placeholder="http://proxy.example:3128" />
        </FormField>
        <FormField label="Country" htmlFor="proxy-country">
          <Input id="proxy-country" {...register('country', { required: true })} placeholder="Germany" />
        </FormField>
        <FormField label="Country code (ISO 2)" htmlFor="proxy-country-code">
          <Input id="proxy-country-code" {...register('country_code')} placeholder="DE" maxLength={2} />
        </FormField>
        {(country || countryCode) ? (
          <p className="sm:col-span-2 text-sm text-[hsl(var(--muted-foreground))]">
            Preview: {countryLabel(country || '—', countryCode)}
          </p>
        ) : null}
        <div className="flex items-center gap-2 sm:col-span-2">
          <Controller
            control={control}
            name="enabled"
            render={({ field }) => (
              <Checkbox checked={!!field.value} onCheckedChange={field.onChange} />
            )}
          />
          <span className="text-sm">Enabled</span>
        </div>
      </FormSection>
      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={onCancel}>Cancel</Button>
        <Button type="submit" disabled={pending}>{proxy?.id ? 'Save proxy' : 'Add proxy'}</Button>
      </div>
    </form>
  )
}
