import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { Copy, Network, RefreshCw, ScrollText, Search, Settings2, Shield } from 'lucide-react'
import { PageHeader } from '../components/PageHeader'
import { InvestigateHistoryPanel } from '../components/investigate/InvestigateHistoryPanel'
import { InvestigateLatencyBar } from '../components/investigate/InvestigateLatencyBar'
import { InvestigateMatchPanel } from '../components/investigate/InvestigateMatchPanel'
import { InvestigatePolicyPanel } from '../components/investigate/InvestigatePolicyPanel'
import { InvestigateRouteMetrics } from '../components/investigate/InvestigateRouteMetrics'
import { InvestigateSamplesTable } from '../components/investigate/InvestigateSamplesTable'
import { api, type InvestigateResult } from '../api/client'
import { configLink, logsLink, routesTabLink, wafLink } from '../lib/deepLinks'
import { pushInvestigateHistory } from '../lib/investigateHistory'
import { copyInvestigateLink } from '../lib/investigateShare'

function queryFromSearchParams(sp: URLSearchParams) {
  const ri = sp.get('ri')
  const pi = sp.get('pi')
  return {
    host: sp.get('host') || '',
    path: sp.get('path') || '/',
    method: sp.get('method') || undefined,
    status: sp.get('status') || undefined,
    client_ip: sp.get('client_ip') || undefined,
    ri: ri != null && ri !== '' ? Number(ri) : undefined,
    pi: pi != null && pi !== '' ? Number(pi) : undefined,
  }
}

