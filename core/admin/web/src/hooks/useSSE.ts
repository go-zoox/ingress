import { useCallback, useEffect, useRef, useState } from 'react'
import type { OverviewSnapshot } from '../api/client'
import { isDetailMetricsSSEPatch, type DetailMetricsSSEPatch } from '../lib/detailMetricsMerge'
import { isOverviewSSEPatch, type OverviewSSEPatch } from '../lib/overviewMerge'

export type OverviewSSESnapshot = {
  snap: OverviewSnapshot
  seq: number
}

export type DetailMetricsSSESnapshot = {
  metrics: Record<string, unknown>
  seq: number
}

const INITIAL_RECONNECT_MS = 1000
const MAX_RECONNECT_MS = 30_000
/** Start REST polling in parallel after this many failed reconnect attempts. */
const FALLBACK_AFTER_ATTEMPTS = 4

/** Polling interval in milliseconds when SSE is unavailable. */
const POLL_INTERVAL_MS = 5000

export type SSEOptions = {
  /** Legacy rolling window for overview SSE (preset ranges). */
  window?: string
  /** Absolute range start (RFC3339) for overview SSE. */
  from?: string
  /** Absolute range end (RFC3339) for overview SSE. */
  to?: string
  /** Route detail: rule index. */
  ri?: string
  /** Route detail: path index. */
  pi?: string
  /** Service detail: catalog service name. */
  name?: string
  /** Route scope: host filter. */
  host?: string
  /** Route scope: path filter. */
  path?: string
  /** Route scope: path match mode. */
  path_match?: string
  /** When false, no connection is opened. */
  enabled?: boolean
}

/** Known SSE event actions per channel (server sends event: channel:action). */
const CHANNEL_ACTIONS: Record<string, string[]> = {
  logs: ['line'],
  waf: ['event'],
  metrics: ['update'],
  health: ['update'],
  overview: ['snapshot', 'patch'],
  route_metrics: ['snapshot', 'patch'],
  service_metrics: ['snapshot', 'patch'],
}

/**
 * useSSE manages an EventSource connection to the admin SSE endpoint.
 * Custom SSE event types require addEventListener (onmessage only handles the default event).
 */
