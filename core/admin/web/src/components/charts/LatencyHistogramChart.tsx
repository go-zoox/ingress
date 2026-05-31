import { memo, useMemo } from 'react'
import type { OverviewMetrics } from '../../api/client'
import { buildLatencyHistogramOption, niceAxisMax } from '../../lib/overviewEChartsOptions'
import { readChartColors } from './chartTheme'
import { EChartView } from './EChartView'

type Props = {
  histogram: OverviewMetrics['latency_histogram']
}

export const LatencyHistogramChart = memo(function LatencyHistogramChart({ histogram }: Props) {
  const colors = useMemo(() => readChartColors(), [])
  const peak = useMemo(
    () => histogram.reduce((max, b) => (b.count > max ? b.count : max), 0),
    [histogram],
  )
  const yMax = useMemo(() => niceAxisMax(peak), [peak])
  const option = useMemo(
    () => buildLatencyHistogramOption(histogram, colors, { yMax }),
    [histogram, colors, yMax],
  )
  return <EChartView option={option} height={180} />
})
