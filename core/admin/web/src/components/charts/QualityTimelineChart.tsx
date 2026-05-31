import { memo, useMemo } from 'react'
import type { OverviewMetrics } from '../../api/client'
import { buildQualityTimelineOption, niceAxisMax } from '../../lib/overviewEChartsOptions'
import { readChartColors } from './chartTheme'
import { EChartView } from './EChartView'

type Props = {
  timeline: OverviewMetrics['timeline']
}

export const QualityTimelineChart = memo(function QualityTimelineChart({ timeline }: Props) {
  const colors = useMemo(() => readChartColors(), [])
  const errorPeak = useMemo(
    () => timeline.reduce((max, b) => (b.error_rate > max ? b.error_rate : max), 0),
    [timeline],
  )
  const wafPeak = useMemo(
    () => timeline.reduce((max, b) => (b.waf_blocks > max ? b.waf_blocks : max), 0),
    [timeline],
  )
  const yMax = useMemo(() => niceAxisMax(errorPeak * 1.15, 1), [errorPeak])
  const y2Max = useMemo(() => niceAxisMax(wafPeak * 1.15, 1), [wafPeak])
  const option = useMemo(
    () => buildQualityTimelineOption(timeline, colors, { yMax, y2Max }),
    [timeline, colors, yMax, y2Max],
  )
  return <EChartView option={option} height={200} />
})
