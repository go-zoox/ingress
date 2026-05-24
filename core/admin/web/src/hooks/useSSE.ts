import { useCallback, useEffect, useRef, useState } from 'react'

/** Maximum SSE reconnection attempts before falling back to polling. */
const MAX_RECONNECT = 3

/** Polling interval in milliseconds when SSE is unavailable. */
const POLL_INTERVAL_MS = 2000

/**
 * useSSE manages an EventSource connection to the admin SSE endpoint.
 * On connection failure it retries up to MAX_RECONNECT times, then
 * falls back to setInterval polling.
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
    // Poll each relevant API endpoint at POLL_INTERVAL_MS
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

  const connect = useCallback(() => {
    // Clean up any existing connection
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

    es.onmessage = (evt) => {
      try {
        const parsed = JSON.parse(evt.data)
        // Determine channel from event type format "channel:action"
        const eventType = (evt as MessageEvent).type ?? ''
        const channel = eventType.split(':')[0] || 'unknown'
        setData((prev) => ({ ...prev, [channel]: parsed }))
      } catch {
        // If data is not JSON, store as raw string
        setData((prev) => ({ ...prev, raw: evt.data }))
      }
    }

    es.onerror = () => {
      setConnected(false)
      reconnectCountRef.current += 1

      if (reconnectCountRef.current >= MAX_RECONNECT) {
        // Give up on SSE, fall back to polling
        es.close()
        esRef.current = null
        setError(new Error('SSE connection failed, falling back to polling'))
        startPolling()
      } else {
        setError(new Error(`SSE connection lost (attempt ${reconnectCountRef.current}/${MAX_RECONNECT})`))
        // EventSource will auto-reconnect; just update state
      }
    }
  }, [buildURL, startPolling])

  useEffect(() => {
    connect()
    return () => {
      close()
    }
  }, [connect, close])

  // Close on page unload
  useEffect(() => {
    const onUnload = () => close()
    window.addEventListener('beforeunload', onUnload)
    return () => window.removeEventListener('beforeunload', onUnload)
  }, [close])

  return { data, connected, error, fallbackPolling, close }
}
