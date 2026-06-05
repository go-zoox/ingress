import { memo, useMemo } from 'react'
import type { OverviewMetrics } from '../../api/client'
import { buildHostTimelineOption, niceAxisMax } from '../../lib/overviewEChartsOptions'
import { OVERVIEW_TIMELINE_LINK_GROUP } from '../../lib/overviewTimelineChartGroup'
import { readChartColors } from './chartTheme'
import { EChartView } from './EChartView'

type Props = {
  series: NonNullable<OverviewMetrics['host_timeline']>
}

export const HostTrafficTimelineChart = memo(function HostTrafficTimelineChart({ series }: Props) {
  const colors = useMemo(() => readChartColors(), [])
  const peak = useMemo(
    () =>
      series.reduce((max, row) => {
        for (const p of row.points) {
          if (p.count > max) max = p.count
        }
        return max
      }, 0),
    [series],
  )
  const yMax = useMemo(() => niceAxisMax(peak), [peak])
  const option = useMemo(
    () => buildHostTimelineOption(series, colors, { yMax }),
    [series, colors, yMax],
  )
  return (
    <EChartView
      option={option}
      height={240}
      linkGroup={OVERVIEW_TIMELINE_LINK_GROUP}
      notMerge
    />
  )
})
