import { useCallback } from 'react'
import DateObjectImport from 'react-date-object'
import persianImport from 'react-date-object/calendars/persian'
import persian_faImport from 'react-date-object/locales/persian_fa'
import gregorianImport from 'react-date-object/calendars/gregorian'
import gregorian_enImport from 'react-date-object/locales/gregorian_en'
import { importDefault } from '@/lib/import-default'
import { useUIStore, type CalendarPref } from '@/store/ui-store'

const DateObject = importDefault(DateObjectImport)
const persian = importDefault(persianImport)
const persian_fa = importDefault(persian_faImport)
const gregorian = importDefault(gregorianImport)
const gregorian_en = importDefault(gregorian_enImport)

function calendarConfig(cal: CalendarPref) {
  if (cal === 'jalali') return { calendar: persian, locale: persian_fa }
  return { calendar: gregorian, locale: gregorian_en }
}

export function formatDateWith(
  value?: string | Date,
  calendar: CalendarPref = 'jalali',
  format = 'YYYY/MM/DD HH:mm:ss',
) {
  if (!value) return '—'
  const d = typeof value === 'string' ? new Date(value) : value
  if (Number.isNaN(d.getTime())) return '—'
  const { calendar: cal, locale } = calendarConfig(calendar)
  return new DateObject(d).convert(cal).setLocale(locale).format(format)
}

export function useFormatDate() {
  const calendar = useUIStore((s) => s.calendar)
  return useCallback(
    (value?: string | Date, format?: string) => formatDateWith(value, calendar, format),
    [calendar],
  )
}

export function getDatePickerConfig(calendar: CalendarPref) {
  return calendarConfig(calendar)
}
