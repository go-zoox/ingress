import { memo, useEffect, useMemo, useState } from 'react'
import type { OverviewMetrics } from '../api/client'
import {
  filterHostTimeline,
  mergeOverviewHostNames,
  overviewHostFilterSummary,
} from '../lib/overviewHostFilter'
import { HostTrafficTimelineChart } from './charts/HostTrafficTimelineChart'

type Props = {
  metrics: OverviewMetrics
  windowLabel: string
}

export const OverviewHostTimelinePanel = memo(function OverviewHostTimelinePanel({
  metrics,
  windowLabel,
}: Props) {
  const series = metrics.host_timeline ?? []
  const hostOptions = useMemo(() => mergeOverviewHostNames(metrics), [metrics])
  const [selectedHosts, setSelectedHosts] = useState<string[]>([])

  useEffect(() => {
    setSelectedHosts((prev) => {
      if (prev.length === 0) return prev
      const allowed = new Set(hostOptions)
      const next = prev.filter((h) => allowed.has(h))
      return next.length === prev.length ? prev : next
    })
  }, [hostOptions])

  const filteredSeries = useMemo(
    () => filterHostTimeline(series, selectedHosts),
    [series, selectedHosts],
  )

  const summary = overviewHostFilterSummary(selectedHosts, hostOptions.length)
  const isAll = selectedHosts.length === 0

  const toggleHost = (host: string, checked: boolean) => {
    setSelectedHosts((prev) => {
      if (checked) {
        if (prev.includes(host)) return prev
        return [...prev, host]
      }
      return prev.filter((h) => h !== host)
    })
  }

  const selectAll = () => setSelectedHosts([])

  if (series.length === 0) return null

  return (
    <div className="panel chart-panel overview-host-timeline-panel">
      <div className="panel-head overview-host-timeline-head">
        <div className="overview-host-timeline-head-start">
          <details className="overview-host-filter form-multi-select">
            <summary className="form-control form-multi-select-summary overview-host-filter-summary">
              <span className="overview-host-filter-label">域名</span>
              <span className="overview-host-filter-value">{summary}</span>
            </summary>
            <div className="form-multi-select-panel overview-host-filter-panel">
              <label className="form-check form-check--inline">
                <input type="checkbox" checked={isAll} onChange={() => selectAll()} />
                <span>全部</span>
              </label>
              {hostOptions.map((host) => (
                <label key={host} className="form-check form-check--inline">
                  <input
                    type="checkbox"
                    checked={!isAll && selectedHosts.includes(host)}
                    onChange={(e) => {
                      if (e.target.checked) {
                        toggleHost(host, true)
                        return
                      }
                      toggleHost(host, false)
                    }}
                  />
                  <span>{host}</span>
                </label>
              ))}
            </div>
          </details>
          <h2>域名流量趋势</h2>
        </div>
        <span className="chart-hint">
          {windowLabel}
          {isAll ? ' · Top 16 + Other · 悬停联动' : ` · 已选 ${filteredSeries.length} 条 · 悬停联动`}
        </span>
      </div>
      <div className="panel-body">
        {filteredSeries.length > 0 ? (
          <HostTrafficTimelineChart series={filteredSeries} />
        ) : (
          <p className="empty-hint">所选域名在当前时间窗口暂无趋势数据（可能未进入 Top 序列）。</p>
        )}
      </div>
    </div>
  )
})
