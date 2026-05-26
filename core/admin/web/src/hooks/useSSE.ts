import { useCallback, useEffect, useRef, useState } from 'react'

/** Maximum SSE reconnection attempts before falling back to polling. */
const MAX_RECONNECT = 3

/** Polling interval in milliseconds when SSE is unavailable. */
const POLL_INTERVAL_MS = 2000

/** Known SSE event actions per channel (server sends event: channel:action). */
const CHANNEL_ACTIONS: Record<string, string[]> = {
  logs: ['line'],
  waf: ['event'],
  metrics: ['update'],
  health: ['update'],
}

/**
 * useSSE manages an EventSource connection to the admin SSE endpoint.
 * Custom SSE event types require addEventListener (onmessage only handles the default event).
 */
export function useSSE(channels: string[] = []) {
  const [data, setData] = useState<Record<string, unknown>>({})
  const [connected, setConnected] = useState(false)
  const [error, setError] = useState<Error | null>(null)
  const [fallbackPolling, setFallbackPolling] = useState(false)

  const esRef = useRef<EventSource | null>(null)
  const reconnectCountRef = useRef(0)
  const pollTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const channelsRef = useRef(channels)
  channelsRef.current = channels

  const buildURL = useCallback(() => {
    const ch = channelsRef.current.join(',')
    return `/api/v1/events/stream?channels=${encodeURIComponent(ch)}`
  }, [])

  const close = useCallback(() => {
    if (esRef.current) {
      esRef.current.close()
      esRef.current = null
    }
    if (pollTimerRef.current) {
      clearInterval(pollTimerRef.current)
      pollTimerRef.current = null
    }
    setConnected(false)
    setFallbackPolling(false)
  }, [])

  const startPolling = useCallback(() => {
    setFallbackPolling(true)
    setConnected(false)
    pollTimerRef.current = window.setInterval(async () => {
      try {
        const res = await fetch('/api/v1/metrics/overview?window=15m')
        const envelope = await res.json()
        if (envelope.code >= 400) return
        setData((prev) => ({ ...prev, metrics: envelope.result }))
      } catch {
        // Silently ignore polling errors
      }
    }, POLL_INTERVAL_MS)
  }, [])

  const handleEvent = useCallback((eventType: string, raw: string) => {
    const channel = eventType.split(':')[0] || 'unknown'
    try {
      const parsed = JSON.parse(raw)
      setData((prev) => ({ ...prev, [channel]: parsed }))
    } catch {
      setData((prev) => ({ ...prev, [channel]: raw }))
    }
  }, [])

  const connect = useCallback(() => {
    if (esRef.current) {
      esRef.current.close()
      esRef.current = null
    }
    if (pollTimerRef.current) {
      clearInterval(pollTimerRef.current)
      pollTimerRef.current = null
    }

    if (channelsRef.current.length === 0) return

    const url = buildURL()
    const es = new EventSource(url)
    esRef.current = es

    es.onopen = () => {
      setConnected(true)
      setError(null)
      reconnectCountRef.current = 0
    }

    // Default message event (e.g. connected)
    es.onmessage = (evt) => {
      if (evt.data) {
        handleEvent('message', evt.data)
      }
    }

    // Named events: logs:line, waf:event, metrics:update, health:update
    for (const ch of channelsRef.current) {
      const actions = CHANNEL_ACTIONS[ch] ?? ['update', 'line', 'event']
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
      reconnectCountRef.current += 1

      if (reconnectCountRef.current >= MAX_RECONNECT) {
        es.close()
        esRef.current = null
        setError(new Error('SSE connection failed, falling back to polling'))
        startPolling()
      } else {
        setError(
          new Error(`SSE connection lost (attempt ${reconnectCountRef.current}/${MAX_RECONNECT})`),
        )
      }
    }
  }, [buildURL, startPolling, handleEvent])

  useEffect(() => {
    connect()
    return () => {
      close()
    }
  }, [connect, close])

  useEffect(() => {
    const onUnload = () => close()
    window.addEventListener('beforeunload', onUnload)
    return () => window.removeEventListener('beforeunload', onUnload)
  }, [close])

  return { data, connected, error, fallbackPolling, close }
}
