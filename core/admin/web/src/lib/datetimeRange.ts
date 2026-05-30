/** RFC3339 ↔ browser datetime-local conversion and display helpers. */

function pad2(n: number): string {
  return String(n).padStart(2, '0')
}

/** Parse RFC3339 (or ISO) into Date; returns null when invalid. */
export function parseRFC3339(value: string): Date | null {
  const trimmed = value.trim()
  if (!trimmed) return null
  const d = new Date(trimmed)
  return Number.isNaN(d.getTime()) ? null : d
}

/** Format RFC3339 for `<input type="datetime-local">` in the user's local timezone. */
export function rfc3339ToDatetimeLocal(value: string): string {
  const d = parseRFC3339(value)
  if (!d) return ''
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}T${pad2(d.getHours())}:${pad2(d.getMinutes())}`
}

/** Convert datetime-local value to RFC3339 with local timezone offset. */
export function datetimeLocalToRFC3339(value: string): string {
  const trimmed = value.trim()
  if (!trimmed) return ''
  const d = new Date(trimmed)
  if (Number.isNaN(d.getTime())) return ''

  const offsetMin = -d.getTimezoneOffset()
  const sign = offsetMin >= 0 ? '+' : '-'
  const abs = Math.abs(offsetMin)
  const oh = pad2(Math.floor(abs / 60))
  const om = pad2(abs % 60)

  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}T${pad2(d.getHours())}:${pad2(d.getMinutes())}:${pad2(d.getSeconds())}${sign}${oh}:${om}`
}

const DISPLAY_OPTS: Intl.DateTimeFormatOptions = {
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  hour12: false,
}

/** Human-readable local datetime for tables; falls back to raw string when unparsable. */
export function formatRFC3339Display(value: string): string {
  const d = parseRFC3339(value)
  if (!d) return value.trim() || '—'
  return d.toLocaleString('zh-CN', DISPLAY_OPTS)
}

export type DateTimeRangeValue = {
  start: string
  end: string
}

export function isDateTimeRangeOrdered(start: string, end: string): boolean {
  const a = parseRFC3339(start)
  const b = parseRFC3339(end)
  if (!a || !b) return true
  return a.getTime() <= b.getTime()
}
