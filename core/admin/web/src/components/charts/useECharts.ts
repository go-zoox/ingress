import { useEffect, useRef } from 'react'
import * as echarts from 'echarts'
import type { ECharts, EChartsOption } from 'echarts'
import {
  registerOverviewTimelineChart,
  unregisterOverviewTimelineChart,
} from '../../lib/overviewTimelineChartGroup'

type Options = {
  /** Replace entire option instead of merging (default false). */
  notMerge?: boolean
  /** ECharts connect group id (overview timeline crosshair sync). */
  linkGroup?: string
}

/** Mount ECharts once; update option in place on changes. */
export function useECharts(option: EChartsOption | null, options?: Options) {
  const rootRef = useRef<HTMLDivElement>(null)
  const chartRef = useRef<ECharts | null>(null)
  const notMerge = options?.notMerge ?? false
  const linkGroup = options?.linkGroup

  useEffect(() => {
    const el = rootRef.current
    if (!el) return

    const chart = echarts.init(el, undefined, { renderer: 'canvas' })
    chartRef.current = chart
    if (linkGroup) {
      registerOverviewTimelineChart(linkGroup, chart)
    }

    const ro = new ResizeObserver(() => {
      chart.resize()
    })
    ro.observe(el)

    return () => {
      if (linkGroup) {
        unregisterOverviewTimelineChart(linkGroup, chart)
      }
      ro.disconnect()
      chart.dispose()
      chartRef.current = null
    }
  }, [linkGroup])

  useEffect(() => {
    const chart = chartRef.current
    if (!chart || !option) return
    chart.setOption(option, { notMerge, lazyUpdate: true })
  }, [option, notMerge])

  return rootRef
}
