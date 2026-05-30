import { memo, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import type { ServiceDetail, ServiceMetrics } from '../../api/client'
import { investigateLink, routeDetailLink } from '../../lib/deepLinks'
import { TrafficTimelineChart } from '../charts/TrafficTimelineChart'
import { QualityTimelineChart } from '../charts/QualityTimelineChart'
import { LatencyTrendChart } from '../charts/LatencyTrendChart'
import { LatencyHistogramChart } from '../charts/LatencyHistogramChart'
import { StatusDonut } from '../charts/StatusDonut'

type Props = {
  detail: ServiceDetail
  metrics: ServiceMetrics
}

export const ServiceDetailCharts = memo(function ServiceDetailCharts({ detail, metrics }: Props) {
  if (metrics.total === 0) {
    return (
      <div className="panel metrics-empty" style={{ marginTop: 16 }}>
        <p className="empty-hint">该时间窗口内暂无匹配此上游目标的访问日志，请切换窗口或确认路由已有流量。</p>
      </div>
    )
  }

  const timeline = metrics.timeline ?? []
  const overviewTimeline = timeline.map((b) => ({
    ...b,
    error_rate: b.error_rate ?? 0,
    cache_hit_rate: b.cache_hit_rate ?? 0,
    waf_blocks: b.waf_blocks ?? 0,
  }))
  const upstream = metrics.upstream
  const hasUpstream = upstream && upstream.samples > 0

  return (
    <>
      {metrics.compare && metrics.compare.site_rpm > 0 ? (
        <div className="panel route-compare-panel" style={{ marginTop: 16 }}>
          <div className="panel-head">
            <h2>与全站对比</h2>
            <span className="chart-hint">{metrics.window}</span>
          </div>
          <div className="panel-body route-compare-grid">
            <CompareStat label="服务流量占比" value={`${metrics.compare.service_share_pct.toFixed(1)}%`} />
            <CompareStat
              label="错误率 vs 全站"
              value={`${metrics.compare.error_rate_vs_site >= 0 ? '+' : ''}${metrics.compare.error_rate_vs_site.toFixed(1)} pp`}
              tone={metrics.compare.error_rate_vs_site > 2 ? 'danger' : undefined}
            />
            <CompareStat label="全站次/分" value={metrics.compare.site_rpm.toFixed(1)} />
            <CompareStat label="全站错误率" value={`${metrics.compare.site_error_rate.toFixed(1)}%`} />
          </div>
        </div>
      ) : null}

      <div className="charts-grid charts-grid-2" style={{ marginTop: 16 }}>
        <ChartPanel title="流量趋势" hint="堆叠状态码">
          <TrafficTimelineChart timeline={overviewTimeline} />
        </ChartPanel>
        <ChartPanel title="质量趋势" hint="错误率 · WAF">
          <QualityTimelineChart timeline={overviewTimeline} />
        </ChartPanel>
      </div>

      <div className="charts-grid charts-grid-2">
        <ChartPanel title="状态码分布">
          <StatusDonut counts={metrics.status_counts} />
        </ChartPanel>
        <ChartPanel title="延迟趋势" hint="P50 · P95">
          <LatencyTrendChart timeline={timeline} />
        </ChartPanel>
      </div>

      <div className="charts-grid charts-grid-2">
        <ChartPanel title="延迟分布" hint="直方图">
          <LatencyHistogramChart histogram={metrics.latency_histogram ?? []} />
        </ChartPanel>
        {hasUpstream ? (
          <ChartPanel title="上游延迟" hint="网关 vs 上游">
            <UpstreamLatencyPanel upstream={upstream!} />
          </ChartPanel>
        ) : (
          <ChartPanel title="Host Top" hint="按请求数">
            <NamedCountList rows={metrics.top_hosts ?? []} emptyHint="无 Host 数据" />
          </ChartPanel>
        )}
      </div>

      {(metrics.top_hosts?.length ?? 0) > 0 && hasUpstream ? (
        <div className="charts-grid charts-grid-2">
          <ChartPanel title="Host Top" hint="按请求数">
            <NamedCountList rows={metrics.top_hosts ?? []} emptyHint="无 Host 数据" />
          </ChartPanel>
          <ChartPanel title="Path Top" hint="按请求数">
            <NamedCountList rows={metrics.top_paths ?? []} emptyHint="无 Path 数据" />
          </ChartPanel>
        </div>
      ) : (metrics.top_paths?.length ?? 0) > 0 ? (
        <div className="panel chart-panel" style={{ marginTop: 16 }}>
          <div className="panel-head">
            <h2>Path Top</h2>
            <span className="chart-hint">按请求数 · {metrics.window}</span>
          </div>
          <div className="panel-body">
            <NamedCountList rows={metrics.top_paths ?? []} emptyHint="无 Path 数据" />
          </div>
        </div>
      ) : null}

      {detail.route_refs.length > 0 ? (
        <div className="panel chart-panel" style={{ marginTop: 16 }}>
          <div className="panel-head">
            <h2>引用路由</h2>
            <span className="chart-hint">{detail.route_ref_count} 条</span>
          </div>
          <div className="panel-body panel-table-wrap">
            <table className="data compact">
              <thead>
                <tr>
                  <th>Host</th>
                  <th>Path</th>
                  <th>目标</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {detail.route_refs.map((ref) => (
                  <tr key={`${ref.rule_index}-${ref.path_index}`}>
                    <td><code>{ref.host}</code></td>
                    <td><code>{ref.path}</code></td>
                    <td><code>{ref.target}</code></td>
                    <td>
                      <Link
                        to={routeDetailLink(ref.rule_index, ref.path_index)}
                        className="btn btn-ghost btn-sm"
                      >
                        路由详情
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      ) : null}

      <div className="charts-grid charts-grid-2">
        <SampleTable title="最慢请求" rows={metrics.slowest ?? []} emptyHint="无延迟数据" />
        <SampleTable title="近期错误请求" rows={metrics.error_samples ?? []} emptyHint="无 4xx/5xx 样本" />
      </div>
    </>
  )
})

function CompareStat({
  label,
  value,
  tone,
}: {
  label: string
  value: string
  tone?: 'danger'
}) {
  return (
    <div className="route-compare-stat">
      <div className="label">{label}</div>
      <div className={`value${tone === 'danger' ? ' text-danger' : ''}`}>{value}</div>
    </div>
  )
}

function UpstreamLatencyPanel({ upstream }: { upstream: NonNullable<ServiceMetrics['upstream']> }) {
  const total = upstream.avg_total_ms || 1
  const upPct = Math.min(100, (upstream.avg_upstream_ms / total) * 100)
  const gwPct = Math.min(100, (upstream.avg_gateway_ms / total) * 100)
  return (
    <div className="upstream-latency-panel">
      <div className="upstream-latency-row">
        <span>平均总耗时</span>
        <strong>{formatMs(upstream.avg_total_ms)}</strong>
      </div>
      <div className="upstream-bar-stack" title="上游 / 网关">
        <span className="upstream-bar upstream-bar-up" style={{ width: `${upPct}%` }} />
        <span className="upstream-bar upstream-bar-gw" style={{ width: `${gwPct}%` }} />
      </div>
      <div className="upstream-latency-row">
        <span>上游</span>
        <span>{formatMs(upstream.avg_upstream_ms)}</span>
      </div>
      <div className="upstream-latency-row">
        <span>网关</span>
        <span>{formatMs(upstream.avg_gateway_ms)}</span>
      </div>
      <div className="upstream-latency-row muted">
        <span>上游 5xx 占比</span>
        <span>{upstream.upstream_error_pct.toFixed(1)}%</span>
      </div>
      <p className="empty-hint" style={{ marginTop: 8 }}>
        样本 {upstream.samples} 条
      </p>
    </div>
  )
}

function NamedCountList({
  rows,
  emptyHint,
}: {
  rows: Array<{ name: string; count: number }>
  emptyHint: string
}) {
  if (rows.length === 0) {
    return <p className="empty-hint">{emptyHint}</p>
  }
  const max = Math.max(1, ...rows.map((r) => r.count))
  return (
    <ul className="bar-list compact">
      {rows.map((r) => (
        <li key={r.name}>
          <span className="bar-label" title={r.name}>
            {r.name}
          </span>
          <span className="bar-track">
            <span className="bar-fill" style={{ width: `${(r.count / max) * 100}%` }} />
          </span>
          <span className="bar-value">{r.count}</span>
        </li>
      ))}
    </ul>
  )
}

function ChartPanel({
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
}

function SampleTable({
  title,
  rows,
  emptyHint,
}: {
  title: string
  rows: Array<{
    host: string
    method: string
    path: string
    status: number
    duration_ms: number
  }>
  emptyHint: string
}) {
  return (
    <div className="panel chart-panel">
      <div className="panel-head">
        <h2>{title}</h2>
      </div>
      <div className="panel-body panel-table-wrap">
        <table className="data compact">
          <thead>
            <tr>
              <th>Host</th>
              <th>请求</th>
              <th>状态</th>
              <th>耗时</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {rows.length === 0 ? (
              <tr>
                <td colSpan={5} className="empty-hint">
                  {emptyHint}
                </td>
              </tr>
            ) : (
              rows.map((s, i) => (
                <tr key={`${s.host}-${s.method}-${s.path}-${i}`}>
                  <td><code>{s.host}</code></td>
                  <td>
                    <code>
                      {s.method} {s.path}
                    </code>
                  </td>
                  <td>{s.status}</td>
                  <td>{formatMs(s.duration_ms)}</td>
                  <td>
                    <Link
                      to={investigateLink({ host: s.host, path: s.path })}
                      className="btn btn-ghost btn-sm"
                    >
                      调查
                    </Link>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function formatMs(ms: number) {
  if (ms >= 1000) return `${(ms / 1000).toFixed(2)}s`
  return `${Math.round(ms)}ms`
}
