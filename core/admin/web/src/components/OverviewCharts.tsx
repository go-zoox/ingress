import { memo } from 'react'
import { Link } from 'react-router-dom'
import type { OverviewMetrics } from '../api/client'
import { TrafficTimelineChart } from './charts/TrafficTimelineChart'
import { QualityTimelineChart } from './charts/QualityTimelineChart'
import { StatusDonut } from './charts/StatusDonut'

type Props = {
  metrics: OverviewMetrics | null
  loading?: boolean
  healthScore?: number | null
  healthClass?: 'ok' | 'warn' | 'danger'
}

export const OverviewCharts = memo(function OverviewCharts({
  metrics,
  loading,
  healthScore,
  healthClass = 'ok',
}: Props) {
  if (loading && metrics === null) {
    return <p style={{ color: 'var(--text-muted)', margin: '0 0 16px' }}>加载请求指标…</p>
  }
  if (!metrics || metrics.total === 0) {
    return (
      <div className="panel metrics-empty">
        <MetricsEmptyMessage metrics={metrics} />
      </div>
    )
  }

  return (
    <>
      <div className="cards overview-kpi-strip">
        <KpiCard
          label="健康度"
          value={healthScore != null ? String(healthScore) : '—'}
          sub="综合评分"
          tone={healthClass}
        />
        <KpiCard
          label="请求量"
          value={String(metrics.total)}
          sub={`≈ ${metrics.rpm.toFixed(1)} 次/分 · ${metrics.window}`}
        />
        <KpiCard
          label="错误率"
          value={`${metrics.error_rate.toFixed(1)}%`}
          sub="4xx + 5xx"
          tone={metrics.error_rate > 10 ? 'danger' : metrics.error_rate > 5 ? 'warn' : undefined}
        />
        <KpiCard
          label="P95 延迟"
          value={formatMs(metrics.p95_ms)}
          sub={`P50 ${formatMs(metrics.p50_ms)}`}
          tone={metrics.p95_ms > 2000 ? 'warn' : undefined}
        />
        <KpiCard
          label="缓存命中"
          value={`${metrics.cache_hit_rate.toFixed(0)}%`}
          sub="cache_hit=1"
        />
        <KpiCard
          label="WAF 拦截"
          value={String(metrics.waf_blocks)}
          sub="waf_block=1"
          tone={metrics.waf_blocks > 0 ? 'warn' : undefined}
        />
      </div>

      <div className="charts-grid charts-grid-2">
        <div className="panel chart-panel">
          <div className="panel-head">
            <h2>流量趋势</h2>
            <span className="chart-hint">按时间桶 · 堆叠状态码</span>
          </div>
          <div className="panel-body">
            <TrafficTimelineChart timeline={metrics.timeline} />
          </div>
        </div>
        <div className="panel chart-panel">
          <div className="panel-head">
            <h2>质量趋势</h2>
            <span className="chart-hint">错误率 % · WAF 拦截次数</span>
          </div>
          <div className="panel-body">
            <QualityTimelineChart timeline={metrics.timeline} />
          </div>
        </div>
      </div>

      <div className="charts-grid charts-grid-3">
        <div className="panel chart-panel">
          <div className="panel-head">
            <h2>状态码分布</h2>
          </div>
          <div className="panel-body">
            <StatusDonut counts={metrics.status_counts} />
          </div>
        </div>
        <div className="panel chart-panel">
          <div className="panel-head">
            <h2>Top Host（请求量）</h2>
          </div>
          <div className="panel-body">
            <HostBarList
              rows={metrics.top_hosts.map((h) => ({
                name: h.name,
                value: h.count,
                sub: `${h.count} 次`,
              }))}
              tone="ok"
            />
          </div>
        </div>
        <div className="panel chart-panel">
          <div className="panel-head">
            <h2>Top Host（错误率）</h2>
          </div>
          <div className="panel-body">
            <HostBarList
              rows={(metrics.top_hosts_error ?? []).map((h) => ({
                name: h.name,
                value: h.error_rate,
                sub: `${h.errors}/${h.count} · ${h.error_rate.toFixed(1)}%`,
              }))}
              tone="warn"
              maxValue={100}
            />
          </div>
        </div>
      </div>

      <div className="panel chart-panel">
        <div className="panel-head">
          <h2>最慢请求</h2>
          <Link to="/logs" className="btn btn-ghost btn-sm">
            访问日志
          </Link>
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
    </>
  )
})

const KpiCard = memo(function KpiCard({
  label,
  value,
  sub,
  tone,
}: {
  label: string
  value: string
  sub: string
  tone?: 'ok' | 'warn' | 'danger'
}) {
  const cls = tone ? `card ${tone}` : 'card'
  return (
    <div className={cls}>
      <div className="label">{label}</div>
      <div className="value">{value}</div>
      <div className="sub">{sub}</div>
    </div>
  )
})

const HostBarList = memo(function HostBarList({
  rows,
  tone,
  maxValue,
}: {
  rows: Array<{ name: string; value: number; sub: string }>
  tone: 'ok' | 'warn'
  maxValue?: number
}) {
  if (rows.length === 0) {
    return <p className="empty-hint">无数据</p>
  }
  const max = maxValue ?? Math.max(1, ...rows.map((r) => r.value))
  const fillClass = tone === 'ok' ? 'seg-2xx' : 'seg-4xx'
  return (
    <>
      {rows.map((h) => (
        <div key={h.name} className="bar-row host-rank">
          <span className="bar-label host-label" title={h.name}>
            {h.name}
          </span>
          <div className="bar-track">
            <div className={`bar-fill ${fillClass}`} style={{ width: `${(h.value / max) * 100}%` }} />
          </div>
          <span className="bar-val">{h.sub}</span>
        </div>
      ))}
    </>
  )
})

function MetricsEmptyMessage({ metrics }: { metrics: OverviewMetrics | null }) {
  if (metrics?.source === 'access_log_empty') {
    return (
      <p className="empty-hint">
        日志文件已配置但尚无内容。等待请求产生后数据会自动刷新。
      </p>
    )
  }
  if (metrics?.source === 'unconfigured') {
    return (
      <p className="empty-hint">
        未配置日志路径。请在 <code>ingress.yaml</code> 的 <code>logging</code> 段配置文件输出，或启用{' '}
        <code>admin.enabled</code>（默认在同目录生成 <code>access.log</code>）。
      </p>
    )
  }
  if (metrics?.source === 'error') {
    return <p className="empty-hint">读取日志文件失败。请检查日志路径是否正确。</p>
  }
  if (metrics?.source === 'access_log_parse_fail') {
    return <p className="empty-hint">日志文件有内容但无法解析。请检查日志格式是否兼容。</p>
  }
  return <p className="empty-hint">暂无访问日志数据。等待请求产生后数据会自动刷新。</p>
}

function formatMs(ms: number) {
  if (ms >= 1000) return `${(ms / 1000).toFixed(2)}s`
  return `${Math.round(ms)}ms`
}
