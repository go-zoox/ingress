import { useEffect, useRef, useState } from 'react'
import { CalendarRange, ChevronDown } from 'lucide-react'
import {
  OVERVIEW_RANGE_PRESETS,
  type OverviewRange,
  formatOverviewRangeLabel,
  fromDatetimeLocalValue,
  localDayBounds,
  toDatetimeLocalValue,
} from '../lib/overviewRange'

type Props = {
  value: OverviewRange
  onChange: (range: OverviewRange) => void
  disabled?: boolean
}

export function OverviewTimeRangePicker({ value, onChange, disabled }: Props) {
  const [open, setOpen] = useState(false)
  const [draftFrom, setDraftFrom] = useState('')
  const [draftTo, setDraftTo] = useState('')
  const panelRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (value.kind === 'absolute') {
      setDraftFrom(toDatetimeLocalValue(value.from))
      setDraftTo(toDatetimeLocalValue(value.to))
    }
  }, [value])

  useEffect(() => {
    if (!open) return
    const onDoc = (e: MouseEvent) => {
      if (panelRef.current && !panelRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', onDoc)
    return () => document.removeEventListener('mousedown', onDoc)
  }, [open])

  const applyAbsolute = () => {
    const from = fromDatetimeLocalValue(draftFrom)
    const to = fromDatetimeLocalValue(draftTo)
    if (!from || !to) return
    if (new Date(to) <= new Date(from)) return
    onChange({ kind: 'absolute', from, to })
    setOpen(false)
  }

  const triggerLabel = formatOverviewRangeLabel(value)

  return (
    <div className="overview-range-picker" ref={panelRef}>
      <button
        type="button"
        className="btn btn-sm btn-ghost overview-range-trigger"
        disabled={disabled}
        aria-haspopup="dialog"
        aria-expanded={open}
        onClick={() => setOpen((v) => !v)}
      >
        <CalendarRange size={14} aria-hidden />
        <span>{triggerLabel}</span>
        <ChevronDown size={14} aria-hidden />
      </button>

      {open ? (
        <div className="overview-range-panel" role="dialog" aria-label="选择时间范围">
          <p className="overview-range-panel-title">相对范围</p>
          <div className="overview-range-presets">
            {OVERVIEW_RANGE_PRESETS.map((opt) => (
              <button
                key={opt.value}
                type="button"
                className={
                  value.kind === 'preset' && value.preset === opt.value
                    ? 'btn btn-sm active'
                    : 'btn btn-sm btn-ghost'
                }
                onClick={() => {
                  onChange({ kind: 'preset', preset: opt.value })
                  setOpen(false)
                }}
              >
                {opt.label}
              </button>
            ))}
          </div>

          <p className="overview-range-panel-title">快捷日期</p>
          <div className="overview-range-presets">
            <button
              type="button"
              className="btn btn-sm btn-ghost"
              onClick={() => {
                const { from, to } = localDayBounds(0)
                onChange({ kind: 'absolute', from, to })
                setOpen(false)
              }}
            >
              今天
            </button>
            <button
              type="button"
              className="btn btn-sm btn-ghost"
              onClick={() => {
                const { from, to } = localDayBounds(-1)
                onChange({ kind: 'absolute', from, to })
                setOpen(false)
              }}
            >
              昨天
            </button>
          </div>

          <p className="overview-range-panel-title">自定义区间</p>
          <div className="overview-range-custom">
            <label>
              <span>开始</span>
              <input
                type="datetime-local"
                className="form-control"
                value={draftFrom}
                onChange={(e) => setDraftFrom(e.target.value)}
              />
            </label>
            <label>
              <span>结束</span>
              <input
                type="datetime-local"
                className="form-control"
                value={draftTo}
                onChange={(e) => setDraftTo(e.target.value)}
              />
            </label>
            <button type="button" className="btn btn-sm btn-primary" onClick={applyAbsolute}>
              应用
            </button>
          </div>
        </div>
      ) : null}
    </div>
  )
}
