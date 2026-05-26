import { useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { HostBadge } from '../HostBadge'
import type { MatchPreview, RouteRow } from '../../api/client'

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

type Props = {
  rows: RouteRow[]
  filter: string
  onFilterChange: (v: string) => void
  expandedHosts: Set<string>
  onToggleHost: (host: string) => void
  match: MatchPreview | null
  highlightHost?: string
}

export function RouteListTab({
  rows,
  filter,
  onFilterChange,
  expandedHosts,
  onToggleHost,
  match,
  highlightHost,
}: Props) {
  const navigate = useNavigate()
  const q = filter.toLowerCase()
  const filtered = rows.filter((r) => {
    if (highlightHost && r.host !== highlightHost) return false
    return `${r.host} ${r.path} ${r.target} ${r.backend_type}`.toLowerCase().includes(q)
  })

  const matchedRuleIndex = match?.matched ? match.rule_index : null
  const matchedPathIndex: number | null = match?.matched ? (match.path_index ?? null) : null
  const groups = useMemo(() => groupByHost(filtered), [filtered])

  return (
    <div className="panel">
      <div className="panel-head">
        <h2>规则列表</h2>
        <input
          type="search"
          placeholder="过滤 host / target…"
          value={filter}
          onChange={(e) => onFilterChange(e.target.value)}
        />
      </div>
      <div className="panel-body panel-table-wrap">
        {groups.length === 0 ? (
          <p className="empty-hint">无匹配规则</p>
        ) : (
          <div className="route-accordion-list">
            {groups.map(({ host, type, rows: hostRows }) => {
              const expanded = expandedHosts.has(host) || Boolean(highlightHost)
              return (
                <div key={host} className="route-accordion">
                  <button type="button" className="route-accordion-header" onClick={() => onToggleHost(host)}>
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
                            const isMatched =
                              match?.matched &&
                              r.rule_index === matchedRuleIndex &&
                              r.path_index === matchedPathIndex
                            return (
                              <tr
                                key={r.id}
                                className={`${isMatched ? 'match-highlight' : ''} route-row-clickable`}
                                onClick={() => navigate(`/routes/${r.rule_index}/${r.path_index}`)}
                                style={{ cursor: 'pointer' }}
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
  )
}
