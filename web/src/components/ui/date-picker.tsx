import RMDP from 'react-multi-date-picker'
import type { DateObject } from 'react-multi-date-picker'
import TimePickerPlugin from 'react-multi-date-picker/plugins/time_picker'
import gregorianImport from 'react-date-object/calendars/gregorian'
import { cn } from '@/lib/utils'
import { getDatePickerConfig } from '@/lib/datetime'
import { importDefault } from '@/lib/import-default'
import { useUIStore } from '@/store/ui-store'
import 'react-multi-date-picker/styles/colors/teal.css'

const DatePicker = importDefault(RMDP)
const TimePicker = importDefault(TimePickerPlugin)
const gregorian = importDefault(gregorianImport)

function toGregorianDateString(d: DateObject | null): string {
  if (!d) return ''
  const g = d.convert(gregorian)
  const y = g.year
  const m = String(g.month.number).padStart(2, '0')
  const day = String(g.day).padStart(2, '0')
  return `${y}-${m}-${day}`
}

function toGregorianISO(d: DateObject | null): string {
  if (!d) return ''
  const date = d.toDate()
  if (!date || Number.isNaN(date.getTime())) return ''
  return date.toISOString()
}

function parseValue(value: string, withTime: boolean): Date | undefined {
  if (!value) return undefined
  const d = new Date(withTime ? value : `${value}T12:00:00`)
  return Number.isNaN(d.getTime()) ? undefined : d
}

export function DatePickerField({
  value,
  onChange,
  mode = 'date',
  className,
  id,
  placeholder,
}: {
  value: string
  onChange: (value: string) => void
  mode?: 'date' | 'datetime'
  className?: string
  id?: string
  placeholder?: string
}) {
  const calendar = useUIStore((s) => s.calendar)
  const { calendar: cal, locale } = getDatePickerConfig(calendar)
  const withTime = mode === 'datetime'

  return (
    <DatePicker
      id={id}
      value={parseValue(value, withTime)}
      onChange={(d) => {
        const obj = Array.isArray(d) ? d[0] : d
        if (!obj || typeof obj === 'string') {
          onChange('')
          return
        }
        onChange(withTime ? toGregorianISO(obj as DateObject) : toGregorianDateString(obj as DateObject))
      }}
      calendar={cal}
      locale={locale}
      format={withTime ? 'YYYY/MM/DD HH:mm' : 'YYYY/MM/DD'}
      plugins={withTime ? [<TimePicker key="time" position="bottom" hideSeconds />] : undefined}
      containerClassName={cn('w-full', className)}
      inputClass={cn(
        'flex h-9 w-full rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--background))] px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[hsl(var(--primary))]',
        className,
      )}
      placeholder={placeholder}
    />
  )
}
