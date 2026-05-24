import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { PageHeader } from '../components/PageHeader'
import { api, type RouteDetail, type RouteMetrics } from '../api/client'
import { useSSE } from '../hooks/useSSE'

type TabKey = 'logs' | 'waf' | 'cache'

export function RouteDetailPage() {
  const { ruleIndex, pathIndex } = useParams<{ ruleIndex: string; pathIndex: string }>()
  const navigate = useNavigate()
  const [detail, setDetail] = useState<RouteDetail | null>(null)
  const [metrics, setMetrics] = useState<RouteMetrics | null>(null)
  const [activeTab, setActiveTab] = useState<TabKey>('logs')
  const [err, setErr] = useState('')
  const [loading, setLoading] = useState(true)

  // SSE for real-time metrics
  const { data: sseData } = useSSE(['metrics'])

  useEffect(() => {
    const ri = Number(ruleIndex)
    const pi = Number(pathIndex)
    if (isNaN(ri) || isNaN(pi)) {
      setErr('无效的路由索引')
      setLoading(false)
      return
    }

    setLoading(true)
    Promise.all([
      api.routeDetail(ri, pi).catch((e: Error) => {
        setErr(e.message)
        return null
      }),
      api.routeMetrics(ri, pi).catch(() => null),
    ]).then(([d, m]) => {
      setDetail(d)
      setMetrics(m)
      setLoading(false)
    })
  }, [ruleIndex, pathIndex])

  // Update metrics from SSE
  useEffect(() => {
    if (sseData.metrics) {
      setMetrics(sseData.metrics as RouteMetrics)
    }
  }, [sseData.metrics])

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
            </div>
            <div className="panel-body">
              {metrics ? (
                <div className="route-metrics-cards">
                  <div className="route-metric-card">
                    <div className="label">QPS</div>
                    <div className="value">{metrics.rpm.toFixed(1)}</div>
                  </div>
                  <div className="route-metric-card">
                    <div className="label">延迟 P50</div>
                    <div className="value">{metrics.p50_ms.toFixed(1)}ms</div>
                  </div>
                  <div className="route-metric-card">
                    <div className="label">延迟 P95</div>
                    <div className="value">{metrics.p95_ms.toFixed(1)}ms</div>
                  </div>
                  <div className="route-metric-card">
                    <div className="label">错误率</div>
                    <div className="value" style={{ color: metrics.error_rate > 5 ? 'var(--danger)' : 'var(--text)' }}>
                      {metrics.error_rate.toFixed(1)}%
                    </div>
                  </div>
                  <div className="route-metric-card">
                    <div className="label">缓存命中率</div>
                    <div className="value">{metrics.cache_hit_rate.toFixed(1)}%</div>
                  </div>
                  <div className="route-metric-card">
                    <div className="label">请求总数</div>
                    <div className="value">{metrics.total}</div>
                  </div>
                </div>
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
            <RouteLogsTab host={detail.host} path={detail.path} />
          )}
          {activeTab === 'waf' && (
            <RouteWAFTab host={detail.host} path={detail.path} />
          )}
          {activeTab === 'cache' && (
            <RouteCacheTab />
          )}
        </div>
      </div>
    </div>
  )
}

/** Sub-component: Access logs filtered by host/path */
function RouteLogsTab({ host, path }: { host: string; path: string }) {
  const [lines, setLines] = useState<string[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    setLoading(true)
    api.logs({ host, q: path, limit: 100 }).then((r) => {
      setLines(r.lines || [])
      setLoading(false)
    }).catch(() => {
      setLines([])
      setLoading(false)
    })
  }, [host, path])

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

/** Sub-component: WAF events filtered by host/path */
function RouteWAFTab({ host, path }: { host: string; path: string }) {
  const [events, setEvents] = useState<{ id: number; action: string; rule: string; client_ip: string; created_at: string }[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    setLoading(true)
    api.wafEvents({ host, path, limit: 50 }).then((data) => {
      setEvents(Array.isArray(data) ? data : [])
      setLoading(false)
    }).catch(() => {
      setEvents([])
      setLoading(false)
    })
  }, [host, path])

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
            <td>{e.rule}</td>
            <td>{e.client_ip}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

/** Sub-component: Cache statistics */
function RouteCacheTab() {
  const [overview, setOverview] = useState<{ global: { enabled: boolean; engine: string; ttl: number }; stats: { hit_rate: number; total_requests: number; cache_hits: number } } | null>(null)
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

  if (loading) return <p className="empty-hint">加载中…</p>
  if (!overview) return <p className="empty-hint">暂无缓存数据</p>

  return (
    <dl className="route-detail-dl">
      <dt>缓存引擎</dt>
      <dd>{overview.global.engine} {overview.global.enabled ? '(已启用)' : '(未启用)'}</dd>
      <dt>全局 TTL</dt>
      <dd>{overview.global.ttl}s</dd>
      <dt>命中率</dt>
      <dd>{(overview.stats.hit_rate * 100).toFixed(1)}%</dd>
      <dt>总请求</dt>
      <dd>{overview.stats.total_requests}</dd>
      <dt>缓存命中</dt>
      <dd>{overview.stats.cache_hits}</dd>
    </dl>
  )
}
