import { useEffect, useRef } from 'react'
import * as echarts from 'echarts'
import type { ECharts, EChartsOption } from 'echarts'

type Options = {
  /** Replace entire option instead of merging (default false). */
  notMerge?: boolean
}

/** Mount ECharts once; update option in place on changes. */
export function useECharts(option: EChartsOption | null, options?: Options) {
  const rootRef = useRef<HTMLDivElement>(null)
  const chartRef = useRef<ECharts | null>(null)
  const notMerge = options?.notMerge ?? false

  useEffect(() => {
    const el = rootRef.current
    if (!el) return

    const chart = echarts.init(el, undefined, { renderer: 'canvas' })
    chartRef.current = chart

    const ro = new ResizeObserver(() => {
      chart.resize()
    })
    ro.observe(el)

    return () => {
      ro.disconnect()
      chart.dispose()
      chartRef.current = null
    }
  }, [])

  useEffect(() => {
    const chart = chartRef.current
    if (!chart || !option) return
    chart.setOption(option, { notMerge, lazyUpdate: true })
  }, [option, notMerge])

  return rootRef
}
