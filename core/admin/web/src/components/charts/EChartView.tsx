import { memo } from 'react'
import type { EChartsOption } from 'echarts'
import { useECharts } from './useECharts'

type Props = {
  option: EChartsOption | null
  className?: string
  height?: number
  notMerge?: boolean
  /** Share crosshair/tooltip with other charts in the same ECharts connect group. */
  linkGroup?: string
}

export const EChartView = memo(function EChartView({
  option,
  className = 'echarts-wrap',
  height = 200,
  notMerge,
  linkGroup,
}: Props) {
  const ref = useECharts(option, { notMerge, linkGroup })
  return <div ref={ref} className={className} style={{ height }} aria-hidden={option == null} />
})
