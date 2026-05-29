import { memo, useMemo, useRef } from 'react'
import type { OverviewMetrics } from '../../api/client'
import { readChartColors } from './chartTheme'
import { useUPlot } from './useUPlot'
import type { AlignedData } from 'uplot'
import type { UPlotOptions } from './useUPlot'

type Props = {
  timeline: OverviewMetrics['timeline']
}

export const QualityTimelineChart = memo(function QualityTimelineChart({ timeline }: Props) {
  const colors = useMemo(() => readChartColors(), [])
  const labels = timeline.map((b) => b.label)

  const data = useMemo(() => {
    const xs = timeline.map((_, i) => i)
    return [
      xs,
      timeline.map((b) => b.error_rate),
      timeline.map((b) => b.waf_blocks),
    ] as AlignedData
  }, [timeline])

  const labelsRef = useRef(labels)
  labelsRef.current = labels

  const opts = useMemo((): UPlotOptions => {
    const c = colors
    return {
      cursor: { drag: { x: false, y: false } },
      legend: { show: true },
      scales: {
        x: { time: false },
        y: { auto: true },
        y2: { auto: true },
      },
      axes: [
        {
          stroke: c.muted,
          grid: { show: false },
          ticks: { show: false },
          values: (_u, ticks) => ticks.map((v) => labelsRef.current[v] ?? ''),
        },
        {
          scale: 'y',
          stroke: c.warn,
          grid: { stroke: c.grid },
          ticks: { stroke: c.grid },
          label: '错误率 %',
        },
        {
          scale: 'y2',
          side: 1,
          stroke: c.danger,
          grid: { show: false },
          ticks: { stroke: c.danger },
          label: 'WAF',
        },
      ],
      series: [
        {},
        { scale: 'y', label: '错误率 %', stroke: c.warn, width: 2 },
        { scale: 'y2', label: 'WAF 拦截', stroke: c.danger, fill: c.danger + '55', width: 1 },
      ],
    }
  }, [colors])

  const ref = useUPlot(opts, data, 200)
  return <div className="uplot-wrap" ref={ref} />
})
