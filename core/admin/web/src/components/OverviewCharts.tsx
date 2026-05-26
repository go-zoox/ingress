import { memo, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import {
  Activity,
  Gauge,
  HardDrive,
  Shield,
  Timer,
  Zap,
  Heart,
} from 'lucide-react'
import type { OverviewMetrics, HealthCheckResult, HealthSummary, TLSCert } from '../api/client'
import { TrafficTimelineChart } from './charts/TrafficTimelineChart'
import { QualityTimelineChart } from './charts/QualityTimelineChart'
import { QPSTimelineChart } from './charts/QPSTimelineChart'
import { CacheTimelineChart } from './charts/CacheTimelineChart'
import { LatencyHistogramChart } from './charts/LatencyHistogramChart'
import { StatusDonut } from './charts/StatusDonut'
import { OverviewDelta } from './OverviewDelta'
import { OverviewHealthMatrix } from './OverviewHealthMatrix'
import { OverviewTLSPanel } from './OverviewTLSPanel'

type Props = {
  metrics: OverviewMetrics | null
  loading?: boolean
  healthScore?: number | null
  healthClass?: 'ok' | 'warn' | 'danger'
  healthChecks?: HealthCheckResult[]
  healthSummary?: HealthSummary
  certs?: TLSCert[]
}

export const OverviewCharts = memo(function OverviewCharts({
  metrics,
  loading,
  healthScore,
  healthClass = 'ok',
  healthChecks = [],
  healthSummary = { total: 0, up: 0, down: 0, unknown: 0 },
  certs = [],
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

  const delta = metrics.delta ?? { has_previous: false }

  return (
    <>
      <div className="cards overview-kpi-strip">
        <KpiCard
          icon={<Heart size={18} />}
          label="健康度"
          value={healthScore != null ? String(healthScore) : '—'}
          sub="综合评分"
          tone={healthClass}
        />
        <KpiCard
          icon={<Zap size={18} />}
          label="QPS"
          value={formatQPS(metrics.qps ?? metrics.rpm / 60)}
          sub={`≈ ${metrics.rpm.toFixed(1)} 次/分 · ${metrics.window}`}
          delta={<OverviewDelta delta={delta} kind="pct" value={delta.rpm_pct ?? delta.total_pct} />}
        />
        <KpiCard
          icon={<Activity size={18} />}
          label="请求量"
          value={String(metrics.total)}
          sub={`${metrics.window}`}
          delta={<OverviewDelta delta={delta} kind="pct" value={delta.total_pct} />}
        />
        <KpiCard
          icon={<Gauge size={18} />}
          label="错误率"
          value={`${metrics.error_rate.toFixed(1)}%`}
          sub="4xx + 5xx"
          tone={metrics.error_rate > 10 ? 'danger' : metrics.error_rate > 5 ? 'warn' : undefined}
          delta={
            <OverviewDelta delta={delta} kind="pp" value={delta.error_rate_delta} badIfUp />
          }
        />
        <KpiCard
          icon={<Timer size={18} />}
          label="P95 延迟"
          value={formatMs(metrics.p95_ms)}
          sub={`P50 ${formatMs(metrics.p50_ms)}`}
          tone={metrics.p95_ms > 2000 ? 'warn' : undefined}
          delta={<OverviewDelta delta={delta} kind="ms" value={delta.p95_delta_ms} badIfUp />}
        />
        <KpiCard
          icon={<HardDrive size={18} />}
          label="缓存命中"
          value={`${metrics.cache_hit_rate.toFixed(0)}%`}
          sub="cache_hit=1"
          delta={
            <OverviewDelta delta={delta} kind="pp" value={delta.cache_hit_delta} badIfUp={false} />
          }
        />
        <KpiCard
          icon={<Shield size={18} />}
          label="WAF 拦截"
          value={String(metrics.waf_blocks)}
          sub="waf_block=1"
          tone={metrics.waf_blocks > 0 ? 'warn' : undefined}
          delta={
            <OverviewDelta delta={delta} kind="count" value={delta.waf_blocks_delta} badIfUp />
          }
        />
      </div>

      <div className="charts-grid charts-grid-2">
        <ChartPanel title="QPS 趋势" hint="每时间桶 req/s">
          <QPSTimelineChart timeline={metrics.timeline} />
        </ChartPanel>
        <ChartPanel title="流量趋势" hint="堆叠状态码">
          <TrafficTimelineChart timeline={metrics.timeline} />
        </ChartPanel>
      </div>

      <div className="charts-grid charts-grid-2">
        <ChartPanel title="质量趋势" hint="错误率 · WAF">
          <QualityTimelineChart timeline={metrics.timeline} />
        </ChartPanel>
        <ChartPanel title="缓存命中趋势" hint="按时间桶 %">
          <CacheTimelineChart timeline={metrics.timeline} />
        </ChartPanel>
      </div>

      <div className="charts-grid charts-grid-2">
        <ChartPanel title="延迟分布" hint="直方图">
          <LatencyHistogramChart histogram={metrics.latency_histogram ?? []} />
        </ChartPanel>
      </div>

      <div className="charts-grid charts-grid-3">
        <ChartPanel title="状态码分布">
          <StatusDonut counts={metrics.status_counts} />
        </ChartPanel>
        <ChartPanel title="Top Host（请求量）">
          <HostBarList
            rows={metrics.top_hosts.map((h) => ({
              name: h.name,
              value: h.count,
              sub: `${h.count} 次`,
            }))}
            tone="ok"
          />
        </ChartPanel>
        <ChartPanel title="Top Host（错误率）">
          <HostBarList
            rows={(metrics.top_hosts_error ?? []).map((h) => ({
              name: h.name,
              value: h.error_rate,
              sub: `${h.errors}/${h.count} · ${h.error_rate.toFixed(1)}%`,
            }))}
            tone="warn"
            maxValue={100}
          />
        </ChartPanel>
      </div>

      <div className="charts-grid charts-grid-2">
        <ChartPanel title="健康检查矩阵" hint={`${healthSummary.up}/${healthSummary.total} UP`}>
          <OverviewHealthMatrix checks={healthChecks} summary={healthSummary} />
        </ChartPanel>
        <ChartPanel title="TLS 证书有效期">
          <OverviewTLSPanel certs={certs} />
        </ChartPanel>
      </div>

      {(metrics.top_paths?.length ?? 0) > 0 ? (
        <div className="panel chart-panel">
          <div className="panel-head">
            <h2>
              <Zap size={16} style={{ verticalAlign: 'text-bottom', marginRight: 6 }} />
              Top Path（请求量）
            </h2>
          </div>
          <div className="panel-body">
            <HostBarList
              rows={(metrics.top_paths ?? []).map((p) => ({
                name: p.name,
                value: p.count,
                sub: String(p.count),
              }))}
              tone="ok"
            />
          </div>
        </div>
      ) : null}

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

const ChartPanel = memo(function ChartPanel({
  title,
  hint,
  children,
}: {
  title: string
  hint?: string
  children: ReactNode
}) {
  return (
    <div className="panel chart-panel">
      <div className="panel-head">
        <h2>{title}</h2>
        {hint ? <span className="chart-hint">{hint}</span> : null}
      </div>
      <div className="panel-body">{children}</div>
    </div>
  )
})

const KpiCard = memo(function KpiCard({
  icon,
  label,
  value,
  sub,
  tone,
  delta,
}: {
  icon: ReactNode
  label: string
  value: string
  sub: string
  tone?: 'ok' | 'warn' | 'danger'
  delta?: ReactNode
}) {
  const cls = tone ? `card kpi-card ${tone}` : 'card kpi-card'
  return (
    <div className={cls}>
      <div className="kpi-card-top">
        <span className="kpi-icon">{icon}</span>
        <span className="label">{label}</span>
      </div>
      <div className="value">{value}</div>
      <div className="sub">{sub}</div>
      {delta ? <div className="kpi-delta-row">{delta}</div> : null}
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
        <code>admin.enabled</code>。
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

function formatQPS(qps: number) {
  if (qps >= 100) return qps.toFixed(0)
  if (qps >= 10) return qps.toFixed(1)
  if (qps >= 1) return qps.toFixed(2)
  return qps.toFixed(3)
}
