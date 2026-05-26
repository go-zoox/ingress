import { memo, useMemo } from 'react'
import type { MetricsTimelineBucket } from '../../api/client'
import { readChartColors } from './chartTheme'
import { useUPlot } from './useUPlot'
import type { AlignedData } from 'uplot'
import type { UPlotOptions } from './useUPlot'

type Props = {
  timeline: MetricsTimelineBucket[]
}

function formatQPS(v: number) {
  if (v >= 100) return v.toFixed(0)
  if (v >= 10) return v.toFixed(1)
  if (v >= 1) return v.toFixed(2)
  return v.toFixed(3)
}

export const QPSTimelineChart = memo(function QPSTimelineChart({ timeline }: Props) {
  const colors = useMemo(() => readChartColors(), [])
  const labels = timeline.map((b) => b.label)

  const data = useMemo(() => {
    const xs = timeline.map((_, i) => i)
    return [xs, timeline.map((b) => b.qps ?? 0)] as AlignedData
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
          values: (_u, ticks) => ticks.map((v) => formatQPS(v)),
        },
      ],
      series: [
        {},
        { label: 'QPS', stroke: c.accent, width: 2, fill: c.accent },
      ],
    }
  }, [colors, labels])

  const ref = useUPlot(opts, data, 200)
  return <div className="uplot-wrap" ref={ref} />
})
