import { memo, useMemo } from 'react'
import type { OverviewMetrics } from '../../api/client'
import { readChartColors } from './chartTheme'
import { useUPlot } from './useUPlot'
import type { AlignedData } from 'uplot'
import type { UPlotOptions } from './useUPlot'

type Props = {
  histogram: OverviewMetrics['latency_histogram']
}

export const LatencyHistogramChart = memo(function LatencyHistogramChart({ histogram }: Props) {
  const colors = useMemo(() => readChartColors(), [])
  const labels = histogram.map((b) => b.label)

  const data = useMemo(() => {
    const xs = histogram.map((_, i) => i)
    return [xs, histogram.map((b) => b.count)] as AlignedData
  }, [histogram])

  const opts = useMemo((): UPlotOptions => {
    const c = colors
    return {
      cursor: { drag: { x: false, y: false } },
      legend: { show: false },
      scales: { x: { time: false }, y: { auto: true } },
      axes: [
        {
          stroke: c.muted,
          grid: { show: false },
          ticks: { show: false },
          values: (_u, ticks) => ticks.map((v) => labels[v] ?? ''),
        },
        { stroke: c.muted, grid: { stroke: c.grid }, ticks: { stroke: c.grid } },
      ],
      series: [{}, { label: '请求数', stroke: c.accent, fill: c.accent, width: 0 }],
    }
  }, [colors, labels])

  const ref = useUPlot(opts, data, 180)
  return <div className="uplot-wrap" ref={ref} />
})
