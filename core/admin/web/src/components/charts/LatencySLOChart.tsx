import { memo, useMemo } from 'react'
import type { OverviewMetrics } from '../../api/client'
import { buildLatencySLOOption } from '../../lib/overviewEChartsOptions'
import { readChartColors } from './chartTheme'
import { EChartView } from './EChartView'

type Props = {
  segments: OverviewMetrics['latency_slo']
}

export const LatencySLOChart = memo(function LatencySLOChart({ segments }: Props) {
  const colors = useMemo(() => readChartColors(), [])
  const option = useMemo(() => buildLatencySLOOption(segments ?? [], colors), [segments, colors])
  return <EChartView option={option} height={180} />
})
