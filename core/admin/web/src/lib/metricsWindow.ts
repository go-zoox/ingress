/** Overview UI value for live SSE mode (maps to {@link LIVE_METRICS_WINDOW} on the API). */
export const LIVE_OVERVIEW_VIEW = 'live'

/** Rolling window used while the overview is in live mode. */
export const LIVE_METRICS_WINDOW = '5m'

export function isLiveOverviewView(view: string) {
  return view.trim() === LIVE_OVERVIEW_VIEW
}

/** Map UI view selection to the metrics API `window` query param. */
export function resolveMetricsWindow(view: string) {
  if (isLiveOverviewView(view)) return LIVE_METRICS_WINDOW
  return normalizeMetricsWindow(view)
}

/** Normalize overview page view selection (includes legacy live). */
export function normalizeOverviewView(view: string) {
  const w = view.trim()
  if (w === LIVE_OVERVIEW_VIEW) return LIVE_OVERVIEW_VIEW
  if (w === '60m') return '1h'
  if (w === '24h' || w === '6h' || w === '1h' || w === '5m' || w === '15m') return w
  return '5m'
}

/** Align client window values with backend normalizeMetricsWindow. */
export function normalizeMetricsWindow(window: string) {
  const w = window.trim()
  if (w === LIVE_OVERVIEW_VIEW) return LIVE_METRICS_WINDOW
  if (w === '60m') return '1h'
  if (w === '24h' || w === '6h' || w === '1h' || w === '5m' || w === '15m') return w
  return '15m'
}

/** Chart bucket count for a metrics window (matches backend timelineBucketsForWindow). */
export function timelineBucketsForWindow(window: string) {
  switch (resolveMetricsWindow(window)) {
    case '24h':
      return 24
    case '6h':
      return 12
    case '1h':
      return 12
    case '5m':
      return 5
    case '15m':
      return 15
    default:
      return 15
  }
}

/** True when a snapshot section belongs to the selected metrics window. */
export function snapshotMatchesWindow(
  value: { window?: string } | null | undefined,
  metricsWindow: string,
): boolean {
  if (!value?.window) return false
  return normalizeMetricsWindow(value.window) === resolveMetricsWindow(metricsWindow)
}

/** Human-readable rolling window label for UI copy. */
export function formatMetricsWindowLabel(window: string): string {
  if (isLiveOverviewView(window)) {
    return '实时 · 最近 5 分钟'
  }
  switch (normalizeMetricsWindow(window)) {
    case '5m':
      return '最近 5 分钟'
    case '15m':
      return '最近 15 分钟'
    case '1h':
      return '最近 1 小时'
    case '6h':
      return '最近 6 小时'
    case '24h':
      return '最近 24 小时'
    default:
      return '最近 15 分钟'
  }
}
