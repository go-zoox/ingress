import { memo } from 'react'
import { useAnimatedListRows } from '../hooks/useAnimatedListRows'
import { useListFlip } from '../hooks/useListFlip'
import { listAnimPhaseClass } from '../lib/listAnim'

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
  const { rows: animRows, flipKeys } = useAnimatedListRows(rows, (row) => row.name)
  const registerFlip = useListFlip(flipKeys)

  if (animRows.length === 0) {
    return <p className="empty-hint">{emptyText}</p>
  }

  const liveRows = animRows.filter((row) => row.phase !== 'exit')
  const max = maxValue ?? Math.max(1, ...liveRows.map((row) => row.item.value))
  const fillClass = tone === 'ok' ? 'seg-2xx' : 'seg-4xx'

  return (
    <>
      {animRows.map((row) => (
        <div
          key={row.key}
          ref={registerFlip(row.key)}
          className={`bar-row host-rank${listAnimPhaseClass(row.phase)}`}
        >
          <span className="bar-label host-label" title={row.item.name}>
            {row.item.name}
          </span>
          <div className="bar-track">
            <div
              className={`bar-fill ${fillClass}`}
              style={{ width: `${(row.item.value / max) * 100}%` }}
            />
          </div>
          <span className="bar-val">{row.item.sub}</span>
        </div>
      ))}
    </>
  )
})
