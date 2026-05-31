import { memo } from 'react'
import type { EChartsOption } from 'echarts'
import { useECharts } from './useECharts'

type Props = {
  option: EChartsOption | null
  className?: string
  height?: number
  notMerge?: boolean
}

export const EChartView = memo(function EChartView({
  option,
  className = 'echarts-wrap',
  height = 200,
  notMerge,
}: Props) {
  const ref = useECharts(option, { notMerge })
  return <div ref={ref} className={className} style={{ height }} aria-hidden={option == null} />
})
