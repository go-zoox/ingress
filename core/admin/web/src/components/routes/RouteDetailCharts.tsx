import { memo, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import type { RouteDetail, RouteMetrics } from '../../api/client'
import { investigateLink } from '../../lib/deepLinks'
import { TrafficTimelineChart } from '../charts/TrafficTimelineChart'
import { QualityTimelineChart } from '../charts/QualityTimelineChart'
import { LatencyTrendChart } from '../charts/LatencyTrendChart'
import { CacheTimelineChart } from '../charts/CacheTimelineChart'
import { LatencyHistogramChart } from '../charts/LatencyHistogramChart'
import { StatusDonut } from '../charts/StatusDonut'

type Props = {
  detail: RouteDetail
  metrics: RouteMetrics
  scopeHost?: string
  scopePath?: string
  pathMatch?: string
}

export const RouteDetailCharts = memo(function RouteDetailCharts({ detail, metrics, scopeHost, scopePath, pathMatch }: Props) {
  if (metrics.total === 0) {
    return (
      <div className="panel metrics-empty" style={{ marginTop: 16 }}>
        <p className="empty-hint">该时间窗口内暂无匹配的访问日志，请切换范围或等待流量进入。</p>
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

  const scopedQs = new URLSearchParams()
  if (scopeHost) scopedQs.set('host', scopeHost)
  if (scopePath) scopedQs.set('path', scopePath)
  if (pathMatch) scopedQs.set('path_match', pathMatch)
  const scopeQsString = scopedQs.toString()

  const showPathBreakdown =
    (metrics.path_breakdown?.length ?? 0) > 1 && detail.path_index < 0
  const showCacheTrend = (detail.cache?.enabled || metrics.route_cache?.enabled) && timeline.length > 0
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
            <CompareStat label="路由流量占比" value={`${metrics.compare.route_share_pct.toFixed(1)}%`} />
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
        {showCacheTrend ? (
          <ChartPanel title="缓存命中趋势" hint="按时间桶 %">
            <CacheTimelineChart timeline={overviewTimeline} />
          </ChartPanel>
        ) : (
          <ChartPanel title="延迟分布" hint="直方图">
            <LatencyHistogramChart histogram={metrics.latency_histogram ?? []} />
          </ChartPanel>
        )}
        {showCacheTrend ? (
          <ChartPanel title="延迟分布" hint="直方图">
            <LatencyHistogramChart histogram={metrics.latency_histogram ?? []} />
          </ChartPanel>
        ) : hasUpstream ? (
          <ChartPanel title="上游延迟" hint="网关 vs 上游">
            <UpstreamLatencyPanel upstream={upstream!} />
          </ChartPanel>
        ) : (
          <ChartPanel title="WAF 规则 Top" hint="DB 事件计数">
            <WAFTopRulesList rules={metrics.waf_top_rules ?? []} />
          </ChartPanel>
        )}
      </div>

      {hasUpstream && showCacheTrend ? (
        <div className="charts-grid charts-grid-2">
          <ChartPanel title="上游延迟" hint="网关 vs 上游">
            <UpstreamLatencyPanel upstream={upstream!} />
          </ChartPanel>
          <ChartPanel title="WAF 规则 Top" hint="DB 事件计数">
            <WAFTopRulesList rules={metrics.waf_top_rules ?? []} />
          </ChartPanel>
        </div>
      ) : null}

      {showPathBreakdown ? (
        <div className="panel chart-panel" style={{ marginTop: 0 }}>
          <div className="panel-head">
            <h2>路径拆分</h2>
            <span className="chart-hint">同 Host 下各 path</span>
          </div>
          <div className="panel-body panel-table-wrap">
            <table className="data compact">
              <thead>
                <tr>
                  <th>Path</th>
                  <th>请求数</th>
                  <th>错误率</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {metrics.path_breakdown!.map((row) => (
                  <tr key={`${row.path_index}-${row.path}`}>
                    <td>
                      <code>{row.path}</code>
                    </td>
                    <td>{row.count}</td>
                    <td>{row.error_rate.toFixed(1)}%</td>
                    <td>
                      {row.path_index >= 0 ? (
                        <Link
                          to={`/routes/${detail.rule_index}/${row.path_index}${scopeQsString ? `?${scopeQsString}` : ''}`}
                          className="btn btn-ghost btn-sm"
                        >
                          详情
                        </Link>
                      ) : null}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      ) : null}

      {(metrics.health_history?.length ?? 0) > 0 ? (
        <div className="panel chart-panel">
          <div className="panel-head">
            <h2>健康检查历史</h2>
            <span className="chart-hint">最近 {metrics.health_history!.length} 次探测</span>
          </div>
          <div className="panel-body">
            <HealthHistoryChart points={metrics.health_history!} />
          </div>
        </div>
      ) : null}

      {(metrics.related_routes?.length ?? 0) > 0 ? (
        <div className="panel chart-panel">
          <div className="panel-head">
            <h2>相关路由</h2>
          </div>
          <div className="panel-body related-routes-grid">
            {metrics.related_routes!.map((r) => (
              <Link
                key={`${r.rule_index}-${r.path_index}`}
                to={`/routes/${r.rule_index}/${r.path_index}${scopeQsString ? `?${scopeQsString}` : ''}`}
                className="related-route-card"
              >
                <span className="badge badge-audit">{relationLabel(r.relation)}</span>
                <code className="related-route-host">
                  {r.host}
                  {r.path}
                </code>
                <span className="related-route-target">{r.target}</span>
              </Link>
            ))}
          </div>
        </div>
      ) : null}

      <div className="charts-grid charts-grid-2">
        <SampleTable
          title="最慢请求"
          rows={metrics.slowest ?? []}
          detail={detail}
          scopeHost={scopeHost}
          emptyHint="无延迟数据"
        />
        <SampleTable
          title="近期错误请求"
          rows={metrics.error_samples ?? []}
          detail={detail}
          scopeHost={scopeHost}
          emptyHint="无 4xx/5xx 样本"
        />
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

function UpstreamLatencyPanel({ upstream }: { upstream: NonNullable<RouteMetrics['upstream']> }) {
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

function WAFTopRulesList({ rules }: { rules: Array<{ name: string; count: number }> }) {
  if (rules.length === 0) {
    return <p className="empty-hint">该窗口内无 WAF 事件</p>
  }
  const max = Math.max(1, ...rules.map((r) => r.count))
  return (
    <ul className="bar-list compact">
      {rules.map((r) => (
        <li key={r.name}>
          <span className="bar-label" title={r.name}>
            {r.name}
          </span>
          <span className="bar-track">
            <span className="bar-fill bar-fill-warn" style={{ width: `${(r.count / max) * 100}%` }} />
          </span>
          <span className="bar-value">{r.count}</span>
        </li>
      ))}
    </ul>
  )
}

function HealthHistoryChart({ points }: { points: import('../../api/client').HealthProbePoint[] }) {
  const maxMs = Math.max(1, ...points.map((p) => p.response_ms))
  return (
    <div className="health-history-chart">
      {points.map((p, i) => (
        <div
          key={`${p.at}-${i}`}
          className={`health-history-bar ${p.status === 'up' ? 'up' : p.status === 'down' ? 'down' : 'unknown'}`}
          style={{ height: `${Math.max(8, (p.response_ms / maxMs) * 100)}%` }}
          title={`${p.status} · ${formatMs(p.response_ms)} · ${p.at}`}
        />
      ))}
    </div>
  )
}

function relationLabel(rel: string) {
  switch (rel) {
    case 'same_backend':
      return '同后端'
    case 'same_host_suffix':
      return '同域后缀'
    default:
      return rel
  }
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
  detail,
  scopeHost,
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
  detail: RouteDetail
  scopeHost?: string
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
              <th>请求</th>
              <th>状态</th>
              <th>耗时</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {rows.length === 0 ? (
              <tr>
                <td colSpan={4} className="empty-hint">
                  {emptyHint}
                </td>
              </tr>
            ) : (
              rows.map((s, i) => (
                <tr key={`${s.method}-${s.path}-${i}`}>
                  <td>
                    <code>
                      {s.method} {s.path}
                    </code>
                  </td>
                  <td>{s.status}</td>
                  <td>{formatMs(s.duration_ms)}</td>
                  <td>
                    <Link
                      to={investigateLink({
                        host: scopeHost || detail.host,
                        path: s.path,
                        ri: detail.rule_index,
                        pi: detail.path_index,
                      })}
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
