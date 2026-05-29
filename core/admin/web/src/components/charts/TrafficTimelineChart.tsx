import { memo, useMemo, useRef } from 'react'
import type { OverviewMetrics } from '../../api/client'
import { readChartColors } from './chartTheme'
import { useUPlot } from './useUPlot'
import type { AlignedData } from 'uplot'
import type { UPlotOptions } from './useUPlot'

type Props = {
  timeline: OverviewMetrics['timeline']
}

export const TrafficTimelineChart = memo(function TrafficTimelineChart({ timeline }: Props) {
  const colors = useMemo(() => readChartColors(), [])
  const labels = timeline.map((b) => b.label)

  const data = useMemo(() => {
    const xs = timeline.map((_, i) => i)
    return [
      xs,
      timeline.map((b) => b['2xx']),
      timeline.map((b) => b['3xx']),
      timeline.map((b) => b['4xx']),
      timeline.map((b) => b['5xx']),
    ] as AlignedData
  }, [timeline])

  const labelsRef = useRef(labels)
  labelsRef.current = labels

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
          values: (_u, ticks) => ticks.map((v) => labelsRef.current[v] ?? ''),
        },
        { stroke: c.muted, grid: { stroke: c.grid }, ticks: { stroke: c.grid } },
      ],
      series: [
        {},
        { label: '2xx', stroke: c.ok, fill: c.ok, width: 0 },
        { label: '3xx', stroke: c.accent, fill: c.accent, width: 0 },
        { label: '4xx', stroke: c.warn, fill: c.warn, width: 0 },
        { label: '5xx', stroke: c.danger, fill: c.danger, width: 0 },
      ],
      bands: [
        { series: [1, 2] },
        { series: [2, 3] },
        { series: [3, 4] },
      ],
    }
  }, [colors])

  const ref = useUPlot(opts, data, 200)
  return <div className="uplot-wrap" ref={ref} />
})
