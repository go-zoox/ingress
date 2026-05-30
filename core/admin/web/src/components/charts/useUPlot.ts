import { useEffect, useRef } from 'react'
import uPlot, { type AlignedData, type Options } from 'uplot'

export type UPlotOptions = Omit<Options, 'width' | 'height'>
import 'uplot/dist/uPlot.min.css'

function seriesCount(data: AlignedData | null) {
  if (!data) return 0
  return Math.max(0, data.length - 1)
}

/** Mount uPlot once per opts shape; update data in place to avoid chart flicker. */
export function useUPlot(opts: UPlotOptions | null, data: AlignedData | null, height: number) {
  const rootRef = useRef<HTMLDivElement>(null)
  const plotRef = useRef<uPlot | null>(null)
  const optsKeyRef = useRef('')

  useEffect(() => {
    const el = rootRef.current
    if (!el || !opts || !data) {
      return
    }

    const optsKey = `${height}:${seriesCount(data)}:${opts.series?.length ?? 0}`
    const needsCreate = !plotRef.current || optsKeyRef.current !== optsKey

    if (needsCreate) {
      plotRef.current?.destroy()
      plotRef.current = null
      optsKeyRef.current = optsKey
      const width = el.clientWidth || 320
      plotRef.current = new uPlot({ ...opts, width, height }, data, el)
    } else {
      const plot = plotRef.current
      if (!plot) return
      plot.setData(data)
      plot.redraw()
    }

    const ro = new ResizeObserver(() => {
      if (!plotRef.current || !rootRef.current) return
      plotRef.current.setSize({ width: rootRef.current.clientWidth, height })
    })
    ro.observe(el)

    return () => {
      ro.disconnect()
    }
  }, [opts, data, height])

  useEffect(() => {
    return () => {
      plotRef.current?.destroy()
      plotRef.current = null
      optsKeyRef.current = ''
    }
  }, [])

  return rootRef
}