export function InvestigatePage() {
  const [searchParams, setSearchParams] = useSearchParams()
  // searchParams object identity is stable in react-router; key off serialized query.
  const searchKey = searchParams.toString()
  const urlQuery = useMemo(() => queryFromSearchParams(searchParams), [searchKey])

  const [hostInput, setHostInput] = useState(urlQuery.host)
  const [pathInput, setPathInput] = useState(urlQuery.path)
  const [data, setData] = useState<InvestigateResult | null>(null)
  const [loadError, setLoadError] = useState('')
  const [inputError, setInputError] = useState('')
  const [loading, setLoading] = useState(false)
  const [historyVersion, setHistoryVersion] = useState(0)
  const [copyHint, setCopyHint] = useState('')

  useEffect(() => {
    setHostInput(urlQuery.host)
    setPathInput(urlQuery.path || '/')
  }, [urlQuery.host, urlQuery.path])

  const runLoad = useCallback(
    async (q: ReturnType<typeof queryFromSearchParams>) => {
      const host = q.host.trim()
      if (!host) {
        setData(null)
        return
      }
      const path = q.path.trim() || '/'

      setLoading(true)
      setLoadError('')
      try {
        const result = await api.investigate({
          host,
          path,
          method: q.method,
          limit: 30,
          ri: q.ri,
          pi: q.pi,
        })
        setData(result)
        pushInvestigateHistory({ host, path, method: q.method })
        setHistoryVersion((v) => v + 1)
      } catch (e) {
        setLoadError((e as Error).message)
        setData(null)
      } finally {
        setLoading(false)
      }
    },
    [],
  )

  useEffect(() => {
    if (urlQuery.host) {
      runLoad(urlQuery)
    } else {
      setData(null)
    }
  }, [searchKey, runLoad])

  const startInvestigate = () => {
    const host = hostInput.trim()
    const pathRaw = pathInput.trim() || '/'
    const path = pathRaw.startsWith('/') ? pathRaw : `/${pathRaw}`
    if (!host) {
      setInputError('请填写 Host')
      return
    }
    setInputError('')
    const nextQuery = {
      host,
      path,
      method: urlQuery.method,
      status: urlQuery.status,
      client_ip: urlQuery.client_ip,
      ri: undefined as number | undefined,
      pi: undefined as number | undefined,
    }
    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev)
        next.set('host', host)
        next.set('path', path)
        next.delete('ri')
        next.delete('pi')
        return next
      },
      { replace: true },
    )
    // Load immediately so the button works even if URL state batching lags one frame.
    void runLoad(nextQuery)
  }

  const anchorSample = data?.samples?.[0]
  const match = data?.match ?? null
  const showRouteMetrics =
    match?.matched && match.path_index != null && match.path_index >= 0

  const copyLink = async () => {
    if (!urlQuery.host) return
    const ok = await copyInvestigateLink({
      host: urlQuery.host,
      path: urlQuery.path,
      method: urlQuery.method,
      status: urlQuery.status,
      ri: urlQuery.ri,
      pi: urlQuery.pi,
      client_ip: urlQuery.client_ip,
    })
    setCopyHint(ok ? '已复制链接' : '复制失败')
    window.setTimeout(() => setCopyHint(''), 2000)
  }

  return (
    <div className="page investigate-page">
      <PageHeader
        title="请求调查"
        desc="聚合路由裁决、访问样本、策略与健康状态，用于排查慢请求、5xx 与 WAF 拦截"
        actions={
          <>
            <button
              type="button"
              className="btn btn-sm btn-ghost"
              disabled={!urlQuery.host}
              onClick={copyLink}
            >
              <Copy size={14} aria-hidden /> 复制链接
            </button>
            {copyHint ? <span className="chart-hint">{copyHint}</span> : null}
            <button
              type="button"
              className="btn btn-sm"
              disabled={loading || !urlQuery.host}
              onClick={() => runLoad(urlQuery)}
            >
              <RefreshCw size={14} aria-hidden /> 刷新
            </button>
          </>
        }
      />

      <div className="panel investigate-context-panel">
        <div className="panel-head">
          <h2>
            <Search size={16} style={{ verticalAlign: 'text-bottom', marginRight: 6 }} />
            调查上下文
          </h2>
        </div>
        <div className="panel-body investigate-context-form">
          <label className="field-label">Host</label>
          <input
            type="text"
            className="field-input"
            value={hostInput}
            onChange={(e) => setHostInput(e.target.value)}
            placeholder="api.example.com"
          />
          <label className="field-label">Path</label>
          <input
            type="text"
            className="field-input-last"
            value={pathInput}
            onChange={(e) => setPathInput(e.target.value)}
            placeholder="/v2/users"
            onKeyDown={(e) => {
              if (e.key === 'Enter') startInvestigate()
            }}
          />
          <button
            type="button"
            className="btn btn-primary"
            style={{ marginTop: 12 }}
            disabled={loading}
            onClick={startInvestigate}
          >
            {loading ? '调查中…' : '开始调查'}
          </button>
          {inputError ? <p className="err" style={{ gridColumn: '1 / -1', margin: '8px 0 0' }}>{inputError}</p> : null}
          {urlQuery.method ? <span className="investigate-tag">method={urlQuery.method}</span> : null}
          {urlQuery.status ? <span className="investigate-tag">status={urlQuery.status}</span> : null}
        </div>
      </div>

      <div className="panel investigate-history-panel">
        <div className="panel-head">
          <h2>最近调查</h2>
        </div>
        <div className="panel-body">
          <InvestigateHistoryPanel version={historyVersion} />
        </div>
      </div>

      {!urlQuery.host && !loading ? (
        <p className="empty-hint">输入 Host 与 Path，或从总览/日志/WAF 通过「调查」入口跳转</p>
      ) : loading && !data ? (
        <p className="empty-hint">加载中…</p>
      ) : (
        <>
          {loadError ? <p className="err">{loadError}</p> : null}

          {urlQuery.host ? (
            <div className="investigate-actions toolbar">
              <Link
                to={logsLink({ host: urlQuery.host, path: urlQuery.path, log: 'access' })}
                className="btn btn-ghost btn-sm"
              >
                <ScrollText size={14} aria-hidden /> 完整日志
              </Link>
              <Link
                to={wafLink({ host: urlQuery.host, path: urlQuery.path, trial: true })}
                className="btn btn-ghost btn-sm"
              >
                <Shield size={14} aria-hidden /> WAF 试匹配
              </Link>
              {match?.matched && match.path_index != null && match.path_index >= 0 ? (
                <Link
                  to={routesTabLink('topology', {
                    highlight_ri: match.rule_index,
                    highlight_pi: match.path_index,
                  })}
                  className="btn btn-ghost btn-sm"
                >
                  <Network size={14} aria-hidden /> 拓扑
                </Link>
              ) : null}
              <Link to={configLink()} className="btn btn-ghost btn-sm">
                <Settings2 size={14} aria-hidden /> 配置
              </Link>
            </div>
          ) : null}

          {data?.stats ? (
            <div className="cards investigate-stats-cards">
              <div className="card">
                <div className="label">样本数</div>
                <div className="value">{data.stats.count}</div>
              </div>
              <div className="card">
                <div className="label">错误率</div>
                <div className="value">{data.stats.error_rate.toFixed(1)}%</div>
              </div>
              <div className="card">
                <div className="label">P95</div>
                <div className="value">{Math.round(data.stats.p95_ms)}ms</div>
              </div>
              <div className="card">
                <div className="label">缓存命中</div>
                <div className="value">{data.stats.cache_hit_rate.toFixed(1)}%</div>
              </div>
            </div>
          ) : null}

          {showRouteMetrics ? (
            <div className="panel">
              <div className="panel-head">
                <h2>路由指标</h2>
                <span className="chart-hint">15m · 规则 #{match!.rule_index}</span>
              </div>
              <div className="panel-body">
                <InvestigateRouteMetrics
                  ruleIndex={match!.rule_index}
                  pathIndex={match!.path_index}
                />
              </div>
            </div>
          ) : null}

          <div className="investigate-grid">
            <div className="panel">
              <div className="panel-head">
                <h2>路由裁决</h2>
              </div>
              <div className="panel-body">
                <InvestigateMatchPanel match={match} />
              </div>
            </div>

            <div className="panel investigate-samples-panel">
              <div className="panel-head">
                <h2>请求样本</h2>
                <span className="chart-hint">{data?.samples.length ?? 0} 条</span>
              </div>
              <div className="panel-body panel-table-wrap">
                {anchorSample ? (
                  <InvestigateLatencyBar
                    durationMs={anchorSample.duration_ms}
                    upstreamDurationMs={anchorSample.upstream_duration_ms ?? 0}
                  />
                ) : null}
                <InvestigateSamplesTable samples={data?.samples ?? []} anchorStatus={urlQuery.status} />
              </div>
            </div>
          </div>

          <div className="panel">
            <div className="panel-head">
              <h2>策略与健康</h2>
            </div>
            <div className="panel-body">
              <InvestigatePolicyPanel
                route={data?.route ?? null}
                wafRecent={data?.waf_recent ?? []}
                healthChecks={data?.health_checks ?? []}
              />
            </div>
          </div>
        </>
      )}
    </div>
  )
}
