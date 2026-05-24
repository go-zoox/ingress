import { useCallback, useEffect, useRef, useState } from 'react'
import { PageHeader } from '../components/PageHeader'
import { api } from '../api/client'
import { loadPreferences } from '../lib/preferences'

const REFRESH_OPTIONS = [
  { value: 0, label: '关闭' },
  { value: 1000, label: '1 秒' },
  { value: 2000, label: '2 秒' },
  { value: 5000, label: '5 秒' },
  { value: 10000, label: '10 秒' },
  { value: 30000, label: '30 秒' },
]

const MAX_LINES = 500

function logLineClass(line: string) {
  const m = line.match(/"\s+(\d{3})\s/)
  if (!m) return ''
  const c = m[1][0]
  if (c === '4' || c === '5') return 'status-4xx'
  if (c === '2') return 'status-2xx'
  return ''
}

export function LogsPage() {
  const [logKind, setLogKind] = useState<'access' | 'error'>('access')
  const [q, setQ] = useState('')
  const [host, setHost] = useState('')
  const [status, setStatus] = useState('')
  const [cacheHit, setCacheHit] = useState('')
  const [wafBlock, setWafBlock] = useState('')
  const [live, setLive] = useState(false)
  const [intervalMs, setIntervalMs] = useState(() => loadPreferences().logLiveIntervalMs)
  const [lines, setLines] = useState<string[]>([])
  const [count, setCount] = useState('—')
  const [err, setErr] = useState('')
  const [lastRefresh, setLastRefresh] = useState('')
  const [logHosts, setLogHosts] = useState<string[]>([])
  const offsetRef = useRef(0)
  const logEndRef = useRef<HTMLDivElement>(null)
  const filtersRef = useRef({ logKind, q, host, status, cacheHit, wafBlock })

  filtersRef.current = { logKind, q, host, status, cacheHit, wafBlock }

  const buildParams = useCallback(
    (offset: number) => ({
      log: filtersRef.current.logKind,
      q: filtersRef.current.q || undefined,
      host: filtersRef.current.host || undefined,
      status:
        filtersRef.current.logKind === 'access' ? filtersRef.current.status || undefined : undefined,
      cache_hit:
        filtersRef.current.logKind === 'access'
          ? filtersRef.current.cacheHit || undefined
          : undefined,
      waf_block:
        filtersRef.current.logKind === 'access'
          ? filtersRef.current.wafBlock || undefined
          : undefined,
      offset,
      limit: 200,
    }),
    [],
  )

  const fetchLogs = useCallback(
    async (incremental: boolean) => {
      setErr('')
      try {
        const offset = incremental ? offsetRef.current : 0
        const r = await api.logs(buildParams(offset))
        const list = Array.isArray(r.lines) ? r.lines : []
        offsetRef.current = r.offset ?? offsetRef.current
        if (incremental && offset > 0) {
          setLines((prev) => {
            const merged = [...prev, ...list]
            return merged.length > MAX_LINES ? merged.slice(-MAX_LINES) : merged
          })
          setCount((prev) => {
            const n = parseInt(prev, 10)
            return `${(Number.isNaN(n) ? 0 : n) + list.length} 条`
          })
        } else {
          setLines(list)
          setCount(`${list.length} 条`)
        }
        setLastRefresh(new Date().toLocaleTimeString('zh-CN', { hour12: false }))
      } catch (e) {
        setErr((e as Error).message)
      }
    },
    [buildParams],
  )

  const search = () => {
    offsetRef.current = 0
    fetchLogs(false)
  }

  const clear = () => {
    setQ('')
    setHost('')
    setStatus('')
    setCacheHit('')
    setWafBlock('')
    offsetRef.current = 0
    setTimeout(() => fetchLogs(false), 0)
  }

  useEffect(() => {
    offsetRef.current = 0
    fetchLogs(false)
  }, [logKind, fetchLogs])

  // fetch distinct hosts for filter dropdown
  useEffect(() => {
    api.logHosts().then(setLogHosts).catch(() => setLogHosts([]))
  }, [])

  useEffect(() => {
    if (!live || intervalMs <= 0) return
    const id = window.setInterval(() => fetchLogs(true), intervalMs)
    return () => window.clearInterval(id)
  }, [live, intervalMs, fetchLogs, logKind, q, host, status, cacheHit, wafBlock])

  useEffect(() => {
    if (live && logEndRef.current) {
      logEndRef.current.scrollIntoView({ behavior: 'smooth' })
    }
  }, [lines, live])

  return (
    <div className="page">
      <PageHeader
        title="日志"
        desc={
          logKind === 'access'
            ? '访问日志：实时 tail、条件监控与过滤'
            : '错误日志：实时 tail 与关键字过滤'
        }
      />
      {err && <p className="err">{err}</p>}
      <div className="panel">
        <div className="panel-head">
          <h2>查询 / 监控</h2>
          {lastRefresh ? <span className="chart-hint">上次刷新 {lastRefresh}</span> : null}
        </div>
        <div className="panel-body toolbar logs-toolbar">
          <select
            value={logKind}
            onChange={(e) => setLogKind(e.target.value as 'access' | 'error')}
          >
            <option value="access">访问日志</option>
            <option value="error">错误日志</option>
          </select>
          <input
            type="search"
            placeholder="关键字…"
            style={{ minWidth: 140 }}
            value={q}
            onChange={(e) => setQ(e.target.value)}
          />
          <select
            value={host}
            onChange={(e) => setHost(e.target.value)}
            style={{ minWidth: 160 }}
          >
            <option value="">全部 Host</option>
            {logHosts.map((h) => (
              <option key={h} value={h}>{h}</option>
            ))}
          </select>
          {logKind === 'access' ? (
            <>
              <select value={status} onChange={(e) => setStatus(e.target.value)}>
                <option value="">全部状态</option>
                <option value="2">2xx</option>
                <option value="3">3xx</option>
                <option value="4">4xx</option>
                <option value="5">5xx</option>
              </select>
              <select value={cacheHit} onChange={(e) => setCacheHit(e.target.value)}>
                <option value="">缓存不限</option>
                <option value="1">cache_hit=1</option>
                <option value="0">cache_hit=0</option>
              </select>
              <select value={wafBlock} onChange={(e) => setWafBlock(e.target.value)}>
                <option value="">WAF 不限</option>
                <option value="1">waf_block=1</option>
                <option value="0">waf_block=0</option>
              </select>
            </>
          ) : null}
          <label className="live-toggle">
            <input type="checkbox" checked={live} onChange={(e) => setLive(e.target.checked)} />
            实时
          </label>
          <select
            value={intervalMs}
            disabled={!live}
            onChange={(e) => setIntervalMs(Number(e.target.value))}
            title="刷新频率"
          >
            {REFRESH_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
          <button type="button" className="btn btn-primary" onClick={search}>
            查询
          </button>
          <button type="button" className="btn btn-ghost" onClick={clear}>
            清空
          </button>
        </div>
      </div>
      <div className="panel">
        <div className="panel-head">
          <h2>{live ? '实时日志' : '结果'}</h2>
          <span className="log-count">
            {count}
            {live && intervalMs > 0 ? ` · 每 ${intervalMs / 1000}s` : ''}
          </span>
        </div>
        <div className="panel-body panel-table-wrap">
          <div className="log-lines log-lines-live">
            {lines.length === 0 ? (
              <div className="empty-hint">无匹配日志</div>
            ) : (
              lines.map((line, i) => (
                <div key={`${i}-${line.slice(0, 40)}`} className={`log-line ${logLineClass(line)}`}>
                  {line}
                </div>
              ))
            )}
            <div ref={logEndRef} />
          </div>
        </div>
      </div>
    </div>
  )
}
