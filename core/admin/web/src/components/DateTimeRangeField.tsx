import { useId } from 'react'
import { FormInput, FormItem } from './Form'
import {
  datetimeLocalToRFC3339,
  formatRFC3339Display,
  isDateTimeRangeOrdered,
  rfc3339ToDatetimeLocal,
  type DateTimeRangeValue,
} from '../lib/datetimeRange'

export function DateTimeRangeField({
  label = '维护时间',
  hint = '留空表示该端不限制；保存为 RFC3339（本地时区）',
  start,
  end,
  onChange,
  showDisplayHint = false,
}: {
  label?: string
  hint?: string
  start: string
  end: string
  onChange: (next: DateTimeRangeValue) => void
  showDisplayHint?: boolean
}) {
  const startId = useId()
  const endId = useId()
  const ordered = isDateTimeRangeOrdered(start, end)

  const patchStart = (local: string) => {
    onChange({ start: datetimeLocalToRFC3339(local), end })
  }

  const patchEnd = (local: string) => {
    onChange({ start, end: datetimeLocalToRFC3339(local) })
  }

  const clear = () => onChange({ start: '', end: '' })

  return (
    <FormItem label={label} hint={hint} full>
      <div className="datetime-range-field">
        <FormInput
          id={startId}
          type="datetime-local"
          aria-label={`${label} 开始`}
          value={rfc3339ToDatetimeLocal(start)}
          onChange={(e) => patchStart(e.target.value)}
        />
        <span className="datetime-range-sep" aria-hidden>
          至
        </span>
        <FormInput
          id={endId}
          type="datetime-local"
          aria-label={`${label} 结束`}
          value={rfc3339ToDatetimeLocal(end)}
          onChange={(e) => patchEnd(e.target.value)}
        />
        {(start || end) ? (
          <button type="button" className="btn btn-sm btn-ghost datetime-range-clear" onClick={clear}>
            清除
          </button>
        ) : null}
      </div>
      {!ordered ? (
        <p className="form-hint form-hint-warn">结束时间应不早于开始时间</p>
      ) : null}
      {showDisplayHint && (start || end) ? (
        <p className="form-hint datetime-range-preview">
          {start ? formatRFC3339Display(start) : '不限开始'}
          {' → '}
          {end ? formatRFC3339Display(end) : '不限结束'}
        </p>
      ) : null}
    </FormItem>
  )
}

export { formatRFC3339Display }
