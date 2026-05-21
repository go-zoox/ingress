import { useEffect, useState } from 'react'
import { PageHeader } from '../components/PageHeader'
import { api } from '../api/client'

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
  const [lines, setLines] = useState<string[]>([])
  const [count, setCount] = useState('—')
  const [err, setErr] = useState('')

  const search = () => {
    setErr('')
    api
      .logs({
        log: logKind,
        q,
        host,
        status: logKind === 'access' ? status : undefined,
      })
      .then((r) => {
        const list = Array.isArray(r.lines) ? r.lines : []
        setLines(list)
        setCount(`${list.length} 条`)
      })
      .catch((e: Error) => setErr(e.message))
  }

  const clear = () => {
    setQ('')
    setHost('')
    setStatus('')
    search()
  }

  useEffect(() => {
    search()
  }, [logKind])

  return (
    <div className="page">
      <PageHeader
        title="日志"
        desc={
          logKind === 'access'
            ? '访问日志：按 host、状态码、关键字过滤'
            : '错误日志：按 host、关键字过滤'
        }
      />
      {err && <p className="err">{err}</p>}
      <div className="panel">
        <div className="panel-head">
          <h2>查询</h2>
        </div>
        <div className="panel-body toolbar">
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
            style={{ minWidth: 160 }}
            value={q}
            onChange={(e) => setQ(e.target.value)}
          />
          <input
            type="text"
            placeholder="Host"
            style={{ width: 160 }}
            value={host}
            onChange={(e) => setHost(e.target.value)}
          />
          {logKind === 'access' ? (
            <select value={status} onChange={(e) => setStatus(e.target.value)}>
              <option value="">全部状态</option>
              <option value="2">2xx</option>
              <option value="3">3xx</option>
              <option value="4">4xx</option>
              <option value="5">5xx</option>
            </select>
          ) : null}
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
          <h2>结果</h2>
          <span className="log-count">{count}</span>
        </div>
        <div className="panel-body panel-table-wrap">
          <div className="log-lines">
            {lines.length === 0 ? (
              <div className="empty-hint">无匹配日志</div>
            ) : (
              lines.map((line, i) => (
                <div key={i} className={`log-line ${logLineClass(line)}`}>
                  {line}
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
