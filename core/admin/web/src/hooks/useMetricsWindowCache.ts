import { useEffect, useRef, useState } from 'react'
import type { OverviewMetrics, SystemMetrics } from '../api/client'
import { type OverviewRange, rangeQueryKey, snapshotMatchesRange } from '../lib/overviewRange'

/** Keep the latest in-window snapshot per range for stale-while-revalidate on tab switches. */
export function useOverviewMetricsCache(
  metrics: OverviewMetrics | null,
  overviewRange: OverviewRange,
): OverviewMetrics | null {
  const windowKey = rangeQueryKey(overviewRange)
  const cacheRef = useRef<Partial<Record<string, OverviewMetrics>>>({})
  const [cached, setCached] = useState<OverviewMetrics | null>(() => cacheRef.current[windowKey] ?? null)

  useEffect(() => {
    setCached(cacheRef.current[windowKey] ?? null)
  }, [windowKey])

  useEffect(() => {
    if (!snapshotMatchesRange(metrics, overviewRange) || !metrics) {
      return
    }
    cacheRef.current[windowKey] = metrics
    setCached(metrics)
  }, [metrics, windowKey, overviewRange])

  return cached
}

export function useSystemMetricsCache(
  system: SystemMetrics | null,
  overviewRange: OverviewRange,
): SystemMetrics | null {
  const windowKey = rangeQueryKey(overviewRange)
  const cacheRef = useRef<Partial<Record<string, SystemMetrics>>>({})
  const [cached, setCached] = useState<SystemMetrics | null>(() => cacheRef.current[windowKey] ?? null)

  useEffect(() => {
    setCached(cacheRef.current[windowKey] ?? null)
  }, [windowKey])

  useEffect(() => {
    if (!snapshotMatchesRange(system, overviewRange) || !system) {
      return
    }
    cacheRef.current[windowKey] = system
    setCached(system)
  }, [system, windowKey, overviewRange])

  return cached
}
