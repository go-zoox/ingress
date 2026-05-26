import { memo, useMemo } from 'react'
import type { OverviewMetrics } from '../../api/client'
import { readChartColors } from './chartTheme'
import { useUPlot } from './useUPlot'
import type { AlignedData } from 'uplot'
import type { UPlotOptions } from './useUPlot'

type Props = {
  timeline: OverviewMetrics['timeline']
}

export const CacheTimelineChart = memo(function CacheTimelineChart({ timeline }: Props) {
  const colors = useMemo(() => readChartColors(), [])
  const labels = timeline.map((b) => b.label)

  const data = useMemo(() => {
    const xs = timeline.map((_, i) => i)
    return [xs, timeline.map((b) => b.cache_hit_rate)] as AlignedData
  }, [timeline])

  const opts = useMemo((): UPlotOptions => {
    const c = colors
    return {
      cursor: { drag: { x: false, y: false } },
      legend: { show: true },
      scales: { x: { time: false }, y: { auto: true, range: [0, 100] } },
      axes: [
        {
          stroke: c.muted,
          grid: { show: false },
          ticks: { show: false },
          values: (_u, ticks) => ticks.map((v) => labels[v] ?? ''),
        },
        {
          stroke: c.ok,
          grid: { stroke: c.grid },
          ticks: { stroke: c.grid },
          label: '命中率 %',
        },
      ],
      series: [{}, { label: '缓存命中 %', stroke: c.ok, fill: c.ok + '44', width: 2 }],
    }
  }, [colors, labels])

  const ref = useUPlot(opts, data, 180)
  return <div className="uplot-wrap" ref={ref} />
})
