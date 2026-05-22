import type { OverviewMetrics } from '../api/client'

type Props = {
  metrics: OverviewMetrics | null
  loading?: boolean
}

export function OverviewCharts({ metrics, loading }: Props) {
  if (loading) {
    return <p style={{ color: 'var(--text-muted)', margin: '0 0 16px' }}>加载请求指标…</p>
  }
  if (!metrics || metrics.total === 0) {
    return (
      <div className="panel metrics-empty">
        <p className="empty-hint">
          暂无访问日志数据。请在 <code>ingress.yaml</code> 的 <code>logging</code> 段配置文件输出（启用{' '}
          <code>admin.enabled</code> 且未配置 logging 时会默认同目录 <code>access.log</code>），或确保已有日志内容。
        </p>
      </div>
    )
  }

  const maxTimeline = Math.max(1, ...metrics.timeline.map((b) => b.count))
  const statusTotal = Object.values(metrics.status_counts).reduce((a, b) => a + b, 0) || 1
  const maxHost = Math.max(1, ...metrics.top_hosts.map((h) => h.count))

  return (
    <>
      <div className="cards metrics-cards">
        <div className="card">
          <div className="label">请求量</div>
          <div className="value">{metrics.total}</div>
          <div className="sub">
            ≈ {metrics.rpm.toFixed(1)} 次/分钟 · {metrics.window}
          </div>
        </div>
        <div className={`card ${metrics.error_rate > 5 ? 'warn' : 'ok'}`}>
          <div className="label">错误率</div>
          <div className="value">{metrics.error_rate.toFixed(1)}%</div>
          <div className="sub">4xx + 5xx</div>
        </div>
        <div className="card">
          <div className="label">P95 延迟</div>
          <div className="value">{formatMs(metrics.p95_ms)}</div>
          <div className="sub">P50 {formatMs(metrics.p50_ms)}</div>
        </div>
        <div className="card">
          <div className="label">缓存命中</div>
          <div className="value">{metrics.cache_hit_rate.toFixed(0)}%</div>
          <div className="sub">cache_hit=1</div>
        </div>
        <div className={`card ${metrics.waf_blocks > 0 ? 'warn' : ''}`}>
          <div className="label">WAF 拦截</div>
          <div className="value">{metrics.waf_blocks}</div>
          <div className="sub">日志行 waf_block=1</div>
        </div>
        <div className="card">
          <div className="label">数据源</div>
          <div className="value">{metricsSourceLabel(metrics.source)}</div>
          <div className="sub">access log 聚合</div>
        </div>
      </div>

      <div className="charts-grid">
        <div className="panel chart-panel">
          <div className="panel-head">
            <h2>请求趋势</h2>
            <span className="chart-hint">按时间桶 · 堆叠状态码</span>
          </div>
          <div className="panel-body">
            <div className="timeline-chart">
              {metrics.timeline.map((b, i) => (
                <div key={`${b.label}-${i}`} className="timeline-col" title={`${b.count} 请求`}>
                  <div className="timeline-bar-area">
                    <div
                      className="timeline-stack"
                      style={{
                        height: b.count > 0 ? `${(b.count / maxTimeline) * 100}%` : '0',
                      }}
                    >
                    {b['5xx'] > 0 ? (
                      <span
                        className="seg seg-5xx"
                        style={{ flex: b['5xx'] }}
                      />
                    ) : null}
                    {b['4xx'] > 0 ? (
                      <span className="seg seg-4xx" style={{ flex: b['4xx'] }} />
                    ) : null}
                    {b['3xx'] > 0 ? (
                      <span className="seg seg-3xx" style={{ flex: b['3xx'] }} />
                    ) : null}
                    {b['2xx'] > 0 ? (
                      <span className="seg seg-2xx" style={{ flex: b['2xx'] }} />
                    ) : null}
                    </div>
                  </div>
                  <span className="timeline-label">{b.label}</span>
                </div>
              ))}
            </div>
            <div className="chart-legend">
              <span className="leg seg-2xx">2xx</span>
              <span className="leg seg-3xx">3xx</span>
              <span className="leg seg-4xx">4xx</span>
              <span className="leg seg-5xx">5xx</span>
            </div>
          </div>
        </div>

        <div className="panel chart-panel">
          <div className="panel-head">
            <h2>状态码分布</h2>
          </div>
          <div className="panel-body">
            {(['2xx', '3xx', '4xx', '5xx'] as const).map((k) => {
              const n = metrics.status_counts[k] ?? 0
              return (
                <div key={k} className="bar-row">
                  <span className={`bar-label seg-${k}`}>{k}</span>
                  <div className="bar-track">
                    <div
                      className={`bar-fill seg-${k}`}
                      style={{ width: `${(n / statusTotal) * 100}%` }}
                    />
                  </div>
                  <span className="bar-val">{n}</span>
                </div>
              )
            })}
          </div>
        </div>
      </div>

      <div className="charts-grid">
        <div className="panel chart-panel">
          <div className="panel-head">
            <h2>Top Host</h2>
          </div>
          <div className="panel-body">
            {metrics.top_hosts.length === 0 ? (
              <p className="empty-hint">无数据</p>
            ) : (
              metrics.top_hosts.map((h) => (
                <div key={h.name} className="bar-row host-rank">
                  <span className="bar-label host-label" title={h.name}>
                    {h.name}
                  </span>
                  <div className="bar-track">
                    <div
                      className="bar-fill seg-2xx"
                      style={{ width: `${(h.count / maxHost) * 100}%` }}
                    />
                  </div>
                  <span className="bar-val">{h.count}</span>
                </div>
              ))
            )}
          </div>
        </div>

        <div className="panel chart-panel">
          <div className="panel-head">
            <h2>最慢请求</h2>
          </div>
          <div className="panel-body panel-table-wrap">
            <table className="data compact">
              <thead>
                <tr>
                  <th>Host</th>
                  <th>请求</th>
                  <th>状态</th>
                  <th>耗时</th>
                </tr>
              </thead>
              <tbody>
                {metrics.slowest.length === 0 ? (
                  <tr>
                    <td colSpan={4} className="empty-hint">
                      无延迟数据
                    </td>
                  </tr>
                ) : (
                  metrics.slowest.map((s, i) => (
                    <tr key={`${s.host}-${s.path}-${i}`}>
                      <td>{s.host}</td>
                      <td>
                        <code>
                          {s.method} {s.path}
                        </code>
                      </td>
                      <td>{s.status}</td>
                      <td>{formatMs(s.duration_ms)}</td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </>
  )
}

function formatMs(ms: number) {
  if (ms >= 1000) return `${(ms / 1000).toFixed(2)}s`
  return `${Math.round(ms)}ms`
}

function metricsSourceLabel(source: string) {
  switch (source) {
    case 'access_log':
      return 'access.log'
    case 'access_log_empty':
      return '空文件'
    case 'unconfigured':
      return '未配置'
    case 'error':
      return '读取失败'
    default:
      return source
  }
}
