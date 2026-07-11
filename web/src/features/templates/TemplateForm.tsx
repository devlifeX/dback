import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { Button } from '@/components/ui/button'
import { Input, Textarea } from '@/components/ui/input'
import { FormField } from '@/components/forms/FormField'
import type { SQLTemplate } from '@/api/types'

const schema = z.object({
  name: z.string().min(1, 'Name is required'),
  body: z.string().min(1, 'SQL body is required'),
  description: z.string().optional(),
})

type FormValues = z.infer<typeof schema>

export function TemplateForm({
  template,
  onSubmit,
  onCancel,
  pending,
}: {
  template?: SQLTemplate
  onSubmit: (values: FormValues) => void
  onCancel: () => void
  pending?: boolean
}) {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: template?.name ?? '',
      body: template?.body ?? '',
      description: template?.description ?? '',
    },
  })

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <FormField label="Name" htmlFor="tpl-name" error={errors.name?.message}>
        <Input id="tpl-name" {...register('name')} />
      </FormField>
      <FormField label="Description" htmlFor="tpl-desc">
        <Input id="tpl-desc" {...register('description')} />
      </FormField>
      <FormField label="SQL body" htmlFor="tpl-body" error={errors.body?.message}>
        <Textarea id="tpl-body" rows={10} className="font-mono text-xs" {...register('body')} />
      </FormField>
      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit" disabled={pending}>
          {template?.id ? 'Save changes' : 'Create template'}
        </Button>
      </div>
    </form>
  )
}
