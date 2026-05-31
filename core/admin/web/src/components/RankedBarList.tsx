import { memo } from 'react'

export type RankedBarRow = {
  name: string
  value: number
  sub: string
}

type Props = {
  rows: RankedBarRow[]
  tone: 'ok' | 'warn'
  maxValue?: number
  emptyText?: string
}

export const RankedBarList = memo(function RankedBarList({
  rows,
  tone,
  maxValue,
  emptyText = '无数据',
}: Props) {
  if (rows.length === 0) {
    return <p className="empty-hint">{emptyText}</p>
  }

  const max = maxValue ?? Math.max(1, ...rows.map((row) => row.value))
  const fillClass = tone === 'ok' ? 'seg-2xx' : 'seg-4xx'

  return (
    <div className="ranked-bar-list">
      {rows.map((row) => (
        <div key={row.name} className="bar-row host-rank">
          <span className="bar-label host-label" title={row.name}>
            {row.name}
          </span>
          <div className="bar-track">
            <div
              className={`bar-fill ${fillClass}`}
              style={{ width: `${(row.value / max) * 100}%` }}
            />
          </div>
          <span className="bar-val">{row.sub}</span>
        </div>
      ))}
    </div>
  )
})
