import { memo, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import type { RouteDetail, RouteMetrics } from '../../api/client'
import { investigateLink } from '../../lib/deepLinks'
import { TrafficTimelineChart } from '../charts/TrafficTimelineChart'
import { QualityTimelineChart } from '../charts/QualityTimelineChart'
import { LatencyTrendChart } from '../charts/LatencyTrendChart'
import { StatusDonut } from '../charts/StatusDonut'

type Props = {
  detail: RouteDetail
  metrics: RouteMetrics
}

export const RouteDetailCharts = memo(function RouteDetailCharts({ detail, metrics }: Props) {
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

  return (
    <>
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
        <SampleTable
          title="最慢请求"
          rows={metrics.slowest ?? []}
          detail={detail}
          emptyHint="无延迟数据"
        />
        <SampleTable
          title="近期错误请求"
          rows={metrics.error_samples ?? []}
          detail={detail}
          emptyHint="无 4xx/5xx 样本"
        />
      </div>
    </>
  )
})

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
                        host: detail.host,
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
