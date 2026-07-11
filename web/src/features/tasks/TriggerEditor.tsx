import { Controller, type Control } from 'react-hook-form'
import { Input } from '@/components/ui/input'
import { FormField } from '@/components/forms/FormField'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import type { Task, TriggerType } from '@/api/types'

const TRIGGER_TYPES: { value: TriggerType; label: string }[] = [
  { value: 'cron', label: 'Cron' },
  { value: 'interval', label: 'Interval' },
  { value: 'one_shot', label: 'One-shot' },
  { value: 'on_boot', label: 'On boot' },
]

export function TriggerEditor({ control }: { control: Control<Task> }) {
  return (
    <div className="grid gap-4 sm:grid-cols-2">
      <FormField label="Trigger type">
        <Controller
          control={control}
          name="trigger.type"
          render={({ field }) => (
            <Select value={field.value} onValueChange={field.onChange}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                {TRIGGER_TYPES.map((t) => <SelectItem key={t.value} value={t.value}>{t.label}</SelectItem>)}
              </SelectContent>
            </Select>
          )}
        />
      </FormField>

      <Controller
        control={control}
        name="trigger.type"
        render={({ field: typeField }) => {
          if (typeField.value === 'cron') {
            return (
              <>
                <FormField label="Cron expression" htmlFor="cron-expr">
                  <Controller control={control} name="trigger.cron.expr" render={({ field }) => (
                    <Input id="cron-expr" {...field} placeholder="0 2 * * *" />
                  )} />
                </FormField>
                <FormField label="Timezone" htmlFor="cron-tz">
                  <Controller control={control} name="trigger.cron.timezone" render={({ field }) => (
                    <Input id="cron-tz" {...field} placeholder="UTC" />
                  )} />
                </FormField>
              </>
            )
          }
          if (typeField.value === 'interval') {
            return (
              <FormField label="Every (seconds)" htmlFor="interval-every" className="sm:col-span-2">
                <Controller
                  control={control}
                  name="trigger.interval.every"
                  render={({ field }) => (
                    <Input
                      id="interval-every"
                      type="number"
                      value={field.value ? Math.round(field.value / 1e9) : ''}
                      onChange={(e) => field.onChange(Number(e.target.value) * 1e9)}
                    />
                  )}
                />
              </FormField>
            )
          }
          if (typeField.value === 'one_shot') {
            return (
              <FormField label="Run at" htmlFor="oneshot-at" className="sm:col-span-2">
                <Controller
                  control={control}
                  name="trigger.one_shot.at"
                  render={({ field }) => (
                    <Input
                      id="oneshot-at"
                      type="datetime-local"
                      value={field.value ? field.value.slice(0, 16) : ''}
                      onChange={(e) => field.onChange(new Date(e.target.value).toISOString())}
                    />
                  )}
                />
              </FormField>
            )
          }
          if (typeField.value === 'on_boot') {
            return (
              <FormField label="Delay (seconds)" htmlFor="onboot-delay" className="sm:col-span-2">
                <Controller
                  control={control}
                  name="trigger.on_boot.delay"
                  render={({ field }) => (
                    <Input
                      id="onboot-delay"
                      type="number"
                      value={field.value ? Math.round(field.value / 1e9) : ''}
                      onChange={(e) => field.onChange(Number(e.target.value) * 1e9)}
                    />
                  )}
                />
              </FormField>
            )
          }
          return <></>
        }}
      />
    </div>
  )
}
