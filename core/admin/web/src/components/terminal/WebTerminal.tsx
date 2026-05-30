import { useCallback, useEffect, useRef, useState } from 'react'
import { RefreshCw } from 'lucide-react'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'

type ConnState = 'idle' | 'connecting' | 'open' | 'closed' | 'error'

function terminalWebSocketURL() {
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${proto}//${window.location.host}/api/v1/terminal/ws`
}

export function WebTerminal() {
  const screenRef = useRef<HTMLDivElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const termRef = useRef<Terminal | null>(null)
  const fitRef = useRef<FitAddon | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const sessionRef = useRef(0)
  const lastDimsRef = useRef<{ rows: number; cols: number } | null>(null)
  const fitFrameRef = useRef(0)
  const [connState, setConnState] = useState<ConnState>('idle')

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

  const connect = useCallback(() => {
    const term = termRef.current
    if (!term) return

    wsRef.current?.close()
    const session = ++sessionRef.current
    setConnState('connecting')

    const ws = new WebSocket(terminalWebSocketURL())
    ws.binaryType = 'arraybuffer'
    wsRef.current = ws

    ws.onopen = () => {
      if (sessionRef.current !== session) return
      setConnState('open')
      lastDimsRef.current = null
      term.reset()
      scheduleFit(true)
    }

    ws.onmessage = (ev) => {
      if (sessionRef.current !== session) return
      if (typeof ev.data === 'string') {
        term.write(ev.data)
      } else {
        term.write(new Uint8Array(ev.data))
      }
    }

    ws.onclose = () => {
      if (sessionRef.current !== session) return
      setConnState('closed')
      term.writeln('\r\n\x1b[90m[连接已断开]\x1b[0m')
    }

    ws.onerror = () => {
      if (sessionRef.current !== session) return
      setConnState('error')
      term.writeln('\r\n\x1b[31m[连接失败]\x1b[0m')
    }
  }, [scheduleFit])

  useEffect(() => {
    const mount = containerRef.current
    const screen = screenRef.current
    if (!mount || !screen) return

    const term = new Terminal({
      cursorBlink: true,
      fontSize: 13,
      lineHeight: 1.2,
      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
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
    applyFit(true)

    termRef.current = term
    fitRef.current = fit

    term.onData((data) => {
      const ws = wsRef.current
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(data)
      }
    })

    connect()

    const onResize = () => scheduleFit()
    window.addEventListener('resize', onResize)
    const ro = new ResizeObserver(onResize)
    ro.observe(screen)

    return () => {
      sessionRef.current += 1
      cancelAnimationFrame(fitFrameRef.current)
      window.removeEventListener('resize', onResize)
      ro.disconnect()
      wsRef.current?.close()
      wsRef.current = null
      term.dispose()
      termRef.current = null
      fitRef.current = null
    }
  }, [applyFit, connect, scheduleFit])

  const connLabel =
    connState === 'open'
      ? '已连接'
      : connState === 'connecting'
        ? '连接中…'
        : connState === 'error'
          ? '连接失败'
          : connState === 'closed'
            ? '已断开'
            : '未连接'

  return (
    <div className="web-terminal">
      <div className="web-terminal-toolbar">
        <span className={`web-terminal-status web-terminal-status--${connState}`}>{connLabel}</span>
        <button type="button" className="btn btn-sm" onClick={connect} disabled={connState === 'connecting'}>
          <RefreshCw size={14} aria-hidden />
          重新连接
        </button>
      </div>
      <div className="web-terminal-screen" ref={screenRef}>
        <div className="web-terminal-mount" ref={containerRef} />
      </div>
    </div>
  )
}
