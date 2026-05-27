import { memo } from 'react'
import type { BackendStat } from '../api/client'

type Props = {
  backends: BackendStat[]
}

export const BackendPerformancePanel = memo(function BackendPerformancePanel({ backends }: Props) {
  if (backends.length === 0) {
    return <p className="empty-hint">暂无后端流量（handler 或未解析 target）</p>
  }

  return (
    <div className="panel-table-wrap">
      <table className="data compact backend-perf-table">
        <thead>
          <tr>
            <th>后端</th>
            <th>请求量</th>
            <th>次/分</th>
            <th>上游 P95</th>
            <th>上游 5xx</th>
          </tr>
        </thead>
        <tbody>
          {backends.map((b) => (
            <tr key={b.name}>
              <td>
                <code className="backend-target" title={b.name}>
                  {b.name}
                </code>
              </td>
              <td>{b.count}</td>
              <td>{b.rpm.toFixed(1)}</td>
              <td className={latencyTone(b.upstream_p95_ms)}>{formatMs(b.upstream_p95_ms)}</td>
              <td className={b.upstream_error_pct > 1 ? 'text-danger' : undefined}>
                {b.upstream_error_pct.toFixed(1)}%
              </td>
            </tr>
          ))}
        </tbody>
      </table>
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
