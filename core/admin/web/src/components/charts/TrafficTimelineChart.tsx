import { memo, useMemo } from 'react'
import type { OverviewMetrics } from '../../api/client'
import { buildTrafficTimelineOption, niceAxisMax } from '../../lib/overviewEChartsOptions'
import { readChartColors } from './chartTheme'
import { EChartView } from './EChartView'

type Props = {
  timeline: OverviewMetrics['timeline']
}

export const TrafficTimelineChart = memo(function TrafficTimelineChart({ timeline }: Props) {
  const colors = useMemo(() => readChartColors(), [])
  const peak = useMemo(
    () =>
      timeline.reduce((max, b) => {
        const sum = b['2xx'] + b['3xx'] + b['4xx'] + b['5xx']
        return sum > max ? sum : max
      }, 0),
    [timeline],
  )
  const yMax = useMemo(() => niceAxisMax(peak), [peak])
  const option = useMemo(
    () => buildTrafficTimelineOption(timeline, colors, { yMax }),
    [timeline, colors, yMax],
  )
  return <EChartView option={option} height={200} />
})
