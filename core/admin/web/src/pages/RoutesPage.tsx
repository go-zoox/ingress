import { useEffect, useMemo, useState } from 'react'
import { PageHeader } from '../components/PageHeader'
import { HostBadge } from '../components/HostBadge'
import { api, type MatchPreview, type RouteRow } from '../api/client'

function groupByHost(rows: RouteRow[]): { host: string; type: string; rows: RouteRow[] }[] {
  const m = new Map<string, { type: string; rows: RouteRow[] }>()
  for (const r of rows) {
    if (!m.has(r.host)) m.set(r.host, { type: r.host_type, rows: [] })
    m.get(r.host)!.rows.push(r)
  }
  return Array.from(m.entries()).map(([host, v]) => ({ host, ...v }))
}

export function RoutesPage() {
  const [rows, setRows] = useState<RouteRow[]>([])
  const [filter, setFilter] = useState('')
  const [urlInput, setUrlInput] = useState('https://api.example.com/v2/users')
  const [match, setMatch] = useState<MatchPreview | null>(null)
  const [err, setErr] = useState('')
  const [expandedHosts, setExpandedHosts] = useState<Set<string>>(new Set())

  useEffect(() => {
    api
      .routes()
      .then((data) => setRows(Array.isArray(data) ? data : []))
      .catch((e: Error) => setErr(e.message))
  }, [])

  const q = filter.toLowerCase()
  const filtered = rows.filter((r) =>
    `${r.host} ${r.path} ${r.target} ${r.backend_type}`.toLowerCase().includes(q),
  )

  const runMatch = () => {
    setErr('')
    setMatch(null)
    try {
      const u = new URL(urlInput)
      api.match(u.hostname, u.pathname).then(setMatch).catch((e: Error) => setErr(e.message))
    } catch {
      setErr('请输入合法的 URL，例如 https://api.example.com/v2/users')
    }
  }

  const toggleHost = (host: string) => {
    setExpandedHosts((prev) => {
      const next = new Set(prev)
      if (next.has(host)) next.delete(host)
      else next.add(host)
      return next
    })
  }

  const groups = useMemo(() => groupByHost(filtered), [filtered])

  return (
    <div className="page">
      <PageHeader title="路由" desc="编译后的 host/path 规则表（含 host_type 推断结果）" />
      {err && <p className="err">{err}</p>}
      <div className="grid-2">
        <div className="panel">
          <div className="panel-head">
            <h2>规则列表</h2>
            <input
              type="search"
              placeholder="过滤 host / target…"
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
            />
          </div>
          <div className="panel-body panel-table-wrap">
            {groups.length === 0 ? (
              <p className="empty-hint">无匹配规则</p>
            ) : (
              <div className="route-accordion-list">
                {groups.map(({ host, type, rows: hostRows }) => {
                  const expanded = expandedHosts.has(host)
                  return (
                    <div key={host} className="route-accordion">
                      <button
                        type="button"
                        className="route-accordion-header"
                        onClick={() => toggleHost(host)}
                      >
                        <span className="route-chevron">{expanded ? '▼' : '▶'}</span>
                        <code>{host}</code>
                        <HostBadge t={type} />
                        <span className="route-count">{hostRows.length} path{hostRows.length > 1 ? 's' : ''}</span>
                      </button>
                      {expanded && (
                        <div className="route-accordion-body">
                          <table className="data">
                            <thead>
                              <tr>
                                <th>Path</th>
                                <th>Backend</th>
                                <th>目标</th>
                              </tr>
                            </thead>
                            <tbody>
                              {hostRows.map((r) => (
                                <tr key={r.id}>
                                  <td>{r.path}</td>
                                  <td>{r.backend_type}</td>
                                  <td><code>{r.target}</code></td>
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        </div>
                      )}
                    </div>
                  )
                })}
              </div>
            )}
          </div>
        </div>
        <div className="panel">
          <div className="panel-head">
            <h2>试匹配</h2>
          </div>
          <div className="panel-body">
            <p className="match-hint">输入完整 URL，自动提取 Host 与 Path 进行匹配。</p>
            <label className="field-label">URL</label>
            <input
              type="text"
              className="field-input-last"
              placeholder="https://api.example.com/v2/users"
              value={urlInput}
              onChange={(e) => setUrlInput(e.target.value)}
              onKeyDown={(e) => { if (e.key === 'Enter') runMatch() }}
            />
            <button type="button" className="btn btn-primary" style={{ width: '100%' }} onClick={runMatch}>
              试匹配
            </button>
            {match && (
              <div className={`match-result ${match.matched ? 'hit' : 'miss'}`}>
                {match.matched ? (
                  <>
                    <h3>命中规则 #{match.rule_index}</h3>
                    <dl>
                      <dt>Host</dt>
                      <dd>
                        {match.host} ({match.host_type})
                      </dd>
                      <dt>Path</dt>
                      <dd>{match.path}</dd>
                      <dt>Backend</dt>
                      <dd>{match.backend_type}</dd>
                      <dt>目标</dt>
                      <dd>
                        <code>{match.target}</code>
                      </dd>
                    </dl>
                  </>
                ) : (
                  <>
                    <h3>未命中</h3>
                    <p>{match.message || '将走 fallback 或返回 404'}</p>
                  </>
                )}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
