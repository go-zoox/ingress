import { useEffect, useState, type ReactNode } from 'react'
import { Link, useParams, useNavigate } from 'react-router-dom'
import { Network, ScrollText, Search, Settings2, Shield } from 'lucide-react'
import { investigateLink, logsLink, routesTabLink, wafLink } from '../lib/deepLinks'
import { PageHeader } from '../components/PageHeader'
import { WafRuleTooltip } from '../components/WafRuleTooltip'
import { useWafRuleLookup } from '../hooks/useWafRuleLookup'
import { api, type MetricsDelta, type RouteDetail, type RouteMetrics } from '../api/client'
import { RouteDetailCharts } from '../components/routes/RouteDetailCharts'
import { OverviewDelta } from '../components/OverviewDelta'

const METRIC_WINDOWS = [
  { value: '15m', label: '15 分钟' },
  { value: '1h', label: '1 小时' },
  { value: '24h', label: '24 小时' },
] as const

type TabKey = 'logs' | 'waf' | 'cache'

export function RouteDetailPage() {
  const { ruleIndex, pathIndex } = useParams<{ ruleIndex: string; pathIndex: string }>()
  const navigate = useNavigate()
  const [detail, setDetail] = useState<RouteDetail | null>(null)
  const [metrics, setMetrics] = useState<RouteMetrics | null>(null)
  const [activeTab, setActiveTab] = useState<TabKey>('logs')
  const [metricWindow, setMetricWindow] = useState('15m')
  const [err, setErr] = useState('')
  const [loading, setLoading] = useState(true)

  const ri = Number(ruleIndex)
  const pi = Number(pathIndex)

  useEffect(() => {
    if (isNaN(ri) || isNaN(pi)) {
      setErr('无效的路由索引')
      setLoading(false)
      return
    }

    setLoading(true)
    api.routeDetail(ri, pi)
      .catch((e: Error) => {
        setErr(e.message)
        return null
      })
      .then((d) => {
        setDetail(d)
        setLoading(false)
      })
  }, [ri, pi])

  useEffect(() => {
    if (isNaN(ri) || isNaN(pi)) return
    api.routeMetrics(ri, pi, metricWindow).then(setMetrics).catch(() => setMetrics(null))
  }, [ri, pi, metricWindow])

  // Refresh per-route metrics (do not use global SSE overview metrics).
  useEffect(() => {
    if (isNaN(ri) || isNaN(pi)) return
    const tick = () => {
      api.routeMetrics(ri, pi, metricWindow).then(setMetrics).catch(() => {})
    }
    const id = window.setInterval(tick, 5000)
    return () => window.clearInterval(id)
  }, [ri, pi, metricWindow])

  if (loading) {
    return (
      <div className="page">
        <PageHeader title="路由详情" desc="加载中…" />
        <p style={{ color: 'var(--text-muted)' }}>加载中…</p>
      </div>
    )
  }

  if (err || !detail) {
    return (
      <div className="page">
        <PageHeader title="路由详情" desc="路由详情" />
        {err && <p className="err">{err}</p>}
        <button type="button" className="btn" onClick={() => navigate('/routes')}>
          返回路由列表
        </button>
      </div>
    )
  }

  const tabs: { key: TabKey; label: string }[] = [
    { key: 'logs', label: '访问日志' },
    { key: 'waf', label: 'WAF 事件' },
    { key: 'cache', label: '缓存统计' },
  ]

  return (
    <div className="page">
      <PageHeader
        title={`路由详情 — ${detail.host}${detail.path}`}
        desc={`规则 #${detail.rule_index} · 路径 #${detail.path_index}`}
        actions={
          <>
            <Link
              to={investigateLink({
                host: detail.host,
                path: detail.path,
                ri: detail.rule_index,
                pi: detail.path_index,
              })}
              className="btn btn-primary btn-sm"
            >
              <Search size={14} aria-hidden /> 调查此路由
            </Link>
            <Link
              to={logsLink({ host: detail.host, log: 'access' })}
              className="btn btn-ghost btn-sm"
            >
              <ScrollText size={14} aria-hidden /> 日志
            </Link>
            <Link
              to={wafLink({ host: detail.host, path: detail.path })}
              className="btn btn-ghost btn-sm"
            >
              <Shield size={14} aria-hidden /> WAF
            </Link>
            <Link
              to={routesTabLink('topology', {
                highlight_ri: detail.rule_index,
                highlight_pi: detail.path_index,
              })}
              className="btn btn-ghost btn-sm"
            >
              <Network size={14} aria-hidden /> 拓扑
            </Link>
            <Link to="/config" className="btn btn-ghost btn-sm">
              <Settings2 size={14} aria-hidden /> 配置
            </Link>
          </>
        }
      />

      <div className="route-detail-grid">
        {/* Left: Configuration Overview */}
        <div className="route-detail-left">
          <div className="panel">
            <div className="panel-head">
              <h2>配置概览</h2>
            </div>
            <div className="panel-body">
              <dl className="route-detail-dl">
                <dt>Host</dt>
                <dd><code>{detail.host}</code></dd>

                <dt>Path</dt>
                <dd><code>{detail.path}</code></dd>

                <dt>Backend</dt>
                <dd>
                  <span className="badge badge-exact">{detail.backend.type}</span>{' '}
                  <code>{detail.backend.target}</code>
                </dd>

                {detail.auth && (
                  <>
                    <dt>认证</dt>
                    <dd>
                      <span className={`badge ${detail.auth.enabled ? 'badge-block' : 'badge-audit'}`}>
                        {detail.auth.summary || detail.auth.type}
                      </span>
                    </dd>
                  </>
                )}

                {detail.cache && (
                  <>
                    <dt>缓存</dt>
                    <dd>
                      <span className="badge badge-exact">已启用</span>{' '}
                      TTL {detail.cache.ttl}s · {detail.cache.max_body_kb}KB 上限
                    </dd>
                  </>
                )}

                {detail.health_check && (
                  <>
                    <dt>健康检查</dt>
                    <dd>
                      <span className="badge badge-exact">已启用</span>{' '}
                      {detail.health_check.method} {detail.health_check.path}
                      {'health_status' in detail && (
                        <span className={`badge ${detail.health_status === 'up' ? 'badge-exact' : 'badge-block'}`} style={{ marginLeft: 8 }}>
                          {String(detail.health_status === 'up' ? '✓ UP' : '✗ DOWN')}
                        </span>
                      )}
                    </dd>
                  </>
                )}

                {detail.waf && (
                  <>
                    <dt>WAF</dt>
                    <dd>
                      {detail.waf.patched ? (
                        <span className="badge badge-wildcard">已覆盖</span>
                      ) : (
                        <span className="badge badge-audit">继承全局</span>
                      )}
                    </dd>
                  </>
                )}
              </dl>
            </div>
          </div>
        </div>

        {/* Right: Real-time Metrics */}
        <div className="route-detail-right">
          <div className="panel">
            <div className="panel-head">
              <h2>实时指标</h2>
              <select
                value={metricWindow}
                onChange={(e) => setMetricWindow(e.target.value)}
                aria-label="指标时间范围"
              >
                {METRIC_WINDOWS.map((w) => (
                  <option key={w.value} value={w.value}>
                    {w.label}
                  </option>
                ))}
              </select>
            </div>
            <div className="panel-body">
              {metrics ? (
                <RouteMetricsKpis metrics={metrics} />
              ) : (
                <p className="empty-hint">暂无指标数据</p>
              )}
            </div>
          </div>

          {/* Health check status */}
          {detail.health_check && (
            <div className="panel" style={{ marginTop: 16 }}>
              <div className="panel-head">
                <h2>健康检查状态</h2>
              </div>
              <div className="panel-body">
                <div className="health-status-row">
                  <span className={`health-dot ${'health_status' in detail && detail.health_status === 'down' ? 'down' : 'up'}`}></span>
                  <span>{'health_status' in detail && detail.health_status === 'down' ? 'DOWN' : 'UP'}</span>
                  <span className="health-check-url">
                    {detail.health_check.method} {detail.health_check.path}
                  </span>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {metrics ? <RouteDetailCharts detail={detail} metrics={metrics} /> : null}

      {/* Bottom: Tab content */}
      <div className="panel" style={{ marginTop: 20 }}>
        <div className="panel-head">
          <div className="route-detail-tabs">
            {tabs.map((t) => (
              <button
                key={t.key}
                type="button"
                className={`config-view-tab ${activeTab === t.key ? 'active' : ''}`}
                onClick={() => setActiveTab(t.key)}
              >
                {t.label}
              </button>
            ))}
          </div>
        </div>
        <div className="panel-body">
          {activeTab === 'logs' && (
            <RouteLogsTab ruleIndex={detail.rule_index} pathIndex={detail.path_index} />
          )}
          {activeTab === 'waf' && (
            <RouteWAFTab ruleIndex={detail.rule_index} pathIndex={detail.path_index} />
          )}
          {activeTab === 'cache' && (
            <RouteCacheTab detail={detail} metrics={metrics} />
          )}
        </div>
      </div>
    </div>
  )
}

/** Sub-component: Access logs filtered by route indices */
function RouteLogsTab({ ruleIndex, pathIndex }: { ruleIndex: number; pathIndex: number }) {
  const [lines, setLines] = useState<string[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    setLoading(true)
    api.logs({ ri: ruleIndex, pi: pathIndex, limit: 500 }).then((r) => {
      const out = r.lines || []
      setLines(out.length > 100 ? out.slice(-100) : out)
      setLoading(false)
    }).catch(() => {
      setLines([])
      setLoading(false)
    })
  }, [ruleIndex, pathIndex])

  if (loading) return <p className="empty-hint">加载中…</p>
  if (lines.length === 0) return <p className="empty-hint">暂无访问日志</p>

  return (
    <div className="log-lines log-lines-live">
      {lines.map((line, i) => (
        <div key={i} className="log-line">{line}</div>
      ))}
    </div>
  )
}

/** Sub-component: WAF events filtered by route indices */
function RouteWAFTab({ ruleIndex, pathIndex }: { ruleIndex: number; pathIndex: number }) {
  const { lookup: ruleLookup } = useWafRuleLookup()
  const [events, setEvents] = useState<{ id: number; action: string; rule: string; client_ip: string; created_at: string }[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    setLoading(true)
    api.wafEvents({ ri: ruleIndex, pi: pathIndex, limit: 50 }).then((data) => {
      setEvents(Array.isArray(data) ? data.slice(0, 50) : [])
      setLoading(false)
    }).catch(() => {
      setEvents([])
      setLoading(false)
    })
  }, [ruleIndex, pathIndex])

  if (loading) return <p className="empty-hint">加载中…</p>
  if (events.length === 0) return <p className="empty-hint">暂无 WAF 事件</p>

  return (
    <table className="data">
      <thead>
        <tr>
          <th>时间</th>
          <th>动作</th>
          <th>规则</th>
          <th>客户端 IP</th>
        </tr>
      </thead>
      <tbody>
        {events.map((e) => (
          <tr key={e.id}>
            <td>{new Date(e.created_at).toLocaleString()}</td>
            <td><span className={`badge badge-${e.action}`}>{e.action}</span></td>
            <td>
              <WafRuleTooltip rule={e.rule} lookup={ruleLookup} />
            </td>
            <td>{e.client_ip}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

const emptyDelta: MetricsDelta = {
  total_pct: 0,
  rpm_pct: 0,
  error_rate_delta: 0,
  cache_hit_delta: 0,
  waf_blocks_delta: 0,
  p95_delta_ms: 0,
  has_previous: false,
}

function RouteMetricsKpis({ metrics }: { metrics: RouteMetrics }) {
  const delta: MetricsDelta = metrics.delta ?? emptyDelta
  const sparkCounts = (metrics.timeline ?? []).map((b) => b.count)
  const sparkErrors = (metrics.timeline ?? []).map((b) => b.error_rate ?? 0)

  return (
    <div className="route-metrics-cards">
      <RouteMetricCard
        label="次/分"
        value={metrics.rpm.toFixed(1)}
        spark={sparkCounts}
        sparkTone="var(--accent)"
        delta={<OverviewDelta delta={delta} kind="pct" value={delta.rpm_pct ?? delta.total_pct} />}
      />
      <RouteMetricCard
        label="延迟 P95"
        value={`${metrics.p95_ms.toFixed(0)}ms`}
        sub={`P50 ${metrics.p50_ms.toFixed(0)}ms`}
        delta={<OverviewDelta delta={delta} kind="ms" value={delta.p95_delta_ms} badIfUp />}
      />
      <RouteMetricCard
        label="错误率"
        value={`${metrics.error_rate.toFixed(1)}%`}
        spark={sparkErrors}
        sparkTone="var(--danger)"
        valueTone={metrics.error_rate > 5 ? 'danger' : undefined}
        delta={<OverviewDelta delta={delta} kind="pp" value={delta.error_rate_delta} badIfUp />}
      />
      <RouteMetricCard
        label="缓存命中"
        value={`${metrics.cache_hit_rate.toFixed(1)}%`}
        delta={<OverviewDelta delta={delta} kind="pp" value={delta.cache_hit_delta} badIfUp={false} />}
      />
      <RouteMetricCard
        label="请求总数"
        value={String(metrics.total)}
        spark={sparkCounts}
        sparkTone="var(--ok)"
        delta={<OverviewDelta delta={delta} kind="pct" value={delta.total_pct} />}
      />
      {(metrics.waf_blocks ?? 0) > 0 ? (
        <RouteMetricCard
          label="WAF 拦截"
          value={String(metrics.waf_blocks)}
          delta={<OverviewDelta delta={delta} kind="count" value={delta.waf_blocks_delta} badIfUp />}
        />
      ) : null}
    </div>
  )
}

function RouteMetricCard({
  label,
  value,
  sub,
  spark,
  sparkTone,
  valueTone,
  delta,
}: {
  label: string
  value: string
  sub?: string
  spark?: number[]
  sparkTone?: string
  valueTone?: 'danger'
  delta?: ReactNode
}) {
  return (
    <div className="route-metric-card route-metric-card-rich">
      <div className="label">{label}</div>
      <div className={`value${valueTone === 'danger' ? ' text-danger' : ''}`}>{value}</div>
      {sub ? <div className="route-metric-sub">{sub}</div> : null}
      {spark && spark.length > 1 ? (
        <div className="kpi-sparkline" aria-hidden>
          {spark.map((v, i) => {
            const max = Math.max(1, ...spark)
            return (
              <span
                key={i}
                style={{
                  height: `${Math.max(4, (v / max) * 100)}%`,
                  background: sparkTone ?? 'var(--accent)',
                }}
              />
            )
          })}
        </div>
      ) : null}
      {delta ? <div className="route-metric-delta">{delta}</div> : null}
    </div>
  )
}

/** Sub-component: Cache statistics */
function RouteCacheTab({
  detail,
  metrics,
}: {
  detail: RouteDetail
  metrics: RouteMetrics | null
}) {
  const [overview, setOverview] = useState<{
    global: { enabled: boolean; engine: string; ttl: number }
    stats: { hit_rate: number; total_requests: number; cache_hits: number }
  } | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api.cacheOverview().then((data) => {
      setOverview(data as never)
      setLoading(false)
    }).catch(() => {
      setOverview(null)
      setLoading(false)
    })
  }, [])

  const routeCache = metrics?.route_cache

  if (!detail.cache?.enabled && !routeCache?.enabled) {
    return <p className="empty-hint">此路由未启用 HTTP 响应缓存（backend.cache）</p>
  }

  if (loading && !routeCache) return <p className="empty-hint">加载中…</p>

  return (
    <>
      {routeCache ? (
        <dl className="route-detail-dl" style={{ marginBottom: 16 }}>
          <dt>本路由命中率</dt>
          <dd>
            {routeCache.hit_rate.toFixed(1)}%（{routeCache.hits}/{routeCache.total} · 窗口 {metrics?.window}）
          </dd>
          <dt>路由 TTL</dt>
          <dd>{routeCache.ttl}s</dd>
          <dt>Body 上限</dt>
          <dd>{routeCache.max_body_kb} KB</dd>
        </dl>
      ) : null}
      {overview ? (
        <dl className="route-detail-dl">
          <dt>缓存引擎</dt>
          <dd>
            {overview.global.engine} {overview.global.enabled ? '(已启用)' : '(未启用)'}
          </dd>
          <dt>全局 TTL</dt>
          <dd>{overview.global.ttl}s</dd>
          <dt>全站命中率</dt>
          <dd>{(overview.stats.hit_rate * 100).toFixed(1)}%</dd>
        </dl>
      ) : (
        <p className="empty-hint">暂无全局缓存概览</p>
      )}
    </>
  )
}
