import { useEffect, useRef } from 'react'
import uPlot, { type AlignedData, type Options } from 'uplot'

export type UPlotOptions = Omit<Options, 'width' | 'height'>
import 'uplot/dist/uPlot.min.css'

export function useUPlot(opts: UPlotOptions | null, data: AlignedData | null, height: number) {
  const rootRef = useRef<HTMLDivElement>(null)
  const plotRef = useRef<uPlot | null>(null)

  useEffect(() => {
    const el = rootRef.current
    if (!el || !opts || !data) {
      return
    }
    const width = el.clientWidth || 320
    const next = new uPlot({ ...opts, width, height }, data, el)
    plotRef.current = next

    const ro = new ResizeObserver(() => {
      if (!plotRef.current || !rootRef.current) return
      plotRef.current.setSize({ width: rootRef.current.clientWidth, height })
    })
    ro.observe(el)

    return () => {
      ro.disconnect()
      plotRef.current?.destroy()
      plotRef.current = null
    }
  }, [opts, data, height])

  return rootRef
}
