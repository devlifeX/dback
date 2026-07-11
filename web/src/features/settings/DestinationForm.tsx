import { Controller, useForm } from 'react-hook-form'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Checkbox } from '@/components/ui/checkbox'
import { FormField } from '@/components/forms/FormField'
import { FormSection } from '@/components/forms/FormSection'
import { SecretField } from '@/components/forms/SecretField'
import type { RemoteDestination } from '@/api/types'

export function DestinationForm({
  destination,
  onSubmit,
  onCancel,
  pending,
}: {
  destination?: RemoteDestination
  onSubmit: (values: RemoteDestination) => void
  onCancel: () => void
  pending?: boolean
}) {
  const { register, handleSubmit, control, watch, setValue } = useForm<RemoteDestination>({
    defaultValues: destination ?? {
      id: '',
      name: '',
      type: 's3',
      s3: { endpoint: '', bucket: '', access_key_id: '', secret_key: '', use_ssl: true },
    },
  })

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <FormSection title="Destination">
        <FormField label="Name" htmlFor="dest-name">
          <Input id="dest-name" {...register('name', { required: true })} />
        </FormField>
        <div className="flex items-center gap-2">
          <Controller control={control} name="s3.use_ssl" render={({ field }) => (
            <Checkbox checked={!!field.value} onCheckedChange={field.onChange} />
          )} />
          <span className="text-sm">Use SSL</span>
        </div>
        <FormField label="Endpoint" htmlFor="s3-endpoint">
          <Input id="s3-endpoint" {...register('s3.endpoint')} />
        </FormField>
        <FormField label="Region" htmlFor="s3-region">
          <Input id="s3-region" {...register('s3.region')} />
        </FormField>
        <FormField label="Bucket" htmlFor="s3-bucket">
          <Input id="s3-bucket" {...register('s3.bucket')} />
        </FormField>
        <FormField label="Access key ID" htmlFor="s3-key">
          <Input id="s3-key" {...register('s3.access_key_id')} />
        </FormField>
        <SecretField
          label="Secret key"
          value={watch('s3.secret_key') ?? ''}
          onChange={(v) => setValue('s3.secret_key', v)}
        />
      </FormSection>
      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={onCancel}>Cancel</Button>
        <Button type="submit" disabled={pending}>{destination?.id ? 'Save' : 'Create'}</Button>
      </div>
    </form>
  )
}
