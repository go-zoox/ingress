import { useEffect, useRef, useState } from 'react'
import type { OverviewMetrics, SystemMetrics } from '../api/client'
import { normalizeMetricsWindow, snapshotMatchesWindow } from '../lib/metricsWindow'

/** Keep the latest in-window snapshot per range for stale-while-revalidate on tab switches. */
export function useOverviewMetricsCache(
  metrics: OverviewMetrics | null,
  metricsWindow: string,
): OverviewMetrics | null {
  const windowKey = normalizeMetricsWindow(metricsWindow)
  const cacheRef = useRef<Partial<Record<string, OverviewMetrics>>>({})
  const [cached, setCached] = useState<OverviewMetrics | null>(() => cacheRef.current[windowKey] ?? null)

  useEffect(() => {
    setCached(cacheRef.current[windowKey] ?? null)
  }, [windowKey])

  useEffect(() => {
    if (!snapshotMatchesWindow(metrics, windowKey) || !metrics) {
      return
    }
    cacheRef.current[windowKey] = metrics
    setCached(metrics)
  }, [metrics, windowKey])

  return cached
}

export function useSystemMetricsCache(
  system: SystemMetrics | null,
  metricsWindow: string,
): SystemMetrics | null {
  const windowKey = normalizeMetricsWindow(metricsWindow)
  const cacheRef = useRef<Partial<Record<string, SystemMetrics>>>({})
  const [cached, setCached] = useState<SystemMetrics | null>(() => cacheRef.current[windowKey] ?? null)

  useEffect(() => {
    setCached(cacheRef.current[windowKey] ?? null)
  }, [windowKey])

  useEffect(() => {
    if (!snapshotMatchesWindow(system, windowKey) || !system) {
      return
    }
    cacheRef.current[windowKey] = system
    setCached(system)
  }, [system, windowKey])

  return cached
}
