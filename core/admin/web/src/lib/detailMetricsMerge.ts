import type { RouteMetrics, ServiceMetrics } from '../api/client'
import { normalizeMetricsWindow } from './metricsWindow'
import type { OverviewRange } from './overviewRange'
import { mergeRollingWindowForRange, resolveApiWindow } from './overviewRange'

export type DetailMetricsSSEPatch = {
  window?: string
  seq?: number
  data?: Record<string, unknown>
}

export function isDetailMetricsSSEPatch(payload: unknown): payload is DetailMetricsSSEPatch {
  if (!payload || typeof payload !== 'object') return false
  return typeof (payload as DetailMetricsSSEPatch).seq === 'number'
}

export function detailPatchWindowMismatch(
  patch: DetailMetricsSSEPatch,
  range: OverviewRange,
  sseWindow?: string,
): boolean {
  if (!patch.window || patch.window === 'range') return false
  const expected = normalizeMetricsWindow(resolveApiWindow(range))
  const patchWindow = normalizeMetricsWindow(patch.window)
  if (range.kind === 'preset') {
    return patchWindow !== expected
  }
  if (sseWindow) {
    return patchWindow !== normalizeMetricsWindow(sseWindow)
  }
  return false
}

export function mergeDetailMetricsPatch<T extends RouteMetrics | ServiceMetrics>(
  base: T,
  patch: DetailMetricsSSEPatch,
  range: OverviewRange,
): T | null {
  if (!patch.data || Object.keys(patch.data).length === 0) {
    if (patch.window && patch.window !== base.window && patch.window !== 'range') {
      return null
    }
    return base
  }
  const rollingWindow = mergeRollingWindowForRange(range)
  if (patch.window && patch.window !== 'range') {
    const expected =
      range.kind === 'preset'
        ? normalizeMetricsWindow(range.preset)
        : rollingWindow
          ? normalizeMetricsWindow(rollingWindow)
          : base.window
    if (normalizeMetricsWindow(patch.window) !== normalizeMetricsWindow(expected)) {
      return null
    }
  }
  return { ...base, ...patch.data } as T
}
