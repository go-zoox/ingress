import { useCallback, useEffect, useRef } from 'react'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import type { ConnState } from '../../lib/terminalTabs'

const RECONNECT_GRACE_MS = 60_000
const RECONNECT_BASE_MS = 300
const RECONNECT_MAX_MS = 5_000

type TerminalSessionMsg = {
  type: 'session'
  id: string
  reattach: boolean
}

type Props = {
  tabId: string
  sessionId: string | null
  active: boolean
  startNew: boolean
  onSessionChange: (tabId: string, sessionId: string) => void
  onConnStateChange: (tabId: string, state: ConnState) => void
}

function terminalWebSocketURL(sessionId?: string | null) {
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const base = `${proto}//${window.location.host}/api/v1/terminal/ws`
  const id = sessionId?.trim()
  if (!id) return base
  return `${base}?session=${encodeURIComponent(id)}`
}

export function TerminalTabPane({
  tabId,
  sessionId,
  active,
  startNew,
  onSessionChange,
  onConnStateChange,
}: Props) {
  const screenRef = useRef<HTMLDivElement>(null)
  const mountRef = useRef<HTMLDivElement>(null)
  const termRef = useRef<Terminal | null>(null)
  const fitRef = useRef<FitAddon | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const wsGenRef = useRef(0)
  const sessionIdRef = useRef<string | null>(sessionId)
  const startNewRef = useRef(startNew)
  const lastDimsRef = useRef<{ rows: number; cols: number } | null>(null)
  const fitFrameRef = useRef(0)
  const reconnectTimerRef = useRef(0)
  const reconnectStartedRef = useRef(0)
  const reconnectAttemptRef = useRef(0)
  const manualCloseRef = useRef(false)

  sessionIdRef.current = sessionId
  startNewRef.current = startNew

  const setConnState = useCallback(
    (state: ConnState) => {
      onConnStateChange(tabId, state)
    },
    [onConnStateChange, tabId],
  )

  const clearReconnectTimer = useCallback(() => {
    if (reconnectTimerRef.current) {
      window.clearTimeout(reconnectTimerRef.current)
      reconnectTimerRef.current = 0
    }
  }, [])

  const applyFit = useCallback((force = false) => {
    const fit = fitRef.current
    if (!fit) return

    fit.fit()
    const dims = fit.proposeDimensions()
    if (!dims) return

    const prev = lastDimsRef.current
    if (!force && prev && prev.rows === dims.rows && prev.cols === dims.cols) {
      return
    }
    lastDimsRef.current = { rows: dims.rows, cols: dims.cols }

    const ws = wsRef.current
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: 'resize', rows: dims.rows, cols: dims.cols }))
    }
  }, [])

  const scheduleFit = useCallback(
    (force = false) => {
      cancelAnimationFrame(fitFrameRef.current)
      fitFrameRef.current = requestAnimationFrame(() => applyFit(force))
    },
    [applyFit],
  )

  const connect = useCallback(
    (opts?: { forceNew?: boolean; auto?: boolean }) => {
      const term = termRef.current
      if (!term) return

      clearReconnectTimer()
      manualCloseRef.current = false

      wsRef.current?.close()
      const gen = ++wsGenRef.current
      setConnState(opts?.auto ? 'reconnecting' : 'connecting')

      const useNew = opts?.forceNew || startNewRef.current
      startNewRef.current = false
      const requestedSession = useNew ? null : sessionIdRef.current

      const ws = new WebSocket(terminalWebSocketURL(requestedSession))
      ws.binaryType = 'arraybuffer'
      wsRef.current = ws

      ws.onopen = () => {
        if (wsGenRef.current !== gen) return
        setConnState('open')
        reconnectAttemptRef.current = 0
        reconnectStartedRef.current = 0
        clearReconnectTimer()
        scheduleFit(true)
      }

      ws.onmessage = (ev) => {
        if (wsGenRef.current !== gen) return
        if (typeof ev.data === 'string') {
          try {
            const msg = JSON.parse(ev.data) as TerminalSessionMsg
            if (msg.type === 'session' && msg.id) {
              sessionIdRef.current = msg.id
              onSessionChange(tabId, msg.id)
              if (!msg.reattach) {
                term.reset()
              }
              scheduleFit(true)
              return
            }
          } catch {
            /* fall through */
          }
          term.write(ev.data)
          return
        }
        term.write(new Uint8Array(ev.data))
      }

      ws.onclose = () => {
        if (wsGenRef.current !== gen) return
        if (manualCloseRef.current) {
          setConnState('closed')
          return
        }

        const started = reconnectStartedRef.current
        if (started === 0) {
          reconnectStartedRef.current = Date.now()
        } else if (Date.now() - started >= RECONNECT_GRACE_MS) {
          setConnState('closed')
          term.writeln('\r\n\x1b[90m[会话已过期，请新建标签或重新连接]\x1b[0m')
          return
        }

        setConnState('reconnecting')
        const attempt = reconnectAttemptRef.current++
        const delay = Math.min(RECONNECT_BASE_MS * 2 ** attempt, RECONNECT_MAX_MS)
        reconnectTimerRef.current = window.setTimeout(() => {
          connect({ auto: true })
        }, delay)
      }

      ws.onerror = () => {
        if (wsGenRef.current !== gen) return
        if (ws.readyState === WebSocket.OPEN) {
          setConnState('error')
        }
      }
    },
    [clearReconnectTimer, onSessionChange, scheduleFit, setConnState, tabId],
  )

  useEffect(() => {
    const mount = mountRef.current
    const screen = screenRef.current
    if (!mount || !screen) return

    const term = new Terminal({
      cursorBlink: true,
      fontSize: 13,
      fontFamily: '"IBM Plex Mono", "SF Mono", Menlo, Monaco, Consolas, monospace',
      letterSpacing: 0,
      lineHeight: 1,
      screenReaderMode: false,
      theme: {
        background: '#0d1117',
        foreground: '#c9d1d9',
        cursor: '#58a6ff',
        selectionBackground: '#264f78',
      },
    })
    const fit = new FitAddon()
    term.loadAddon(fit)
    term.open(mount)

    termRef.current = term
    fitRef.current = fit

    const fitWhenReady = () => {
      if (active) scheduleFit(true)
    }
    if (document.fonts?.ready) {
      void document.fonts.ready.then(fitWhenReady)
    } else {
      fitWhenReady()
    }

    term.onData((data) => {
      const ws = wsRef.current
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(data)
      }
    })

    connect()

    const onResize = () => {
      if (active) scheduleFit()
    }
    window.addEventListener('resize', onResize)
    const ro = new ResizeObserver(onResize)
    ro.observe(screen)

    return () => {
      manualCloseRef.current = true
      const ws = wsRef.current
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'close' }))
      }
      wsGenRef.current += 1
      clearReconnectTimer()
      cancelAnimationFrame(fitFrameRef.current)
      window.removeEventListener('resize', onResize)
      ro.disconnect()
      ws?.close()
      wsRef.current = null
      term.dispose()
      termRef.current = null
      fitRef.current = null
    }
    // Mount once per tab; connect reads refs for session/startNew.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tabId])

  useEffect(() => {
    if (active) {
      scheduleFit(true)
    }
  }, [active, scheduleFit])

  return (
    <div className={`web-terminal-pane${active ? ' active' : ''}`} data-tab-id={tabId} role="tabpanel">
      <div className="web-terminal-screen" ref={screenRef}>
        <div className="web-terminal-mount" ref={mountRef} />
      </div>
    </div>
  )
}
