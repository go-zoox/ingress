import { memo, useMemo } from 'react'
import type { MetricsTimelineBucket } from '../../api/client'
import { readChartColors } from './chartTheme'
import { useUPlot } from './useUPlot'
import type { AlignedData } from 'uplot'
import type { UPlotOptions } from './useUPlot'

type Props = {
  timeline: MetricsTimelineBucket[]
}

export const UpstreamLatencyTrendChart = memo(function UpstreamLatencyTrendChart({ timeline }: Props) {
  const colors = useMemo(() => readChartColors(), [])
  const labels = timeline.map((b) => b.label)
  const hasData = timeline.some((b) => (b.upstream_p95_ms ?? 0) > 0)

  const data = useMemo(() => {
    const xs = timeline.map((_, i) => i)
    return [xs, timeline.map((b) => b.upstream_p95_ms ?? 0)] as AlignedData
  }, [timeline])

  const opts = useMemo((): UPlotOptions => {
    const c = colors
    return {
      cursor: { drag: { x: false, y: false } },
      legend: { show: true },
      scales: { x: { time: false }, y: { auto: true } },
      axes: [
        {
          stroke: c.muted,
          grid: { show: false },
          ticks: { show: false },
          values: (_u, ticks) => ticks.map((v) => labels[v] ?? ''),
        },
        {
          stroke: c.muted,
          grid: { stroke: c.grid },
          ticks: { stroke: c.grid },
          values: (_u, ticks) => ticks.map((v) => (v >= 1000 ? `${(v / 1000).toFixed(1)}s` : `${v}ms`)),
        },
      ],
      series: [
        {},
        { label: '上游 P95', stroke: c.warn, width: 2, fill: c.warn },
      ],
    }
  }, [colors, labels])

  const ref = useUPlot(opts, data, 200)

  if (!hasData) {
    return <p className="empty-hint">暂无上游耗时数据（需 access log 含 upstream_response_time）</p>
  }

  return <div className="uplot-wrap" ref={ref} />
})
