import { memo } from 'react'
import { Minus, TrendingDown, TrendingUp } from 'lucide-react'
import type { OverviewMetrics } from '../api/client'

type Props = {
  delta: OverviewMetrics['delta']
  kind: 'pct' | 'pp' | 'count' | 'ms'
  value: number
  /** When true, an increase is shown as negative (bad) styling. */
  badIfUp?: boolean
}

export const OverviewDelta = memo(function OverviewDelta({ delta, kind, value, badIfUp }: Props) {
  if (!delta?.has_previous) {
    return <span className="kpi-delta kpi-delta-na">—</span>
  }

  const abs = Math.abs(value)
  const flat = kind === 'pp' ? abs < 0.05 : kind === 'ms' ? abs < 1 : kind === 'count' ? abs === 0 : abs < 0.5

  if (flat) {
    return (
      <span className="kpi-delta kpi-delta-flat">
        <Minus size={12} aria-hidden />
        环比持平
      </span>
    )
  }

  const up = value > 0
  const isBad = badIfUp ? up : !up
  const cls = isBad ? 'kpi-delta-bad' : 'kpi-delta-good'
  const Icon = up ? TrendingUp : TrendingDown

  return (
    <span className={`kpi-delta ${cls}`}>
      <Icon size={12} aria-hidden />
      {formatDelta(kind, value)}
    </span>
  )
})

function formatDelta(kind: Props['kind'], value: number) {
  const sign = value > 0 ? '+' : ''
  switch (kind) {
    case 'pp':
      return `${sign}${value.toFixed(1)} pp`
    case 'ms':
      return `${sign}${Math.round(value)} ms`
    case 'count':
      return `${sign}${value}`
    default:
      return `${sign}${value.toFixed(0)}%`
  }
}
