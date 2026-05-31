import { memo, useMemo } from 'react'
import type { MetricsTimelineBucket } from '../../api/client'
import { buildUpstreamLatencyTrendOption } from '../../lib/overviewEChartsOptions'
import { readChartColors } from './chartTheme'
import { EChartView } from './EChartView'

type Props = {
  timeline: MetricsTimelineBucket[]
}

export const UpstreamLatencyTrendChart = memo(function UpstreamLatencyTrendChart({ timeline }: Props) {
  const colors = useMemo(() => readChartColors(), [])
  const hasData = timeline.some((b) => (b.upstream_p95_ms ?? 0) > 0)
  const option = useMemo(
    () => (hasData ? buildUpstreamLatencyTrendOption(timeline, colors) : null),
    [timeline, colors, hasData],
  )

  if (!hasData) {
    return <p className="empty-hint">暂无上游耗时数据（需 access log 含 upstream_response_time）</p>
  }

  return <EChartView option={option} height={200} />
})
