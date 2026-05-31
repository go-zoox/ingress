import { useEffect, useRef } from 'react'
import uPlot, { type AlignedData, type Options } from 'uplot'

export type UPlotOptions = Omit<Options, 'width' | 'height'>
import 'uplot/dist/uPlot.min.css'

function alignedDataEqual(a: AlignedData, b: AlignedData): boolean {
  if (a.length !== b.length) return false
  for (let i = 0; i < a.length; i++) {
    const sa = a[i]
    const sb = b[i]
    if (!Array.isArray(sa) || !Array.isArray(sb) || sa.length !== sb.length) return false
    for (let j = 0; j < sa.length; j++) {
      if (sa[j] !== sb[j]) return false
    }
  }
  return true
}

/** Mount uPlot once per opts shape; update data in place to avoid chart flicker. */
export function useUPlot(opts: UPlotOptions | null, data: AlignedData | null, height: number) {
  const rootRef = useRef<HTMLDivElement>(null)
  const plotRef = useRef<uPlot | null>(null)
  const dataRef = useRef<AlignedData | null>(null)
  const resizeRafRef = useRef<ReturnType<typeof globalThis.requestAnimationFrame> | null>(null)

  const bucketCount = data?.[0]?.length ?? 0
  const optsKey = opts ? `${height}:${opts.series?.length ?? 0}:${bucketCount}` : ''

  useEffect(() => {
    const el = rootRef.current
    if (!el || !opts || !data) {
      return
    }

    plotRef.current?.destroy()
    plotRef.current = null
    el.replaceChildren()
    dataRef.current = data
    const width = el.clientWidth || 320
    plotRef.current = new uPlot({ ...opts, width, height }, data, el)

    return () => {
      plotRef.current?.destroy()
      plotRef.current = null
      el.replaceChildren()
      dataRef.current = null
    }
  }, [optsKey, opts, height])

  useEffect(() => {
    const plot = plotRef.current
    if (!plot || !data) return
    if (dataRef.current && alignedDataEqual(dataRef.current, data)) return
    dataRef.current = data
    plot.setData(data, false)
  }, [data])

  useEffect(() => {
    const el = rootRef.current
    if (!el) return

    const ro = new ResizeObserver(() => {
      if (resizeRafRef.current != null) {
        globalThis.cancelAnimationFrame(resizeRafRef.current)
      }
      resizeRafRef.current = globalThis.requestAnimationFrame(() => {
        resizeRafRef.current = null
        if (!plotRef.current || !rootRef.current) return
        plotRef.current.setSize({ width: rootRef.current.clientWidth, height })
      })
    })
    ro.observe(el)

    return () => {
      ro.disconnect()
      if (resizeRafRef.current != null) {
        globalThis.cancelAnimationFrame(resizeRafRef.current)
        resizeRafRef.current = null
      }
    }
  }, [height])

  return rootRef
}
