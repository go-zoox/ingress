import { memo, useMemo } from 'react'
import { buildStatusDonutOption } from '../../lib/overviewEChartsOptions'
import { readChartColors } from './chartTheme'
import { EChartView } from './EChartView'

type Props = {
  counts: Record<string, number>
}

const ORDER = ['2xx', '3xx', '4xx', '5xx'] as const
const COLORS: Record<string, string> = {
  '2xx': 'var(--ok)',
  '3xx': 'var(--accent)',
  '4xx': 'var(--warn)',
  '5xx': 'var(--danger)',
}

export const StatusDonut = memo(function StatusDonut({ counts }: Props) {
  const colors = useMemo(() => readChartColors(), [])
  const option = useMemo(() => buildStatusDonutOption(counts, colors), [counts, colors])
  const total = ORDER.reduce((s, k) => s + (counts[k] ?? 0), 0)

  if (total === 0 || !option) {
    return <p className="empty-hint">无状态码数据</p>
  }

  return (
    <div className="status-donut">
      <div className="status-donut-ring status-donut-ring-echarts">
        <EChartView option={option} className="status-donut-echarts" height={120} notMerge />
        <div className="status-donut-hole">
          <span className="status-donut-total">{total}</span>
          <span className="status-donut-sub">请求</span>
        </div>
      </div>
      <ul className="status-donut-legend">
        {ORDER.map((k) => {
          const n = counts[k] ?? 0
          if (n === 0) return null
          const pct = ((n / total) * 100).toFixed(1)
          return (
            <li key={k}>
              <span className="dot" style={{ background: COLORS[k] }} />
              <span className="name">{k}</span>
              <span className="val">
                {n} ({pct}%)
              </span>
            </li>
          )
        })}
      </ul>
    </div>
  )
})
