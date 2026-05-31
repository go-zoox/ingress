/** Align client window values with backend normalizeMetricsWindow. */
export function normalizeMetricsWindow(window: string) {
  const w = window.trim()
  if (w === '60m') return '1h'
  if (w === '24h' || w === '1h' || w === '5m' || w === '15m') return w
  return '15m'
}

/** Chart bucket count for a metrics window (matches backend timelineBucketsForWindow). */
export function timelineBucketsForWindow(window: string) {
  switch (normalizeMetricsWindow(window)) {
    case '24h':
      return 24
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
  return normalizeMetricsWindow(value.window) === normalizeMetricsWindow(metricsWindow)
}

/** Human-readable rolling window label for UI copy. */
export function formatMetricsWindowLabel(window: string): string {
  switch (normalizeMetricsWindow(window)) {
    case '5m':
      return '最近 5 分钟'
    case '15m':
      return '最近 15 分钟'
    case '1h':
      return '最近 1 小时'
    case '24h':
      return '最近 24 小时'
    default:
      return '最近 15 分钟'
  }
}
