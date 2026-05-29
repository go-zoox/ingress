import { useCallback, useEffect, useRef, useState } from 'react'
import { isOverviewSSEPatch, type OverviewSSEPatch } from '../lib/overviewMerge'

const INITIAL_RECONNECT_MS = 1000
const MAX_RECONNECT_MS = 30_000
/** Start REST polling in parallel after this many failed reconnect attempts. */
const FALLBACK_AFTER_ATTEMPTS = 4

/** Polling interval in milliseconds when SSE is unavailable. */
const POLL_INTERVAL_MS = 5000

export type SSEOptions = {
  /** Metrics window for the overview SSE channel. */
  window?: string
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
}

/**
 * useSSE manages an EventSource connection to the admin SSE endpoint.
 * Custom SSE event types require addEventListener (onmessage only handles the default event).
 */
export function useSSE(channels: string[] = [], options?: SSEOptions) {
  const [data, setData] = useState<Record<string, unknown>>({})
  const [overviewPatch, setOverviewPatch] = useState<OverviewSSEPatch | null>(null)
  const [connected, setConnected] = useState(false)
  const [reconnecting, setReconnecting] = useState(false)
  const [fallbackPolling, setFallbackPolling] = useState(false)

  const esRef = useRef<EventSource | null>(null)
  const pollTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
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
    const window = optionsRef.current?.window
    if (window) {
      params.set('window', window)
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

    const window = optionsRef.current?.window || '15m'
    const channels = channelsRef.current

    setFallbackPolling(true)
    if (channels.includes('overview') && !channels.includes('metrics')) {
      // Overview page polls REST snapshot; keep reconnect attempts running.
      return
    }
    pollTimerRef.current = window.setInterval(async () => {
      try {
        if (channels.includes('metrics')) {
          const res = await fetch(
            `/api/v1/metrics/overview?window=${encodeURIComponent(window)}`,
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
        // Initial full state comes from REST; SSE only streams field-level patches.
        return
      }
      if (eventType === 'overview:patch' || isOverviewSSEPatch(parsed)) {
        setOverviewPatch(parsed as OverviewSSEPatch)
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
    const es = new EventSource(url)
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
  }, [connect, close, options?.window, options?.enabled, channels.join(',')])

  useEffect(() => {
    const onUnload = () => close()
    window.addEventListener('beforeunload', onUnload)
    return () => window.removeEventListener('beforeunload', onUnload)
  }, [close])

  return { data, overviewPatch, connected, reconnecting, fallbackPolling, close }
}
