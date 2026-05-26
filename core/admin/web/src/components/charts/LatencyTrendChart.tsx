import { memo, useMemo } from 'react'
import type { MetricsTimelineBucket } from '../../api/client'
import { readChartColors } from './chartTheme'
import { useUPlot } from './useUPlot'
import type { AlignedData } from 'uplot'
import type { UPlotOptions } from './useUPlot'

type Props = {
  timeline: MetricsTimelineBucket[]
}

export const LatencyTrendChart = memo(function LatencyTrendChart({ timeline }: Props) {
  const colors = useMemo(() => readChartColors(), [])
  const labels = timeline.map((b) => b.label)

  const data = useMemo(() => {
    const xs = timeline.map((_, i) => i)
    return [
      xs,
      timeline.map((b) => b.p50_ms ?? 0),
      timeline.map((b) => b.p95_ms ?? 0),
    ] as AlignedData
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
        { label: 'P50', stroke: c.accent, width: 2 },
        { label: 'P95', stroke: c.warn, width: 2 },
      ],
    }
  }, [colors, labels])

  const ref = useUPlot(opts, data, 200)
  return <div className="uplot-wrap" ref={ref} />
})
