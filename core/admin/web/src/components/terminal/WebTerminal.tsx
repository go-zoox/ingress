import { useCallback, useEffect, useRef, useState } from 'react'
import { Plus, RefreshCw } from 'lucide-react'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'

type ConnState = 'idle' | 'connecting' | 'open' | 'closed' | 'reconnecting' | 'error'

const SESSION_STORAGE_KEY = 'ingress_admin_terminal_session'
const LEGACY_TABS_STORAGE_KEY = 'ingress_admin_terminal_tabs_v2'
const RECONNECT_GRACE_MS = 60_000
const RECONNECT_BASE_MS = 300
const RECONNECT_MAX_MS = 5_000
const SESSION_HANDSHAKE_MS = 8_000

type TerminalSessionMsg = {
  type: 'session'
  id: string
  reattach: boolean
}

function terminalWebSocketURL(sessionId?: string | null) {
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const base = `${proto}//${window.location.host}/api/v1/terminal/ws`
  const id = sessionId?.trim()
  if (!id) return base
  return `${base}?session=${encodeURIComponent(id)}`
}

function readStoredSessionId() {
  try {
    return sessionStorage.getItem(SESSION_STORAGE_KEY)
  } catch {
    return null
  }
}

function storeSessionId(id: string) {
  try {
    sessionStorage.setItem(SESSION_STORAGE_KEY, id)
  } catch {
    /* ignore */
  }
}

function clearStoredSessionId() {
  try {
    sessionStorage.removeItem(SESSION_STORAGE_KEY)
    sessionStorage.removeItem(LEGACY_TABS_STORAGE_KEY)
  } catch {
    /* ignore */
  }
}

