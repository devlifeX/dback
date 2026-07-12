import { Controller, useForm } from 'react-hook-form'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Checkbox } from '@/components/ui/checkbox'
import { FormField } from '@/components/forms/FormField'
import { FormSection } from '@/components/forms/FormSection'
import type { CreateUserInput, UpdateUserInput } from '@/api/users'

type FormValues = {
  phone: string
  password: string
  name: string
  enabled: boolean
}

export function UserForm({
  initial,
  onSubmit,
  onCancel,
  pending,
  mode,
}: {
  initial?: Partial<FormValues>
  onSubmit: (values: CreateUserInput | UpdateUserInput) => void
  onCancel: () => void
  pending?: boolean
  mode: 'create' | 'edit'
}) {
  const { register, handleSubmit, control } = useForm<FormValues>({
    defaultValues: {
      phone: initial?.phone ?? '',
      password: '',
      name: initial?.name ?? '',
      enabled: initial?.enabled ?? true,
    },
  })

  return (
    <form
      onSubmit={handleSubmit((values) => {
        if (mode === 'create') {
          onSubmit({ phone: values.phone, password: values.password, name: values.name })
          return
        }
        const update: UpdateUserInput = { name: values.name, enabled: values.enabled }
        if (values.phone) update.phone = values.phone
        if (values.password) update.password = values.password
        onSubmit(update)
      })}
      className="space-y-4"
    >
      <FormSection title={mode === 'create' ? 'New user' : 'Edit user'}>
        <FormField label="Phone" htmlFor="user-phone">
          <Input id="user-phone" placeholder="09XXXXXXXXX" {...register('phone', { required: mode === 'create' })} />
        </FormField>
        <FormField label="Name" htmlFor="user-name">
          <Input id="user-name" {...register('name')} />
        </FormField>
        <FormField label={mode === 'create' ? 'Password' : 'New password (optional)'} htmlFor="user-password">
          <Input
            id="user-password"
            type="password"
            autoComplete="new-password"
            {...register('password', { required: mode === 'create', minLength: 6 })}
          />
        </FormField>
        {mode === 'edit' ? (
          <div className="flex items-center gap-2 sm:col-span-2">
            <Controller
              control={control}
              name="enabled"
              render={({ field }) => <Checkbox checked={field.value} onCheckedChange={field.onChange} />}
            />
            <span className="text-sm">Account enabled</span>
          </div>
        ) : null}
      </FormSection>
      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit" disabled={pending}>
          {mode === 'create' ? 'Create user' : 'Save changes'}
        </Button>
      </div>
    </form>
  )
}
