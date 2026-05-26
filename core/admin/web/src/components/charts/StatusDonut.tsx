import { memo, useMemo } from 'react'

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
  const segments = useMemo(() => {
    const total = ORDER.reduce((s, k) => s + (counts[k] ?? 0), 0) || 1
    let offset = 0
    return ORDER.map((k) => {
      const n = counts[k] ?? 0
      const pct = (n / total) * 100
      const seg = { key: k, n, pct, offset }
      offset += pct
      return seg
    }).filter((s) => s.n > 0)
  }, [counts])

  const total = ORDER.reduce((s, k) => s + (counts[k] ?? 0), 0)

  if (total === 0) {
    return <p className="empty-hint">无状态码数据</p>
  }

  const gradient = segments
    .map((s) => `${COLORS[s.key]} ${s.offset}% ${s.offset + s.pct}%`)
    .join(', ')

  return (
    <div className="status-donut">
      <div
        className="status-donut-ring"
        style={{ background: `conic-gradient(${gradient})` }}
        aria-hidden
      >
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
