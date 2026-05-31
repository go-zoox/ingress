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
    default:
      return 8
  }
}
