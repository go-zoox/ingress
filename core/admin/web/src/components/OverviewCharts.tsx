import { memo, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import {
  Activity,
  Gauge,
  HardDrive,
  Shield,
  Timer,
  Heart,
  Zap,
} from 'lucide-react'
import type { OverviewMetrics, HealthCheckResult, HealthSummary, TLSCert } from '../api/client'
import { formatMetricsWindowLabel, snapshotMatchesWindow } from '../lib/metricsWindow'
import { TrafficTimelineChart } from './charts/TrafficTimelineChart'
import { QualityTimelineChart } from './charts/QualityTimelineChart'
import { CacheTimelineChart } from './charts/CacheTimelineChart'
import { LatencyHistogramChart } from './charts/LatencyHistogramChart'
import { UpstreamLatencyTrendChart } from './charts/UpstreamLatencyTrendChart'
import { BackendPerformancePanel } from './BackendPerformancePanel'
import { StatusDonut } from './charts/StatusDonut'
import { OverviewDelta } from './OverviewDelta'
import { OverviewHealthMatrix } from './OverviewHealthMatrix'
import { OverviewTLSPanel } from './OverviewTLSPanel'
import { HostTrafficTable } from './HostTrafficTable'
import { RankedBarList } from './RankedBarList'

type Props = {
  metrics: OverviewMetrics | null
  metricsWindow: string
  /** Throttled snapshot for charts/lists; KPI strip prefers live metrics for this window. */
  chartMetrics?: OverviewMetrics | null
  /** Initial load — no metrics yet. */
  loading?: boolean
  /** Window switch — dim chart body only (KPI stays live). */
  refreshing?: boolean
  healthScore?: number | null
  healthClass?: 'ok' | 'warn' | 'danger'
  healthChecks?: HealthCheckResult[]
  healthSummary?: HealthSummary
  certs?: TLSCert[]
}

export const OverviewCharts = memo(function OverviewCharts({
  metrics,
  metricsWindow,
  chartMetrics: chartMetricsProp,
  loading,
  refreshing,
  healthScore,
  healthClass = 'ok',
  healthChecks = [],
  healthSummary = { total: 0, up: 0, down: 0, unknown: 0 },
  certs = [],
}: Props) {
  if (loading && metrics === null && !chartMetricsProp) {
    return <p style={{ color: 'var(--text-muted)', margin: '0 0 16px' }}>加载请求指标…</p>
  }

  const liveMetrics = snapshotMatchesWindow(metrics, metricsWindow) ? metrics : null
  const displayMetrics = liveMetrics ?? chartMetricsProp
  const charts = liveMetrics ?? chartMetricsProp
  const windowLabel = formatMetricsWindowLabel(metricsWindow)
  const rangeHint = displayMetrics?.window_stale ? `${windowLabel} · 历史时段` : windowLabel

  if (!displayMetrics || displayMetrics.total === 0) {
    return (
      <div className="panel metrics-empty">
        <MetricsEmptyMessage metrics={displayMetrics ?? null} windowLabel={windowLabel} />
      </div>
    )
  }

  const delta = displayMetrics.delta ?? { has_previous: false }
  const rpm = displayMetrics.rpm ?? 0
  const errorRate = displayMetrics.error_rate ?? 0
  const cacheHitRate = displayMetrics.cache_hit_rate ?? 0
  const p50Ms = displayMetrics.p50_ms ?? 0
  const p95Ms = displayMetrics.p95_ms ?? 0
  const wafBlocks = displayMetrics.waf_blocks ?? 0
  const chartsReady = charts != null && snapshotMatchesWindow(charts, metricsWindow)
  const chartBodyKey = `${metricsWindow}:${charts?.timeline?.length ?? 0}`

  return (
    <div className="overview-charts-wrap">
      <div className="cards overview-kpi-strip">
        <KpiCard
          icon={<Heart size={18} />}
          label="健康度"
          value={healthScore != null ? String(healthScore) : '—'}
          sub="综合评分"
          tone={healthClass}
        />
        <KpiCard
          icon={<Activity size={18} />}
          label="请求量"
          value={String(displayMetrics.total)}
          sub={`≈ ${rpm.toFixed(1)} 次/分 · ${rangeHint}`}
          delta={<OverviewDelta delta={delta} kind="pct" value={delta.total_pct} />}
        />
        <KpiCard
          icon={<Gauge size={18} />}
          label="错误率"
          value={`${errorRate.toFixed(1)}%`}
          sub="4xx + 5xx"
          tone={errorRate > 10 ? 'danger' : errorRate > 5 ? 'warn' : undefined}
          delta={
            <OverviewDelta delta={delta} kind="pp" value={delta.error_rate_delta} badIfUp />
          }
        />
        <KpiCard
          icon={<Timer size={18} />}
          label="P95 延迟"
          value={formatMs(p95Ms)}
          sub={`P50 ${formatMs(p50Ms)}`}
          tone={p95Ms > 2000 ? 'warn' : undefined}
          delta={<OverviewDelta delta={delta} kind="ms" value={delta.p95_delta_ms} badIfUp />}
        />
        <KpiCard
          icon={<HardDrive size={18} />}
          label="缓存命中"
          value={`${cacheHitRate.toFixed(0)}%`}
          sub="cache_hit=1"
          delta={
            <OverviewDelta delta={delta} kind="pp" value={delta.cache_hit_delta} badIfUp={false} />
          }
        />
        <KpiCard
          icon={<Shield size={18} />}
          label="WAF 拦截"
          value={String(wafBlocks)}
          sub="waf_block=1"
          tone={wafBlocks > 0 ? 'warn' : undefined}
          delta={
            <OverviewDelta delta={delta} kind="count" value={delta.waf_blocks_delta} badIfUp />
          }
        />
      </div>

      <div
        key={chartBodyKey}
        className={refreshing ? 'overview-charts-body is-refreshing' : 'overview-charts-body'}
      >
        {!chartsReady ? (
          <p className="empty-hint overview-charts-loading">{refreshing ? `加载${windowLabel}数据…` : '暂无图表数据'}</p>
        ) : (
          <>
        <div className="charts-grid charts-grid-2">
          <ChartPanel title="流量趋势" hint={`${windowLabel} · 堆叠状态码`}>
            <TrafficTimelineChart timeline={charts!.timeline ?? []} />
          </ChartPanel>
          <ChartPanel title="质量趋势" hint="错误率 · WAF">
            <QualityTimelineChart timeline={charts.timeline ?? []} />
          </ChartPanel>
        </div>

        <div className="charts-grid charts-grid-2">
          <ChartPanel title="缓存命中趋势" hint="按时间桶 %">
            <CacheTimelineChart timeline={charts.timeline ?? []} />
          </ChartPanel>
          <ChartPanel title="延迟分布" hint="直方图">
            <LatencyHistogramChart histogram={charts.latency_histogram ?? []} />
          </ChartPanel>
        </div>

        <div className="charts-grid charts-grid-2">
          <ChartPanel title="上游延迟趋势" hint="upstream_response_time P95 · 不含缓存命中">
            <UpstreamLatencyTrendChart timeline={charts.timeline ?? []} />
          </ChartPanel>
          <ChartPanel title="后端性能" hint="按 target 聚合 · 不含 handler">
            <BackendPerformancePanel backends={charts.top_backends ?? []} />
          </ChartPanel>
        </div>

        <div className="charts-grid charts-grid-3">
          <ChartPanel title="状态码分布">
            <StatusDonut counts={charts.status_counts} />
          </ChartPanel>
          <ChartPanel title="Top Host（请求量）">
            <RankedBarList
              rows={(charts.top_hosts ?? []).map((h) => ({
                name: h.name,
                value: h.count,
                sub: `${h.count} 次`,
              }))}
              tone="ok"
            />
          </ChartPanel>
          <ChartPanel title="Top Host（错误率）">
            <RankedBarList
              rows={(charts.top_hosts_error ?? []).map((h) => ({
                name: h.name,
                value: h.error_rate,
                sub: `${h.errors}/${h.count} · ${h.error_rate.toFixed(1)}%`,
              }))}
              tone="warn"
              maxValue={100}
            />
          </ChartPanel>
        </div>

        {(charts.host_traffic?.length ?? 0) > 0 ? (
          <div className="panel chart-panel">
            <div className="panel-head">
              <h2>域名 PV / UV</h2>
              <span className="chart-hint">
                PV = 请求数 · UV = 独立 IP（优先 real_ip）· {windowLabel}
              </span>
            </div>
            <div className="panel-body panel-table-wrap">
              <HostTrafficTable rows={charts.host_traffic!} />
            </div>
          </div>
        ) : null}

        <div className="charts-grid charts-grid-2">
          <ChartPanel title="健康检查矩阵" hint={`${healthSummary.up}/${healthSummary.total} UP`}>
            <OverviewHealthMatrix checks={healthChecks} summary={healthSummary} />
          </ChartPanel>
          <ChartPanel title="TLS 证书有效期">
            <OverviewTLSPanel certs={certs} />
          </ChartPanel>
        </div>

        {(charts.top_paths?.length ?? 0) > 0 ? (
          <div className="panel chart-panel">
            <div className="panel-head">
              <h2>
                <Zap size={16} style={{ verticalAlign: 'text-bottom', marginRight: 6 }} />
                Top Path（请求量）
              </h2>
            </div>
            <div className="panel-body">
              <RankedBarList
                rows={(charts.top_paths ?? []).map((p) => ({
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
                {(charts.slowest ?? []).length === 0 ? (
                  <tr>
                    <td colSpan={4} className="empty-hint">
                      无延迟数据
                    </td>
                  </tr>
                ) : (
                  (charts.slowest ?? []).map((s, i) => (
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
        )}
      </div>
    </div>
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

function MetricsEmptyMessage({
  metrics,
  windowLabel,
}: {
  metrics: OverviewMetrics | null
  windowLabel?: string
}) {
  const skipped = metrics?.parse_skipped ?? 0
  const range = windowLabel ? `${windowLabel} ` : ''

  if (skipped > 0 && (metrics?.parseable_in_tail ?? 0) === 0) {
    return (
      <p className="empty-hint">
        部分 access.log 行无法解析，已跳过（本次扫描 {skipped} 行）。请检查日志格式或在「事件 →
        日志解析」中处理。
      </p>
    )
  }
  if (metrics?.source === 'access_log_empty') {
    return (
      <p className="empty-hint">
        日志文件已配置但为空。等待 ingress 产生访问记录后会自动刷新。
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
  return <p className="empty-hint">{range}暂无访问日志数据。等待请求产生后数据会自动刷新。</p>
}

function formatMs(ms: number) {
  if (ms >= 1000) return `${(ms / 1000).toFixed(2)}s`
  return `${Math.round(ms)}ms`
}
