import { Controller, useForm } from 'react-hook-form'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Checkbox } from '@/components/ui/checkbox'
import { FormField } from '@/components/forms/FormField'
import { FormSection } from '@/components/forms/FormSection'
import type { Host, NotifyChannel, Task } from '@/api/types'
import { TriggerEditor } from './TriggerEditor'
import { ActionChainEditor } from './ActionChainEditor'

const emptyTask = (): Task => ({
  id: '',
  name: '',
  enabled: true,
  profile_ids: [],
  trigger: { type: 'cron', cron: { expr: '0 2 * * *', timezone: 'UTC' } },
  actions: [{ operation: 'backup_db' }],
})

export function TaskForm({
  task,
  hosts,
  notifyChannels = [],
  onSubmit,
  onCancel,
  pending,
}: {
  task?: Task
  hosts: Host[]
  notifyChannels?: NotifyChannel[]
  onSubmit: (values: Task) => void
  onCancel: () => void
  pending?: boolean
}) {
  const { register, handleSubmit, control, watch, setValue } = useForm<Task>({
    defaultValues: task ?? emptyTask(),
  })

  const profileIds = watch('profile_ids') ?? []
  const channelIds = watch('notify_channel_ids') ?? []

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
      <FormSection title="Basic">
        <FormField label="Name" htmlFor="task-name">
          <Input id="task-name" {...register('name', { required: true })} />
        </FormField>
        <div className="flex items-center gap-2">
          <Controller control={control} name="enabled" render={({ field }) => (
            <Checkbox checked={field.value} onCheckedChange={field.onChange} />
          )} />
          <span className="text-sm">Enabled</span>
        </div>
      </FormSection>

      <div>
        <p className="mb-2 text-sm font-medium">Profiles</p>
        <div className="grid gap-2 sm:grid-cols-2">
          {hosts.map((h) => (
            <label key={h.id} className="flex items-center gap-2 text-sm">
              <Checkbox
                checked={profileIds.includes(h.id)}
                onCheckedChange={(c) => {
                  const next = c ? [...profileIds, h.id] : profileIds.filter((x) => x !== h.id)
                  setValue('profile_ids', next)
                }}
              />
              {h.name}
            </label>
          ))}
        </div>
      </div>

      {notifyChannels.length > 0 ? (
        <div>
          <p className="mb-1 text-sm font-medium">Notification channels</p>
          <p className="mb-2 text-xs text-[hsl(var(--muted-foreground))]">
            Leave empty to use all enabled channels. Select one or more to limit notifications for this task.
          </p>
          <div className="grid gap-2 sm:grid-cols-2">
            {notifyChannels.map((ch) => (
              <label key={ch.id} className="flex items-center gap-2 text-sm">
                <Checkbox
                  checked={channelIds.includes(ch.id)}
                  onCheckedChange={(c) => {
                    const next = c ? [...channelIds, ch.id] : channelIds.filter((x) => x !== ch.id)
                    setValue('notify_channel_ids', next)
                  }}
                />
                {ch.name}
                {!ch.enabled ? <span className="text-xs text-[hsl(var(--muted-foreground))]">(disabled)</span> : null}
              </label>
            ))}
          </div>
        </div>
      ) : null}

      <TriggerEditor control={control} />
      <ActionChainEditor control={control} />

      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={onCancel}>Cancel</Button>
        <Button type="submit" disabled={pending}>{task?.id ? 'Save task' : 'Create task'}</Button>
      </div>
    </form>
  )
}
