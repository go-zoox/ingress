import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { PageHeader } from '../components/PageHeader'
import { RouteRulesManageTab } from '../components/routes/RouteRulesManageTab'
import { RouteMatchTab } from '../components/routes/RouteMatchTab'
import { RouteTopologyTab } from '../components/routes/RouteTopologyTab'
import { ToastContainer, useToast } from '../components/Toast'
import { api, type MatchPreview } from '../api/client'

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

  const [urlInput, setUrlInput] = useState('https://api.example.com/v2/users')
  const [match, setMatch] = useState<MatchPreview | null>(null)
  const [matchError, setMatchError] = useState('')
  const { toast, show, clear } = useToast()

  const reloadRows = () => {
    api.routes().catch(() => [])
  }

  useEffect(() => {
    reloadRows()
  }, [])

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

  const tabDesc = useMemo(() => {
    switch (tab) {
      case 'topology':
        return 'Host → Path → Backend 关系图；点击节点进入路由详情'
      case 'match':
        return '输入 URL 验证命中规则，可跳转路由详情'
      default:
        return '路由规则增删改查；保存或发布 reload 后生效，与配置中心 rules 模块同步'
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
        <RouteRulesManageTab
          onPublished={reloadRows}
          onSaveSuccess={(msg) => show(msg)}
          onSaveError={(msg) => show(msg, 'error')}
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
      {toast && <ToastContainer message={toast.message} type={toast.type} onDone={clear} />}
    </div>
  )
}
