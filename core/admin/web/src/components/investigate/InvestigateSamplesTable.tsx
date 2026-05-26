import type { InvestigateSample } from '../../api/client'

type Props = {
  samples: InvestigateSample[]
  anchorStatus?: string
}

export function InvestigateSamplesTable({ samples, anchorStatus }: Props) {
  if (samples.length === 0) {
    return <p className="empty-hint">无匹配的访问日志样本（放宽 path 或检查 logging 配置）</p>
  }

  return (
    <table className="data investigate-samples-table">
      <thead>
        <tr>
          <th>方法</th>
          <th>路径</th>
          <th>状态</th>
          <th>耗时</th>
          <th>上游</th>
          <th>缓存</th>
          <th>WAF</th>
        </tr>
      </thead>
      <tbody>
        {samples.map((s, i) => {
          const anchored =
            anchorStatus != null &&
            anchorStatus !== '' &&
            String(s.status).startsWith(anchorStatus)
          const statusClass =
            s.status >= 500 || s.status >= 400 ? 'status-4xx' : s.status >= 200 && s.status < 300 ? 'status-2xx' : ''
          return (
            <tr
              key={`${i}-${s.method}-${s.path}-${s.status}`}
              className={`${statusClass}${anchored ? ' match-highlight' : ''}`}
            >
              <td>{s.method}</td>
              <td>
                <code className="investigate-path-cell">{s.path}</code>
              </td>
              <td>{s.status}</td>
              <td>{formatMs(s.duration_ms)}</td>
              <td>
                {s.upstream_duration_ms != null && s.upstream_duration_ms > 0
                  ? formatMs(s.upstream_duration_ms)
                  : '—'}
              </td>
              <td>{s.cache_hit ? '命中' : '—'}</td>
              <td>{s.waf_block ? '拦截' : '—'}</td>
            </tr>
          )
        })}
      </tbody>
    </table>
  )
}

function formatMs(ms: number) {
  if (ms >= 1000) return `${(ms / 1000).toFixed(2)}s`
  return `${Math.round(ms)}ms`
}
