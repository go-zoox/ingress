import { useEffect, useState } from 'react'
import { PageHeader } from '../components/PageHeader'
import { HostBadge } from '../components/HostBadge'
import { api, type MatchPreview, type RouteRow } from '../api/client'

export function RoutesPage() {
  const [rows, setRows] = useState<RouteRow[]>([])
  const [filter, setFilter] = useState('')
  const [host, setHost] = useState('api.example.com')
  const [path, setPath] = useState('/v2/users')
  const [match, setMatch] = useState<MatchPreview | null>(null)
  const [err, setErr] = useState('')

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
    api.match(host, path).then(setMatch).catch((e: Error) => setErr(e.message))
  }

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
            <table className="data">
              <thead>
                <tr>
                  <th>Host</th>
                  <th>类型</th>
                  <th>Path</th>
                  <th>Backend</th>
                  <th>目标</th>
                </tr>
              </thead>
              <tbody>
                {filtered.length === 0 ? (
                  <tr>
                    <td colSpan={5} className="empty-hint">
                      无匹配规则
                    </td>
                  </tr>
                ) : (
                  filtered.map((r) => (
                    <tr key={r.id}>
                      <td>
                        <code>{r.host}</code>
                      </td>
                      <td>
                        <HostBadge t={r.host_type} />
                      </td>
                      <td>{r.path}</td>
                      <td>{r.backend_type}</td>
                      <td>
                        <code>{r.target}</code>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>
        <div className="panel">
          <div className="panel-head">
            <h2>试匹配</h2>
          </div>
          <div className="panel-body">
            <p className="match-hint">输入请求 Host 与 Path，预览将命中的规则。</p>
            <label className="field-label">Host</label>
            <input
              type="text"
              className="field-input"
              value={host}
              onChange={(e) => setHost(e.target.value)}
            />
            <label className="field-label">Path</label>
            <input
              type="text"
              className="field-input-last"
              value={path}
              onChange={(e) => setPath(e.target.value)}
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
