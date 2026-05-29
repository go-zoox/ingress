import { useEffect, useState, useRef, useCallback, useMemo, type ReactNode } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { OverviewCharts } from '../components/OverviewCharts'
import { OverviewAttentionPanel } from '../components/OverviewAttentionPanel'
import { ConfigGovernanceBadges } from '../components/ConfigGovernanceBadges'
import { SystemResourceStrip } from '../components/SystemResourceStrip'
import {
  api,
  type OverviewMetrics,
  type SystemMetrics,
  type AccessLogParseIssue,
  type WAFEvent,
  type TLSCert,
  type ConfigRevisionSummary,
  type HealthCheckResult,
  type HealthSummary,
  type IngressStatus,
} from '../api/client'
import { useSSE } from '../hooks/useSSE'
import { loadPreferences, savePreferences } from '../lib/preferences'
import { computeHealthScore, healthScoreClass } from '../lib/overviewHealthScore'
import { FileText, LayoutDashboard, Radio, Route, Settings2 } from 'lucide-react'

const WINDOW_OPTIONS = [
  { value: '5m', label: '5 分钟' },
  { value: '15m', label: '15 分钟' },
  { value: '1h', label: '1 小时' },
  { value: '24h', label: '24 小时' },
] as const

export function OverviewPage() {
  const navigate = useNavigate()
  const [status, setStatus] = useState<IngressStatus | null>(null)
  const [metrics, setMetrics] = useState<OverviewMetrics | null>(null)
  const [systemMetrics, setSystemMetrics] = useState<SystemMetrics | null>(null)
  const [parseIssues, setParseIssues] = useState<AccessLogParseIssue[]>([])
  const [metricsLoading, setMetricsLoading] = useState(true)
  const [err, setErr] = useState('')
  const [certs, setCerts] = useState<TLSCert[]>([])
  const [healthChecks, setHealthChecks] = useState<HealthCheckResult[]>([])
  const [healthSummary, setHealthSummary] = useState<HealthSummary>({
    total: 0,
    up: 0,
    down: 0,
    unknown: 0,
  })
  const [wafBlocks, setWafBlocks] = useState<WAFEvent[]>([])
  const [revisions, setRevisions] = useState<ConfigRevisionSummary[]>([])
  const [metricsWindow, setMetricsWindow] = useState(() => loadPreferences().metricsWindow)
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const mountedRef = useRef(false)

  const { data: sseData, connected: sseConnected } = useSSE(['metrics'])

  const loadParseIssues = useCallback(() => {
    api
      .parseIssues('open', 10)
      .then((d) => setParseIssues(Array.isArray(d) ? d : []))
      .catch(() => setParseIssues([]))
  }, [])

  const fetchMetrics = useCallback(() => {
    if (!mountedRef.current) {
      setMetricsLoading(true)
    }
    Promise.all([api.overviewMetrics(metricsWindow), api.systemMetrics(metricsWindow)])
      .then(([overview, system]) => {
        setMetrics(overview)
        setSystemMetrics(system)
        setMetricsLoading(false)
        mountedRef.current = true
        loadParseIssues()
      })
      .catch(() => {
        if (!mountedRef.current) {
          setMetrics(null)
          setSystemMetrics(null)
          setMetricsLoading(false)
          mountedRef.current = true
        }
      })
  }, [metricsWindow, loadParseIssues])

  const handleParseIssueStatus = useCallback(
    async (id: number, status: 'ignored' | 'resolved') => {
      try {
        await api.updateParseIssueStatus(id, status)
        loadParseIssues()
        fetchMetrics()
      } catch {
        // keep current list on failure
      }
    },
    [loadParseIssues, fetchMetrics],
  )

  const loadWafBlocks = useCallback(() => {
    api
      .wafEvents({ action: 'block', status: 'open', limit: 30 })
      .then((d) => {
        const list = Array.isArray(d) ? d : []
        setWafBlocks(list.filter((e) => e.action === 'block').slice(0, 8))
      })
      .catch(() => setWafBlocks([]))
  }, [])

  const handleWafEventStatus = useCallback(
    async (id: number, status: 'ignored' | 'resolved') => {
      try {
        await api.updateWafEventStatus(id, status)
        loadWafBlocks()
      } catch {
        // keep current list on failure
      }
    },
    [loadWafBlocks],
  )

  const loadAux = useCallback(() => {
    api
      .tlsCerts()
      .then((d) => setCerts(Array.isArray(d) ? d : []))
      .catch(() => setCerts([]))
    api
      .healthCheck()
      .then((d) => {
        setHealthChecks(d.checks || [])
        setHealthSummary(d.summary || { total: 0, up: 0, down: 0, unknown: 0 })
      })
      .catch(() => {
        setHealthChecks([])
        setHealthSummary({ total: 0, up: 0, down: 0, unknown: 0 })
      })
    loadWafBlocks()
    api
      .configRevisions(5)
      .then((d) => setRevisions(Array.isArray(d) ? d : []))
      .catch(() => setRevisions([]))
  }, [loadWafBlocks])

  useEffect(() => {
    api
      .status()
      .then(setStatus)
      .catch((e: Error) => setErr(e.message))
    loadAux()
    fetchMetrics()

    const refreshMs = loadPreferences().metricsRefreshMs
    if (refreshMs > 0) {
      timerRef.current = window.setInterval(() => {
        fetchMetrics()
        loadAux()
      }, refreshMs)
    }

    return () => {
      if (timerRef.current != null) {
        window.clearInterval(timerRef.current)
      }
    }
  }, [fetchMetrics, loadAux])

  useEffect(() => {
    if (sseData.metrics) {
      setMetrics(sseData.metrics as OverviewMetrics)
      setMetricsLoading(false)
    }
  }, [sseData.metrics])

  const onWindowChange = (value: string) => {
    setMetricsWindow(value)
    const prefs = loadPreferences()
    savePreferences({ ...prefs, metricsWindow: value })
    mountedRef.current = false
    setMetricsLoading(true)
  }

  const certWarn = certs.filter((c) => c.days_remaining < 30).length
  const certCritical = certs.filter((c) => c.days_remaining < 7).length

  const healthScore = useMemo(() => {
    if (!metrics || metrics.total === 0) return null
    return computeHealthScore({
      errorRate: metrics.error_rate,
      p95Ms: metrics.p95_ms,
      healthDown: healthSummary.down,
      certCritical,
      certWarn,
      wafBlocks: metrics.waf_blocks,
    })
  }, [metrics, healthSummary.down, certCritical, certWarn])

  const healthClass = healthScore != null ? healthScoreClass(healthScore) : 'ok'

  if (err) {
    return (
      <div className="page">
        <p className="err">{err}</p>
      </div>
    )
  }

  if (!status) {
    return (
      <div className="page">
        <p style={{ color: 'var(--text-muted)' }}>加载中…</p>
      </div>
    )
  }

  const reloadReady = Boolean(status.reload_ready)
  const wafLabel = status.waf_enabled ? (status.waf_log_only ? 'WAF 记录' : 'WAF 拦截') : 'WAF 关'
  const fileHash = String(status.file_hash || status.config_hash || '')
  const runtimeHash = String(status.runtime_hash || '')
  const latestHash = String(status.latest_revision_hash || (revisions.length > 0 ? revisions[0].hash : ''))

  return (
    <div className="page overview-page">
      {!reloadReady ? (
        <p className="err" style={{ marginBottom: 16 }}>
          无法热加载：未找到可发送 SIGHUP 的 ingress 进程。请执行{' '}
          <code>ingress run -c {String(status.config_path)}</code> 且 pid 文件与 admin 一致。
        </p>
      ) : null}

      <div className="overview-toolbar">
        <div className="overview-window-tabs" role="tablist" aria-label="指标时间窗口">
          {WINDOW_OPTIONS.map((opt) => (
            <button
              key={opt.value}
              type="button"
              role="tab"
              aria-selected={metricsWindow === opt.value}
              className={metricsWindow === opt.value ? 'btn btn-sm active' : 'btn btn-sm btn-ghost'}
              onClick={() => onWindowChange(opt.value)}
            >
              {opt.label}
            </button>
          ))}
        </div>
        <div className="overview-toolbar-meta">
          <span className={`overview-badge ${sseConnected ? 'ok' : ''}`}>
            <Radio size={12} aria-hidden />
            {sseConnected ? 'SSE 已连接' : '轮询刷新'}
          </span>
          <span className="overview-badge">
            <LayoutDashboard size={12} aria-hidden />
            数据源 {metricsSourceLabel(metrics?.source)}
          </span>
        </div>
        <div className="overview-toolbar-actions">
          <button type="button" className="btn btn-ghost btn-sm" onClick={() => navigate('/logs')}>
            <FileText size={14} aria-hidden /> 日志
          </button>
          <button type="button" className="btn btn-ghost btn-sm" onClick={() => navigate('/routes')}>
            <Route size={14} aria-hidden /> 路由
          </button>
          <button type="button" className="btn btn-primary btn-sm" onClick={() => navigate('/config')}>
            <Settings2 size={14} aria-hidden /> 配置
          </button>
        </div>
      </div>

      <OverviewCharts
        metrics={metrics}
        loading={metricsLoading}
        healthScore={healthScore}
        healthClass={healthClass}
        healthChecks={healthChecks}
        healthSummary={healthSummary}
        certs={certs}
      />

      <div className="panel overview-system-panel">
        <div className="panel-head">
          <h2>系统状态</h2>
        </div>
        <div className="panel-body">
          <SystemResourceStrip
            system={systemMetrics}
            metrics={metrics}
            loading={metricsLoading}
          />
          <div className="overview-system-divider" />
          <div className="overview-system-grid">
            <SystemBadge label="热加载" value={reloadReady ? '就绪' : '不可用'} tone={reloadReady ? 'ok' : 'warn'} />
            <SystemBadge
              label="版本"
              value={String(status.version || '—')}
              sub={
                <>
                  hash {fileHash.slice(0, 8)}…
                  <ConfigGovernanceBadges
                    fileHash={fileHash}
                    runtimeHash={runtimeHash}
                    latestRevisionHash={latestHash}
                    runtimeDrift={status.runtime_drift}
                    revisionDrift={status.revision_drift}
                  />
                </>
              }
            />
            <SystemBadge
              label="监听"
              value={String(status.listen_http)}
              sub={`HTTPS ${String(status.listen_https || '—')}`}
            />
            <SystemBadge label="路由" value={String(status.rules_count)} sub="条规则" />
            <SystemBadge label="WAF" value={wafLabel} />
            <SystemBadge
              label="证书"
              value={
                certCritical > 0
                  ? `${certCritical} 紧急`
                  : certWarn > 0
                    ? `${certWarn} 关注`
                    : '正常'
              }
              tone={certCritical > 0 ? 'danger' : certWarn > 0 ? 'warn' : 'ok'}
            />
            <SystemBadge
              label="健康检查"
              value={healthSummary.total > 0 ? `${healthSummary.up}/${healthSummary.total} UP` : '未配置'}
              tone={healthSummary.down > 0 ? 'danger' : healthSummary.total > 0 ? 'ok' : undefined}
            />
            <SystemBadge label="最近 reload" value={String(status.last_reload || '—')} />
          </div>
        </div>
      </div>

      <OverviewAttentionPanel
        metrics={metrics}
        certs={certs}
        healthChecks={healthChecks}
        wafBlocks={wafBlocks}
        parseIssues={parseIssues}
        onParseIssueStatus={handleParseIssueStatus}
        onWafEventStatus={handleWafEventStatus}
      />

      {revisions.length > 0 ? (
        <div className="panel">
          <div className="panel-head">
            <h2>配置发布</h2>
            <Link to="/config" className="btn btn-ghost btn-sm">
              配置中心
            </Link>
          </div>
          <div className="panel-body">
            <ul className="revision-mini-list">
              {revisions.slice(0, 3).map((r) => (
                <li key={r.id}>
                  <code>{r.hash.slice(0, 8)}</code>
                  <span>{r.note || '—'}</span>
                  <time>{formatTime(r.created_at)}</time>
                </li>
              ))}
            </ul>
          </div>
        </div>
      ) : null}
    </div>
  )
}

function SystemBadge({
  label,
  value,
  sub,
  tone,
}: {
  label: string
  value: string
  sub?: ReactNode
  tone?: 'ok' | 'warn' | 'danger'
}) {
  const cls = tone ? `overview-sys-badge overview-sys-${tone}` : 'overview-sys-badge'
  return (
    <div className={cls}>
      <span className="overview-sys-label">{label}</span>
      <span className="overview-sys-value">{value}</span>
      {sub ? <span className="overview-sys-sub">{sub}</span> : null}
    </div>
  )
}

function metricsSourceLabel(source?: string) {
  switch (source) {
    case 'access_log':
      return 'access.log'
    case 'access_log_empty':
      return '空文件'
    case 'access_log_parse_fail':
      return '解析异常'
    case 'unconfigured':
      return '未配置'
    case 'error':
      return '读取失败'
    default:
      return source || '—'
  }
}

function formatTime(iso: string) {
  try {
    return new Date(iso).toLocaleString('zh-CN', { hour12: false })
  } catch {
    return iso
  }
}
