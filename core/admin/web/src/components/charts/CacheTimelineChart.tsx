import { memo, useMemo } from 'react'
import type { OverviewMetrics } from '../../api/client'
import { buildCacheTimelineOption } from '../../lib/overviewEChartsOptions'
import { readChartColors } from './chartTheme'
import { EChartView } from './EChartView'

type Props = {
  timeline: OverviewMetrics['timeline']
}

export const CacheTimelineChart = memo(function CacheTimelineChart({ timeline }: Props) {
  const colors = useMemo(() => readChartColors(), [])
  const option = useMemo(() => buildCacheTimelineOption(timeline, colors), [timeline, colors])
  return <EChartView option={option} height={180} />
})
