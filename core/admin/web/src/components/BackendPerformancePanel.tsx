import { memo } from 'react'
import type { BackendStat } from '../api/client'
import { useAnimatedListRows } from '../hooks/useAnimatedListRows'
import { useListFlip } from '../hooks/useListFlip'
import { listAnimPhaseClass } from '../lib/listAnim'

type Props = {
  backends: BackendStat[]
}

export const BackendPerformancePanel = memo(function BackendPerformancePanel({ backends }: Props) {
  const { rows: animRows, flipKeys } = useAnimatedListRows(backends, (b) => b.name)
  const registerFlip = useListFlip(flipKeys)

  if (animRows.length === 0) {
    return <p className="empty-hint">暂无后端流量（handler 或未解析 target）</p>
  }

  return (
    <div className="panel-table-wrap">
      <div className="backend-perf-grid data compact">
        <div className="backend-perf-row backend-perf-head">
          <span>后端</span>
          <span>请求量</span>
          <span>次/分</span>
          <span>上游 P95</span>
          <span>上游 5xx</span>
        </div>
        {animRows.map((row) => {
          const b = row.item
          return (
            <div
              key={row.key}
              ref={registerFlip(row.key)}
              className={`backend-perf-row${listAnimPhaseClass(row.phase)}`}
            >
              <span>
                <code className="backend-target" title={b.name}>
                  {b.name}
                </code>
              </span>
              <span>{b.count}</span>
              <span>{b.rpm.toFixed(1)}</span>
              <span className={latencyTone(b.upstream_p95_ms)}>{formatMs(b.upstream_p95_ms)}</span>
              <span className={b.upstream_error_pct > 1 ? 'text-danger' : undefined}>
                {b.upstream_error_pct.toFixed(1)}%
              </span>
            </div>
          )
        })}
      </div>
    </div>
  )
})

function formatMs(ms: number) {
  if (ms <= 0) return '—'
  if (ms >= 1000) return `${(ms / 1000).toFixed(2)}s`
  return `${Math.round(ms)}ms`
}

function latencyTone(ms: number) {
  if (ms <= 0) return undefined
  if (ms > 2000) return 'text-danger'
  if (ms > 500) return 'text-warn'
  return undefined
}
