import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { Activity, Radio, ScrollText, Settings2, Route } from 'lucide-react'
import { healthLink, logsLink, routeDetailLink } from '../lib/deepLinks'
import { PageHeader } from '../components/PageHeader'
import { api, type MetricsDelta, type ServiceDetail, type ServiceMetrics } from '../api/client'
import { ServiceDetailCharts } from '../components/services/ServiceDetailCharts'
import { OverviewDelta } from '../components/OverviewDelta'
import { metricsSourceLabel } from '../lib/metricsSource'
import { loadPreferences, savePreferences } from '../lib/preferences'
import { normalizeMetricsWindow } from '../lib/metricsWindow'

const METRICS_AUTO_REFRESH_MS = 5000

const WINDOW_OPTIONS = [
  { value: '5m', label: '5 分钟' },
  { value: '15m', label: '15 分钟' },
  { value: '1h', label: '1 小时' },
  { value: '6h', label: '6 小时' },
  { value: '24h', label: '24 小时' },
] as const

type TabKey = 'logs' | 'health'

export function ServiceDetailPage() {
  const { name: nameParam } = useParams<{ name: string }>()
  const navigate = useNavigate()
  const serviceName = nameParam ? decodeURIComponent(nameParam) : ''

  const [detail, setDetail] = useState<ServiceDetail | null>(null)
  const [metrics, setMetrics] = useState<ServiceMetrics | null>(null)
  const [metricsLoading, setMetricsLoading] = useState(true)
  const [activeTab, setActiveTab] = useState<TabKey>('logs')
  const [metricWindow, setMetricWindow] = useState(() =>
    normalizeMetricsWindow(loadPreferences().metricsWindow),
  )
  const [err, setErr] = useState('')
  const [loading, setLoading] = useState(true)
  const metricsMountedRef = useRef(false)
  const metricsTimerRef = useRef<number | null>(null)

  useEffect(() => {
    if (!serviceName) {
      setErr('无效的服务名称')
      setLoading(false)
      return
    }

    setLoading(true)
    api
      .serviceDetail(serviceName)
      .catch((e: Error) => {
        setErr(e.message)
        return null
      })
      .then((d) => {
        setDetail(d)
        setLoading(false)
      })
  }, [serviceName])

  const fetchMetrics = useCallback(() => {
    if (!serviceName) return
    if (!metricsMountedRef.current) {
      setMetricsLoading(true)
    }
    api
      .serviceMetrics(serviceName, metricWindow)
      .then((m) => {
        setMetrics(m)
        setMetricsLoading(false)
        metricsMountedRef.current = true
      })
      .catch(() => {
        if (!metricsMountedRef.current) {
          setMetrics(null)
          setMetricsLoading(false)
          metricsMountedRef.current = true
        }
      })
  }, [serviceName, metricWindow])

  useEffect(() => {
    if (!serviceName) return
    fetchMetrics()
    if (METRICS_AUTO_REFRESH_MS > 0) {
      metricsTimerRef.current = window.setInterval(fetchMetrics, METRICS_AUTO_REFRESH_MS)
    }
    return () => {
      if (metricsTimerRef.current != null) {
        window.clearInterval(metricsTimerRef.current)
      }
    }
  }, [fetchMetrics, serviceName])

  const onWindowChange = (value: string) => {
    setMetricWindow(value)
    savePreferences({ ...loadPreferences(), metricsWindow: value })
    metricsMountedRef.current = false
    setMetricsLoading(true)
  }

  if (loading) {
    return (
      <div className="page">
        <PageHeader title="服务详情" desc="加载中…" />
        <p style={{ color: 'var(--text-muted)' }}>加载中…</p>
      </div>
    )
  }

  if (err || !detail) {
    return (
      <div className="page">
        <PageHeader title="服务详情" desc="服务详情" />
        {err && <p className="err">{err}</p>}
        <button type="button" className="btn" onClick={() => navigate('/services')}>
          返回服务列表
        </button>
      </div>
    )
  }

  const healthStatus = deriveHealthStatus(metrics)
  const tabs: { key: TabKey; label: string }[] = [
    { key: 'logs', label: '访问日志' },
    { key: 'health', label: '健康检查' },
  ]

  return (
    <div className="page">
      <PageHeader
        title={`服务详情 — ${detail.name}`}
        desc={`上游 ${detail.target} · ${detail.route_ref_count} 条路由引用`}
        actions={
          <>
            <Link
              to={logsLink({ log: 'access', q: detail.target })}
              className="btn btn-primary btn-sm"
            >
              <ScrollText size={14} aria-hidden /> 访问日志
            </Link>
            {detail.health_check ? (
              <Link to={healthLink({ host: detail.name })} className="btn btn-ghost btn-sm">
                <Activity size={14} aria-hidden /> 健康检查
              </Link>
            ) : null}
            <Link to="/services" className="btn btn-ghost btn-sm">
              <Route size={14} aria-hidden /> 服务列表
            </Link>
            <Link to="/config?focus=services" className="btn btn-ghost btn-sm">
              <Settings2 size={14} aria-hidden /> 配置
            </Link>
          </>
        }
      />

      <div className="overview-toolbar">
        <div className="overview-window-tabs" role="tablist" aria-label="指标时间窗口">
          {WINDOW_OPTIONS.map((opt) => (
            <button
              key={opt.value}
              type="button"
              role="tab"
              aria-selected={metricWindow === opt.value}
              className={metricWindow === opt.value ? 'btn btn-sm active' : 'btn btn-sm btn-ghost'}
              onClick={() => onWindowChange(opt.value)}
            >
              {opt.label}
            </button>
          ))}
        </div>
        <div className="overview-toolbar-meta">
          <span className="overview-badge">
            <Radio size={12} aria-hidden />
            轮询刷新
          </span>
          <span className="overview-badge">
            数据源 {metricsSourceLabel(metrics?.source)}
          </span>
        </div>
      </div>

      <div className="route-detail-grid">
        <div className="route-detail-left">
          <div className="panel">
            <div className="panel-head">
              <h2>配置概览</h2>
            </div>
            <div className="panel-body">
              <dl className="route-detail-dl">
                <dt>名称</dt>
                <dd><code>{detail.name}</code></dd>

                <dt>目标</dt>
                <dd><code>{detail.target}</code></dd>

                <dt>协议</dt>
                <dd><span className="badge badge-exact">{detail.protocol || 'http'}</span></dd>

                <dt>端口</dt>
                <dd>{detail.port > 0 ? detail.port : '默认'}</dd>

                {detail.mode ? (
                  <>
                    <dt>Host 模式</dt>
                    <dd><span className="badge badge-audit">{detail.mode}</span></dd>
                  </>
                ) : null}

                {detail.note ? (
                  <>
                    <dt>备注</dt>
                    <dd>{detail.note}</dd>
                  </>
                ) : null}

                {detail.target_aliases.length > 1 ? (
                  <>
                    <dt>日志目标别名</dt>
                    <dd>
                      {detail.target_aliases.map((t) => (
                        <code key={t} style={{ display: 'block', marginBottom: 4 }}>
                          {t}
                        </code>
                      ))}
                    </dd>
                  </>
                ) : null}

                {detail.health_check ? (
                  <>
                    <dt>健康检查</dt>
                    <dd>
                      <span className="badge badge-exact">已启用</span>{' '}
                      {detail.health_check.method} {detail.health_check.path}
                      {healthStatus ? (
                        <span
                          className={`badge ${healthStatus === 'up' ? 'badge-exact' : 'badge-block'}`}
                          style={{ marginLeft: 8 }}
                        >
                          {healthStatus === 'up' ? '✓ UP' : healthStatus === 'down' ? '✗ DOWN' : '?'}
                        </span>
                      ) : null}
                    </dd>
                  </>
                ) : null}

                <dt>引用路由</dt>
                <dd>{detail.route_ref_count} 条</dd>
              </dl>
            </div>
          </div>

          {detail.route_refs.length > 0 ? (
            <div className="panel" style={{ marginTop: 16 }}>
              <div className="panel-head">
                <h2>引用路由</h2>
              </div>
              <div className="panel-body panel-table-wrap">
                <table className="data compact">
                  <thead>
                    <tr>
                      <th>Host</th>
                      <th>Path</th>
                      <th></th>
                    </tr>
                  </thead>
                  <tbody>
                    {detail.route_refs.map((ref) => (
                      <tr key={`${ref.rule_index}-${ref.path_index}`}>
                        <td><code>{ref.host}</code></td>
                        <td><code>{ref.path}</code></td>
                        <td>
                          <Link
                            to={routeDetailLink(ref.rule_index, ref.path_index)}
                            className="btn btn-ghost btn-sm"
                          >
                            详情
                          </Link>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          ) : (
            <div className="panel" style={{ marginTop: 16 }}>
              <div className="panel-head">
                <h2>引用路由</h2>
              </div>
              <div className="panel-body">
                <p className="empty-hint">暂无路由引用此服务名；在路由 backend 中选择本服务后会出现在此。</p>
              </div>
            </div>
          )}
        </div>

        <div className="route-detail-right">
          <div className="panel">
            <div className="panel-head">
              <h2>实时指标</h2>
            </div>
            <div className="panel-body">
              {metricsLoading && !metrics ? (
                <p className="empty-hint">加载中…</p>
              ) : metrics ? (
                <ServiceMetricsKpis metrics={metrics} />
              ) : (
                <p className="empty-hint">暂无指标数据</p>
              )}
            </div>
          </div>

          {detail.health_check && metrics?.health_summary && metrics.health_summary.total > 0 ? (
            <div className="panel" style={{ marginTop: 16 }}>
              <div className="panel-head">
                <h2>健康检查状态</h2>
              </div>
              <div className="panel-body">
                <div className="health-status-row">
                  <span className={`health-dot ${healthStatus === 'down' ? 'down' : 'up'}`} />
                  <span>
                    UP {metrics.health_summary.up} / DOWN {metrics.health_summary.down}
                    {metrics.health_summary.unknown > 0
                      ? ` / ? ${metrics.health_summary.unknown}`
                      : ''}
                  </span>
                </div>
              </div>
            </div>
          ) : null}
        </div>
      </div>

      {metrics ? <ServiceDetailCharts detail={detail} metrics={metrics} /> : null}

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
            <ServiceLogsTab
              target={detail.target}
              aliases={detail.target_aliases}
              refreshMs={METRICS_AUTO_REFRESH_MS}
            />
          )}
          {activeTab === 'health' && (
            <ServiceHealthTab
              checks={metrics?.health_checks ?? []}
              configured={Boolean(detail.health_check)}
            />
          )}
        </div>
      </div>
    </div>
  )
}

function deriveHealthStatus(metrics: ServiceMetrics | null): string {
  const summary = metrics?.health_summary
  if (!summary || summary.total === 0) return ''
  if (summary.down > 0) return 'down'
  if (summary.up > 0) return 'up'
  return 'unknown'
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

function ServiceMetricsKpis({ metrics }: { metrics: ServiceMetrics }) {
  const delta: MetricsDelta = metrics.delta ?? emptyDelta
  const sparkCounts = (metrics.timeline ?? []).map((b) => b.count)
  const sparkErrors = (metrics.timeline ?? []).map((b) => b.error_rate ?? 0)

  return (
    <div className="route-metrics-cards">
      <MetricCard
        label="次/分"
        value={metrics.rpm.toFixed(1)}
        spark={sparkCounts}
        sparkTone="var(--accent)"
        delta={<OverviewDelta delta={delta} kind="pct" value={delta.rpm_pct ?? delta.total_pct} />}
      />
      <MetricCard
        label="延迟 P95"
        value={`${metrics.p95_ms.toFixed(0)}ms`}
        sub={`P50 ${metrics.p50_ms.toFixed(0)}ms`}
        delta={<OverviewDelta delta={delta} kind="ms" value={delta.p95_delta_ms} badIfUp />}
      />
      <MetricCard
        label="错误率"
        value={`${metrics.error_rate.toFixed(1)}%`}
        spark={sparkErrors}
        sparkTone="var(--danger)"
        valueTone={metrics.error_rate > 5 ? 'danger' : undefined}
        delta={<OverviewDelta delta={delta} kind="pp" value={delta.error_rate_delta} badIfUp />}
      />
      <MetricCard
        label="缓存命中"
        value={`${metrics.cache_hit_rate.toFixed(1)}%`}
        delta={<OverviewDelta delta={delta} kind="pp" value={delta.cache_hit_delta} badIfUp={false} />}
      />
      <MetricCard
        label="请求总数"
        value={String(metrics.total)}
        spark={sparkCounts}
        sparkTone="var(--ok)"
        delta={<OverviewDelta delta={delta} kind="pct" value={delta.total_pct} />}
      />
      {metrics.compare && metrics.compare.service_share_pct > 0 ? (
        <MetricCard
          label="全站占比"
          value={`${metrics.compare.service_share_pct.toFixed(1)}%`}
          sub={`全站 ${metrics.compare.site_rpm.toFixed(0)} rpm`}
        />
      ) : null}
    </div>
  )
}

function MetricCard({
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

function ServiceLogsTab({
  target,
  aliases,
  refreshMs,
}: {
  target: string
  aliases: string[]
  refreshMs: number
}) {
  const [lines, setLines] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const mountedRef = useRef(false)
  const aliasSet = useMemo(() => new Set(aliases.length > 0 ? aliases : [target]), [aliases, target])

  const fetchLogs = useCallback(() => {
    if (!mountedRef.current) setLoading(true)
    api
      .logs({ limit: 500, q: target })
      .then((r) => {
        const raw = r.lines || []
        const filtered = raw.filter((line) => {
          const low = line.toLowerCase()
          for (const a of aliasSet) {
            if (low.includes(a.toLowerCase())) return true
          }
          return false
        })
        setLines(filtered.length > 100 ? filtered.slice(-100) : filtered)
        setLoading(false)
        mountedRef.current = true
      })
      .catch(() => {
        setLines([])
        setLoading(false)
        mountedRef.current = true
      })
  }, [target, aliasSet])

  useEffect(() => {
    fetchLogs()
    if (refreshMs <= 0) return
    const id = window.setInterval(fetchLogs, refreshMs)
    return () => window.clearInterval(id)
  }, [fetchLogs, refreshMs])

  if (loading && lines.length === 0) return <p className="empty-hint">加载中…</p>
  if (lines.length === 0) {
    return <p className="empty-hint">暂无匹配此上游目标的访问日志</p>
  }

  return (
    <div className="log-lines log-lines-live">
      {lines.map((line, i) => (
        <div key={i} className="log-line">
          {line}
        </div>
      ))}
    </div>
  )
}

function ServiceHealthTab({
  checks,
  configured,
}: {
  checks: ServiceMetrics['health_checks']
  configured: boolean
}) {
  if (!configured) {
    return <p className="empty-hint">此服务未在 catalog 中配置 healthcheck</p>
  }
  if (!checks || checks.length === 0) {
    return (
      <p className="empty-hint">
        暂无探测结果（请确认路由 backend 已启用 healthcheck 且 Admin 健康检查任务在运行）
      </p>
    )
  }

  return (
    <table className="data">
      <thead>
        <tr>
          <th>后端</th>
          <th>URL</th>
          <th>状态</th>
          <th>响应</th>
          <th>最近检查</th>
        </tr>
      </thead>
      <tbody>
        {checks.map((c) => (
          <tr key={c.key}>
            <td><code>{c.backend}</code></td>
            <td><code>{c.url}</code></td>
            <td>
              <span
                className={`badge ${c.status === 'up' ? 'badge-exact' : c.status === 'down' ? 'badge-block' : 'badge-audit'}`}
              >
                {c.status}
              </span>
            </td>
            <td>{c.response_ms}ms</td>
            <td>{c.last_check ? new Date(c.last_check).toLocaleString() : '—'}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}
