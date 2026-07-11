import { useFieldArray, type Control } from 'react-hook-form'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { FormField } from '@/components/forms/FormField'
import { OPERATION_KINDS, type Task } from '@/api/types'
import { Controller } from 'react-hook-form'

export function ActionChainEditor({ control }: { control: Control<Task> }) {
  const { fields, append, remove } = useFieldArray({ control, name: 'actions' })

  return (
    <div className="space-y-3">
      <p className="text-sm font-medium">Action chain</p>
      {fields.map((field, index) => (
        <div key={field.id} className="flex items-end gap-2">
          <span className="pb-2 text-sm text-[hsl(var(--muted-foreground))]">{index + 1}.</span>
          <FormField label={index === 0 ? 'Operation' : ''} className="flex-1">
            <Controller
              control={control}
              name={`actions.${index}.operation`}
              render={({ field: f }) => (
                <Select value={f.value} onValueChange={f.onChange}>
                  <SelectTrigger><SelectValue placeholder="Select operation" /></SelectTrigger>
                  <SelectContent>
                    {OPERATION_KINDS.map((k) => <SelectItem key={k.value} value={k.value}>{k.label}</SelectItem>)}
                  </SelectContent>
                </Select>
              )}
            />
          </FormField>
          <Button type="button" variant="outline" size="sm" onClick={() => remove(index)}>Remove</Button>
        </div>
      ))}
      <Button type="button" variant="outline" size="sm" onClick={() => append({ operation: 'backup_db' })}>
        Add action
      </Button>
    </div>
  )
}