export function useSSE(channels: string[] = [], options?: SSEOptions) {
  const [data, setData] = useState<Record<string, unknown>>({})
  const [overviewPatch, setOverviewPatch] = useState<OverviewSSEPatch | null>(null)
  const [overviewSnapshot, setOverviewSnapshot] = useState<OverviewSSESnapshot | null>(null)
  const overviewSnapshotSeqRef = useRef(0)
  const [detailMetricsPatch, setDetailMetricsPatch] = useState<DetailMetricsSSEPatch | null>(null)
  const [detailMetricsSnapshot, setDetailMetricsSnapshot] = useState<DetailMetricsSSESnapshot | null>(null)
  const detailMetricsSnapshotSeqRef = useRef(0)
  const [connected, setConnected] = useState(false)
  const [reconnecting, setReconnecting] = useState(false)
  const [fallbackPolling, setFallbackPolling] = useState(false)

  const esRef = useRef<EventSource | null>(null)
  const pollTimerRef = useRef<number | null>(null)
  const reconnectTimerRef = useRef<number | null>(null)
  const reconnectDelayRef = useRef(INITIAL_RECONNECT_MS)
  const reconnectAttemptRef = useRef(0)
  const intentionalCloseRef = useRef(false)
  const channelsRef = useRef(channels)
  const optionsRef = useRef(options)
  channelsRef.current = channels
  optionsRef.current = options

  const buildURL = useCallback(() => {
    const ch = channelsRef.current.join(',')
    const params = new URLSearchParams({ channels: ch })
    const opts = optionsRef.current
    if (opts?.window) {
      params.set('window', opts.window)
    }
    if (opts?.from) {
      params.set('from', opts.from)
    }
    if (opts?.to) {
      params.set('to', opts.to)
    }
    if (opts?.ri) {
      params.set('ri', opts.ri)
    }
    if (opts?.pi) {
      params.set('pi', opts.pi)
    }
    if (opts?.name) {
      params.set('name', opts.name)
    }
    if (opts?.host) {
      params.set('host', opts.host)
    }
    if (opts?.path) {
      params.set('path', opts.path)
    }
    if (opts?.path_match) {
      params.set('path_match', opts.path_match)
    }
    return `/api/v1/events/stream?${params.toString()}`
  }, [])

  const stopPolling = useCallback(() => {
    if (pollTimerRef.current) {
      clearInterval(pollTimerRef.current)
      pollTimerRef.current = null
    }
    setFallbackPolling(false)
  }, [])

  const clearReconnectTimer = useCallback(() => {
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current)
      reconnectTimerRef.current = null
    }
  }, [])

  const close = useCallback(() => {
    intentionalCloseRef.current = true
    clearReconnectTimer()
    if (esRef.current) {
      esRef.current.close()
      esRef.current = null
    }
    stopPolling()
    setConnected(false)
    setReconnecting(false)
  }, [clearReconnectTimer, stopPolling])

  const startPolling = useCallback(() => {
    if (pollTimerRef.current) return

    const metricsWindow = optionsRef.current?.window || '15m'
    const channels = channelsRef.current

    setFallbackPolling(true)
    if (
      channels.includes('overview') ||
      channels.includes('route_metrics') ||
      channels.includes('service_metrics')
    ) {
      if (!channels.includes('metrics')) {
        // Detail/overview pages poll REST snapshot; keep reconnect attempts running.
        return
      }
    }
    pollTimerRef.current = window.setInterval(async () => {
      try {
        if (channels.includes('metrics')) {
          const res = await fetch(
            `/api/v1/metrics/overview?window=${encodeURIComponent(metricsWindow)}`,
          )
          const envelope = await res.json()
          if (envelope.code >= 400) return
          setData((prev) => ({ ...prev, metrics: envelope.result }))
        }
      } catch {
        // keep polling; SSE reconnect loop continues in parallel
      }
    }, POLL_INTERVAL_MS)
  }, [])

  const handleEvent = useCallback((eventType: string, raw: string) => {
    try {
      const parsed = JSON.parse(raw)
      if (eventType === 'overview:snapshot') {
        overviewSnapshotSeqRef.current += 1
        setOverviewSnapshot({
          snap: parsed as OverviewSnapshot,
          seq: overviewSnapshotSeqRef.current,
        })
        return
      }
      if (eventType === 'overview:patch' || isOverviewSSEPatch(parsed)) {
        setOverviewPatch(parsed as OverviewSSEPatch)
        return
      }
      if (eventType === 'route_metrics:snapshot' || eventType === 'service_metrics:snapshot') {
        detailMetricsSnapshotSeqRef.current += 1
        setDetailMetricsSnapshot({
          metrics: parsed as Record<string, unknown>,
          seq: detailMetricsSnapshotSeqRef.current,
        })
        return
      }
      if (
        eventType === 'route_metrics:patch' ||
        eventType === 'service_metrics:patch' ||
        isDetailMetricsSSEPatch(parsed)
      ) {
        const patch = parsed as DetailMetricsSSEPatch
        if (typeof patch.data === 'string') {
          try {
            patch.data = JSON.parse(patch.data) as Record<string, unknown>
          } catch {
            // keep raw
          }
        }
        setDetailMetricsPatch(patch)
        return
      }
      const channel = eventType.split(':')[0] || 'unknown'
      setData((prev) => ({ ...prev, [channel]: parsed }))
    } catch {
      const channel = eventType.split(':')[0] || 'unknown'
      setData((prev) => ({ ...prev, [channel]: raw }))
    }
  }, [])

  const scheduleReconnect = useCallback(() => {
    clearReconnectTimer()
    if (intentionalCloseRef.current) return

    setReconnecting(true)
    const delay = reconnectDelayRef.current
    reconnectTimerRef.current = window.setTimeout(() => {
      reconnectAttemptRef.current += 1
      if (reconnectAttemptRef.current >= FALLBACK_AFTER_ATTEMPTS) {
        startPolling()
      }
      openEventSourceRef.current()
    }, delay)
    reconnectDelayRef.current = Math.min(Math.round(delay * 1.8), MAX_RECONNECT_MS)
  }, [clearReconnectTimer, startPolling])

  const openEventSourceRef = useRef<() => void>(() => {})

  openEventSourceRef.current = () => {
    if (intentionalCloseRef.current) return
    if (channelsRef.current.length === 0 || optionsRef.current?.enabled === false) return

    if (esRef.current) {
      esRef.current.close()
      esRef.current = null
    }

    const url = buildURL()
    const es = new EventSource(url, { withCredentials: true })
    esRef.current = es

    es.onopen = () => {
      reconnectAttemptRef.current = 0
      reconnectDelayRef.current = INITIAL_RECONNECT_MS
      setConnected(true)
      setReconnecting(false)
      stopPolling()
    }

    es.onmessage = (evt) => {
      if (evt.data) {
        handleEvent('message', evt.data)
      }
    }

    for (const ch of channelsRef.current) {
      const actions = CHANNEL_ACTIONS[ch] ?? ['update', 'line', 'event', 'snapshot']
      for (const action of actions) {
        const eventName = `${ch}:${action}`
        es.addEventListener(eventName, (evt) => {
          const e = evt as MessageEvent
          handleEvent(eventName, e.data)
        })
      }
    }

    es.onerror = () => {
      setConnected(false)
      es.close()
      if (esRef.current === es) {
        esRef.current = null
      }
      scheduleReconnect()
    }
  }

  const connect = useCallback(() => {
    intentionalCloseRef.current = false
    reconnectAttemptRef.current = 0
    reconnectDelayRef.current = INITIAL_RECONNECT_MS
    clearReconnectTimer()
    stopPolling()
    setReconnecting(false)
    openEventSourceRef.current()
  }, [clearReconnectTimer, stopPolling])

  useEffect(() => {
    connect()
    return () => {
      close()
    }
  }, [connect, close, options?.window, options?.from, options?.to, options?.enabled, options?.ri, options?.pi, options?.name, options?.host, options?.path, options?.path_match, channels.join(',')])

  useEffect(() => {
    setOverviewPatch(null)
    setOverviewSnapshot(null)
    setDetailMetricsPatch(null)
    setDetailMetricsSnapshot(null)
  }, [options?.window, options?.from, options?.to, options?.enabled, options?.ri, options?.pi, options?.name, options?.host, options?.path, options?.path_match])

  useEffect(() => {
    const onUnload = () => close()
    window.addEventListener('beforeunload', onUnload)
    return () => window.removeEventListener('beforeunload', onUnload)
  }, [close])

  return {
    data,
    overviewPatch,
    overviewSnapshot,
    detailMetricsPatch,
    detailMetricsSnapshot,
    connected,
    reconnecting,
    fallbackPolling,
    close,
  }
}
