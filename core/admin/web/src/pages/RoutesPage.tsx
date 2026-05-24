import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { PageHeader } from '../components/PageHeader'
import { HostBadge } from '../components/HostBadge'
import { api, type MatchPreview, type RouteRow } from '../api/client'

interface HostGroup {
  host: string
  type: string
  rows: RouteRow[]
}

function groupByHost(rows: RouteRow[]): HostGroup[] {
  const m = new Map<string, { type: string; rows: RouteRow[] }>()
  for (const r of rows) {
    if (!m.has(r.host)) m.set(r.host, { type: r.host_type, rows: [] })
    m.get(r.host)!.rows.push(r)
  }
  return Array.from(m.entries()).map(([host, v]) => ({ host, ...v }))
}

export function RoutesPage() {
  const navigate = useNavigate()
  const [rows, setRows] = useState<RouteRow[]>([])
  const [filter, setFilter] = useState('')
  const [urlInput, setUrlInput] = useState('https://api.example.com/v2/users')
  const [match, setMatch] = useState<MatchPreview | null>(null)
  const [matchError, setMatchError] = useState('')
  const [expandedHosts, setExpandedHosts] = useState<Set<string>>(new Set())

  useEffect(() => {
    api
      .routes()
      .then((data) => setRows(Array.isArray(data) ? data : []))
      .catch(() => setRows([]))
  }, [])

  // derived: which host, rule index & path index are matched
  const matchedHost = match?.matched ? match.host : null
  const matchedRuleIndex = match?.matched ? match.rule_index : null
  const matchedPathIndex: number | null = match?.matched ? (match.path_index ?? null) : null

  // auto-expand matched host
  useEffect(() => {
    if (matchedHost && !expandedHosts.has(matchedHost)) {
      setExpandedHosts((prev) => new Set(prev).add(matchedHost))
    }
  }, [matchedHost])

  const q = filter.toLowerCase()
  const filtered = rows.filter((r) =>
    `${r.host} ${r.path} ${r.target} ${r.backend_type}`.toLowerCase().includes(q),
  )

  const runMatch = () => {
    setMatchError('')
    setMatch(null)
    try {
      const u = new URL(urlInput)
      api
        .match(u.hostname, u.pathname)
        .then((m) => setMatch(m))
        .catch((e: Error) => setMatchError(e.message))
    } catch {
      setMatchError('请输入合法的 URL，例如 https://api.example.com/v2/users')
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
      {matchError && <p className="err">{matchError}</p>}

      <div className="grid-2">
        {/* ---- left: route list ---- */}
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
                        <span className="route-count">
                          {hostRows.length} path{hostRows.length > 1 ? 's' : ''}
                        </span>
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
                              {hostRows.map((r) => {
                                const isHostRow = r.path_index === -1
                                const isMatched =
                                  match?.matched &&
                                  r.rule_index === matchedRuleIndex &&
                                  (isHostRow
                                    ? (matchedPathIndex == null || matchedPathIndex < 0)
                                    : r.path_index === matchedPathIndex)
                                return (
                                  <tr
                                    key={r.id}
                                    className={`${isMatched ? 'match-highlight' : ''} route-row-clickable`}
                                    onClick={() => {
                                      if (!isHostRow) {
                                        navigate(`/routes/${r.rule_index}/${r.path_index}`)
                                      }
                                    }}
                                    style={{ cursor: isHostRow ? 'default' : 'pointer' }}
                                  >
                                    <td>{r.path}</td>
                                    <td>{r.backend_type}</td>
                                    <td>
                                      <code>{r.target}</code>
                                    </td>
                                  </tr>
                                )
                              })}
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

        {/* ---- right: match input + result ---- */}
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
            <button
              type="button"
              className="btn btn-primary"
              style={{ width: '100%' }}
              onClick={runMatch}
            >
              试匹配
            </button>

            {/* inline match result */}
            {match && (
              <div className={`match-result ${match.matched ? 'hit' : 'miss'}`}>
                {match.matched ? (
                  <>
                    <h3>
                      命中规则 #{match.rule_index}
                      {match.fallback && '（fallback）'}
                    </h3>
                    <dl>
                      <dt>Host</dt>
                      <dd>
                        {match.host}（{match.host_type}）
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
                    <p className="match-hint" style={{ marginTop: 8 }}>
                      已在左侧列表中高亮对应规则。
                    </p>
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