export function WebTerminal() {
  const screenRef = useRef<HTMLDivElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const termRef = useRef<Terminal | null>(null)
  const fitRef = useRef<FitAddon | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const wsGenRef = useRef(0)
  const sessionReadyRef = useRef(false)
  const handshakeTimerRef = useRef(0)
  const lastDimsRef = useRef<{ rows: number; cols: number } | null>(null)
  const fitFrameRef = useRef(0)
  const reconnectTimerRef = useRef(0)
  const reconnectStartedRef = useRef(0)
  const reconnectAttemptRef = useRef(0)
  const manualCloseRef = useRef(false)
  const newSessionRef = useRef(false)
  const leavingPageRef = useRef(false)
  const [connState, setConnState] = useState<ConnState>('idle')

  const clearReconnectTimer = useCallback(() => {
    if (reconnectTimerRef.current) {
      window.clearTimeout(reconnectTimerRef.current)
      reconnectTimerRef.current = 0
    }
  }, [])

  const clearHandshakeTimer = useCallback(() => {
    if (handshakeTimerRef.current) {
      window.clearTimeout(handshakeTimerRef.current)
      handshakeTimerRef.current = 0
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
    if (ws && ws.readyState === WebSocket.OPEN && sessionReadyRef.current) {
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
      clearHandshakeTimer()
      manualCloseRef.current = false
      if (!opts?.auto) {
        reconnectStartedRef.current = 0
        reconnectAttemptRef.current = 0
      }
      sessionReadyRef.current = false

      if (opts?.forceNew) {
        newSessionRef.current = true
        clearStoredSessionId()
      }

      const prev = wsRef.current
      if (prev) {
        prev.onopen = null
        prev.onmessage = null
        prev.onclose = null
        prev.onerror = null
        prev.close()
        wsRef.current = null
      }

      const gen = ++wsGenRef.current
      setConnState(opts?.auto ? 'reconnecting' : 'connecting')

      const sessionId = newSessionRef.current ? null : readStoredSessionId()
      newSessionRef.current = false

      const ws = new WebSocket(terminalWebSocketURL(sessionId))
      ws.binaryType = 'arraybuffer'
      wsRef.current = ws

      handshakeTimerRef.current = window.setTimeout(() => {
        if (wsGenRef.current !== gen || sessionReadyRef.current) return
        ws.close()
        setConnState('error')
        term.writeln('\r\n\x1b[31m[终端握手超时，请重新连接]\x1b[0m')
      }, SESSION_HANDSHAKE_MS)

      ws.onopen = () => {
        if (wsGenRef.current !== gen) return
        scheduleFit(true)
      }

      ws.onmessage = (ev) => {
        if (wsGenRef.current !== gen) return
        if (typeof ev.data === 'string') {
          try {
            const msg = JSON.parse(ev.data) as TerminalSessionMsg
            if (msg.type === 'session' && msg.id) {
              clearHandshakeTimer()
              sessionReadyRef.current = true
              storeSessionId(msg.id)
              setConnState('open')
              if (!msg.reattach) {
                term.reset()
              }
              scheduleFit(true)
              return
            }
          } catch {
            /* shell text */
          }
          if (sessionReadyRef.current) {
            term.write(ev.data)
          }
          return
        }
        if (sessionReadyRef.current) {
          term.write(new Uint8Array(ev.data))
        }
      }

      ws.onclose = () => {
        if (wsGenRef.current !== gen) return
        clearHandshakeTimer()
        sessionReadyRef.current = false
        if (manualCloseRef.current || leavingPageRef.current) {
          setConnState('closed')
          return
        }

        const started = reconnectStartedRef.current
        if (started === 0) {
          reconnectStartedRef.current = Date.now()
        } else if (Date.now() - started >= RECONNECT_GRACE_MS) {
          setConnState('closed')
          term.writeln('\r\n\x1b[90m[会话已过期，请新建终端或重新连接]\x1b[0m')
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
        sessionReadyRef.current = false
        clearHandshakeTimer()
        setConnState('error')
      }
    },
    [clearHandshakeTimer, clearReconnectTimer, scheduleFit],
  )

  const connectRef = useRef(connect)
  connectRef.current = connect

  useEffect(() => {
    const mount = containerRef.current
    const screen = screenRef.current
    if (!mount || !screen) return

    leavingPageRef.current = false

    try {
      sessionStorage.removeItem(LEGACY_TABS_STORAGE_KEY)
    } catch {
      /* ignore */
    }

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

    const fitWhenReady = () => scheduleFit(true)
    if (document.fonts?.ready) {
      void document.fonts.ready.then(fitWhenReady)
    } else {
      fitWhenReady()
    }

    term.onData((data) => {
      const ws = wsRef.current
      if (ws && ws.readyState === WebSocket.OPEN && sessionReadyRef.current) {
        ws.send(data)
      }
    })

    connectRef.current()

    const onResize = () => scheduleFit()
    window.addEventListener('resize', onResize)
    const ro = new ResizeObserver(onResize)
    ro.observe(screen)

    const onPageHide = () => {
      leavingPageRef.current = true
      clearReconnectTimer()
      clearHandshakeTimer()
      wsRef.current?.close()
    }
    window.addEventListener('pagehide', onPageHide)

    return () => {
      leavingPageRef.current = true
      manualCloseRef.current = true
      wsGenRef.current += 1
      clearReconnectTimer()
      clearHandshakeTimer()
      cancelAnimationFrame(fitFrameRef.current)
      window.removeEventListener('resize', onResize)
      window.removeEventListener('pagehide', onPageHide)
      ro.disconnect()
      const ws = wsRef.current
      if (ws) {
        ws.onopen = null
        ws.onmessage = null
        ws.onclose = null
        ws.onerror = null
        ws.close()
      }
      wsRef.current = null
      term.dispose()
      termRef.current = null
      fitRef.current = null
    }
    // Mount terminal once; connect uses refs for latest logic.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const connLabel =
    connState === 'open'
      ? '已连接'
      : connState === 'connecting'
        ? '连接中…'
        : connState === 'reconnecting'
          ? '重连中…'
          : connState === 'error'
            ? '连接失败'
            : connState === 'closed'
              ? '已断开'
              : '未连接'

  return (
    <div className="web-terminal">
      <div className="web-terminal-toolbar">
        <span className={`web-terminal-status web-terminal-status--${connState}`}>{connLabel}</span>
        <div className="web-terminal-actions">
          <button
            type="button"
            className="btn btn-sm"
            onClick={() => connect({ forceNew: true })}
            disabled={connState === 'connecting' || connState === 'reconnecting'}
          >
            <Plus size={14} aria-hidden />
            新建终端
          </button>
          <button
            type="button"
            className="btn btn-sm"
            onClick={() => connect()}
            disabled={connState === 'connecting' || connState === 'reconnecting'}
          >
            <RefreshCw size={14} aria-hidden />
            重新连接
          </button>
        </div>
      </div>
      <div className="web-terminal-screen" ref={screenRef}>
        <div className="web-terminal-mount" ref={containerRef} />
      </div>
    </div>
  )
}
