import { memo, useMemo, type CSSProperties } from 'react'
import type { OverviewMetrics } from '../api/client'

type Props = {
  metrics: OverviewMetrics | null
  loading?: boolean
}

export const OverviewCharts = memo(function OverviewCharts({ metrics, loading }: Props) {
  // Only show loading spinner on the very first render when there's no data yet.
  // Subsequent refreshes keep showing the previous data seamlessly.
  if (loading && metrics === null) {
    return <p style={{ color: 'var(--text-muted)', margin: '0 0 16px' }}>加载请求指标…</p>
  }
  if (!metrics || metrics.total === 0) {
    return (
      <div className="panel metrics-empty">
        {metrics && metrics.source === 'access_log_empty' ? (
          <p className="empty-hint">
            日志文件已配置但尚无内容。等待请求产生后数据会自动刷新。
          </p>
        ) : metrics && metrics.source === 'unconfigured' ? (
          <p className="empty-hint">
            未配置日志路径。请在 <code>ingress.yaml</code> 的 <code>logging</code> 段配置文件输出，或启用{' '}
            <code>admin.enabled</code>（默认在同目录生成 <code>access.log</code>）。
          </p>
        ) : metrics && metrics.source === 'error' ? (
          <p className="empty-hint">
            读取日志文件失败。请检查日志路径是否正确。
          </p>
        ) : metrics && metrics.source === 'access_log_parse_fail' ? (
          <p className="empty-hint">
            日志文件有内容但无法解析。请检查日志格式是否兼容。
          </p>
        ) : (
          <p className="empty-hint">
            暂无访问日志数据。等待请求产生后数据会自动刷新。
          </p>
        )}
      </div>
    )
  }

  return (
    <>
      <MetricsCards metrics={metrics} />
      <div className="charts-grid">
        <TimelinePanel metrics={metrics} />
        <StatusPanel metrics={metrics} />
      </div>
      <div className="charts-grid">
        <HostPanel metrics={metrics} />
        <SlowestPanel metrics={metrics} />
      </div>
    </>
  )
})

/* ── Metric summary cards ── */

const MetricsCards = memo(function MetricsCards({ metrics }: { metrics: OverviewMetrics }) {
  return (
    <div className="cards metrics-cards">
      <MetricCard label="请求量" value={String(metrics.total)} sub={`≈ ${metrics.rpm.toFixed(1)} 次/分钟 · ${metrics.window}`} />
      <MetricCard
        label="错误率"
        value={`${metrics.error_rate.toFixed(1)}%`}
        sub="4xx + 5xx"
        warn={metrics.error_rate > 5}
      />
      <MetricCard label="P95 延迟" value={formatMs(metrics.p95_ms)} sub={`P50 ${formatMs(metrics.p50_ms)}`} />
      <MetricCard label="缓存命中" value={`${metrics.cache_hit_rate.toFixed(0)}%`} sub="cache_hit=1" />
      <MetricCard label="WAF 拦截" value={String(metrics.waf_blocks)} sub="日志行 waf_block=1" warn={metrics.waf_blocks > 0} />
      <MetricCard label="数据源" value={metricsSourceLabel(metrics.source)} sub="access log 聚合" />
    </div>
  )
})

const MetricCard = memo(function MetricCard({
  label,
  value,
  sub,
  warn,
  ok,
}: {
  label: string
  value: string
  sub: string
  warn?: boolean
  ok?: boolean
}) {
  const cls = warn ? 'warn' : ok ? 'ok' : ''
  return (
    <div className={cls ? `card ${cls}` : 'card'}>
      <div className="label">{label}</div>
      <div className="value">{value}</div>
      <div className="sub">{sub}</div>
    </div>
  )
})

/* ── Timeline chart ── */

const TimelinePanel = memo(function TimelinePanel({ metrics }: { metrics: OverviewMetrics }) {
  const maxTimeline = Math.max(1, ...metrics.timeline.map((b) => b.count))

  return (
    <div className="panel chart-panel">
      <div className="panel-head">
        <h2>请求趋势</h2>
        <span className="chart-hint">按时间桶 · 堆叠状态码</span>
      </div>
      <div className="panel-body">
        <div className="timeline-chart">
          {metrics.timeline.map((b, i) => (
            <TimelineCol key={b.label} bucket={b} max={maxTimeline} />
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
  )
})

const TimelineCol = memo(function TimelineCol({
  bucket,
  max,
}: {
  bucket: OverviewMetrics['timeline'][number]
  max: number
}) {
  const heightPct = bucket.count > 0 ? (bucket.count / max) * 100 : 0
  const style: CSSProperties = { height: heightPct > 0 ? `${heightPct}%` : '0' }

  return (
    <div className="timeline-col" title={`${bucket.count} 请求`}>
      <div className="timeline-bar-area">
        <div className="timeline-stack" style={style}>
          {bucket['5xx'] > 0 ? <span className="seg seg-5xx" style={{ flex: bucket['5xx'] }} /> : null}
          {bucket['4xx'] > 0 ? <span className="seg seg-4xx" style={{ flex: bucket['4xx'] }} /> : null}
          {bucket['3xx'] > 0 ? <span className="seg seg-3xx" style={{ flex: bucket['3xx'] }} /> : null}
          {bucket['2xx'] > 0 ? <span className="seg seg-2xx" style={{ flex: bucket['2xx'] }} /> : null}
        </div>
      </div>
      <span className="timeline-label">{bucket.label}</span>
    </div>
  )
})

/* ── Status code distribution ── */

const StatusPanel = memo(function StatusPanel({ metrics }: { metrics: OverviewMetrics }) {
  const total = useMemo(
    () => Object.values(metrics.status_counts).reduce((a, b) => a + b, 0) || 1,
    [metrics.status_counts],
  )

  return (
    <div className="panel chart-panel">
      <div className="panel-head">
        <h2>状态码分布</h2>
      </div>
      <div className="panel-body">
        {(['2xx', '3xx', '4xx', '5xx'] as const).map((k) => {
          const n = metrics.status_counts[k] ?? 0
          return <BarRow key={k} label={k} count={n} max={total} />
        })}
      </div>
    </div>
  )
})

/* ── Top hosts ── */

const HostPanel = memo(function HostPanel({ metrics }: { metrics: OverviewMetrics }) {
  const maxHost = Math.max(1, ...metrics.top_hosts.map((h) => h.count))

  return (
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
                <div className="bar-fill seg-2xx" style={{ width: `${(h.count / maxHost) * 100}%` }} />
              </div>
              <span className="bar-val">{h.count}</span>
            </div>
          ))
        )}
      </div>
    </div>
  )
})

/* ── Slowest requests ── */

const SlowestPanel = memo(function SlowestPanel({ metrics }: { metrics: OverviewMetrics }) {
  return (
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
  )
})

/* ── Shared bar row ── */

const BarRow = memo(function BarRow({
  label,
  count,
  max,
}: {
  label: string
  count: number
  max: number
}) {
  return (
    <div className="bar-row">
      <span className={`bar-label seg-${label}`}>{label}</span>
      <div className="bar-track">
        <div className={`bar-fill seg-${label}`} style={{ width: `${(count / max) * 100}%` }} />
      </div>
      <span className="bar-val">{count}</span>
    </div>
  )
})

/* ── Utilities ── */

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
    case 'access_log_parse_fail':
      return '解析失败'
    case 'unconfigured':
      return '未配置'
    case 'error':
      return '读取失败'
    default:
      return source
  }
}
