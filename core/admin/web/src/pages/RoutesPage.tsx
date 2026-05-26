import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { PageHeader } from '../components/PageHeader'
import { RouteListTab } from '../components/routes/RouteListTab'
import { RouteMatchTab } from '../components/routes/RouteMatchTab'
import { RouteTopologyTab } from '../components/routes/RouteTopologyTab'
import { api, type MatchPreview, type RouteRow } from '../api/client'

const TABS = [
  { id: 'list', label: '列表' },
  { id: 'topology', label: '拓扑' },
  { id: 'match', label: '试匹配' },
] as const

type TabId = (typeof TABS)[number]['id']

function parseTab(raw: string | null): TabId {
  if (raw === 'topology' || raw === 'match') return raw
  return 'list'
}

export function RoutesPage() {
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const tab = parseTab(searchParams.get('tab'))
  const highlightRi = searchParams.get('ri') ? Number(searchParams.get('ri')) : undefined
  const highlightPi = searchParams.get('pi') ? Number(searchParams.get('pi')) : undefined
  const highlightHost = searchParams.get('host') || undefined

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

  useEffect(() => {
    if (highlightHost) {
      setExpandedHosts((prev) => new Set(prev).add(highlightHost))
    }
  }, [highlightHost])

  const matchedHost = match?.matched ? match.host : null
  useEffect(() => {
    if (matchedHost) {
      setExpandedHosts((prev) => new Set(prev).add(matchedHost))
    }
  }, [matchedHost])

  const setTab = (id: TabId) => {
    const next = new URLSearchParams(searchParams)
    next.set('tab', id)
    setSearchParams(next, { replace: true })
  }

  const runMatch = () => {
    setMatchError('')
    setMatch(null)
    try {
      const u = new URL(urlInput)
      api
        .match(u.hostname, u.pathname)
        .then((m) => {
          setMatch(m)
          if (m.matched) setTab('list')
        })
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

  const tabDesc = useMemo(() => {
    switch (tab) {
      case 'topology':
        return 'Host → Path → Backend 关系图；点击节点进入路由详情'
      case 'match':
        return '输入 URL 验证命中规则，可跳转路由详情'
      default:
        return '按 Host 分组的路由表；点击进入单路由观测（日志、WAF、缓存）'
    }
  }, [tab])

  return (
    <div className="page">
      <PageHeader title="路由" desc={tabDesc} />

      <div className="page-tabs route-page-tabs" role="tablist" aria-label="路由视图">
        {TABS.map((t) => (
          <button
            key={t.id}
            type="button"
            role="tab"
            aria-selected={tab === t.id}
            className={tab === t.id ? 'btn btn-sm active' : 'btn btn-sm btn-ghost'}
            onClick={() => setTab(t.id)}
          >
            {t.label}
          </button>
        ))}
      </div>

      {tab === 'list' ? (
        <RouteListTab
          rows={rows}
          filter={filter}
          onFilterChange={setFilter}
          expandedHosts={expandedHosts}
          onToggleHost={toggleHost}
          match={match}
          highlightHost={highlightHost}
        />
      ) : null}

      {tab === 'topology' ? (
        <RouteTopologyTab highlightRi={highlightRi} highlightPi={highlightPi} />
      ) : null}

      {tab === 'match' ? (
        <RouteMatchTab
          urlInput={urlInput}
          onUrlInputChange={setUrlInput}
          onMatch={runMatch}
          match={match}
          matchError={matchError}
          onOpenRoute={(ri, pi) => navigate(`/routes/${ri}/${pi}`)}
        />
      ) : null}
    </div>
  )
}
