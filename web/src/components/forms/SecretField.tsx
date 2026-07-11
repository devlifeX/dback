import { Eye, EyeOff } from 'lucide-react'
import { useState } from 'react'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { FormField } from './FormField'

export function SecretField({
  label,
  value,
  onChange,
  placeholder,
  error,
  hint,
  id,
}: {
  label: string
  value: string
  onChange: (value: string) => void
  placeholder?: string
  error?: string
  hint?: string
  id?: string
}) {
  const [visible, setVisible] = useState(false)
  const fieldId = id ?? label.toLowerCase().replace(/\s+/g, '-')

  return (
    <FormField label={label} htmlFor={fieldId} error={error} hint={hint}>
      <div className="relative">
        <Input
          id={fieldId}
          type={visible ? 'text' : 'password'}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder}
          className="pr-10"
          autoComplete="off"
        />
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="absolute right-0 top-0 h-9 w-9"
          onClick={() => setVisible((v) => !v)}
          tabIndex={-1}
        >
          {visible ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
        </Button>
      </div>
    </FormField>
  )
}
