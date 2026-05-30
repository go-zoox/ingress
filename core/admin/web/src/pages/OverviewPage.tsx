import { useEffect, useState, useCallback, useMemo, useRef, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
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
  type OverviewSnapshot,
} from '../api/client'
import { useSSE } from '../hooks/useSSE'
import { mergeOverviewPatch } from '../lib/overviewMerge'
import { useOverviewStream } from '../context/OverviewStreamContext'
import { loadPreferences, savePreferences } from '../lib/preferences'
import { normalizeMetricsWindow } from '../lib/metricsWindow'
import { computeHealthScore, healthScoreClass } from '../lib/overviewHealthScore'
import { PageLoading } from '../components/PageLoading'

const WINDOW_OPTIONS = [
  { value: '5m', label: '5 分钟' },
  { value: '15m', label: '15 分钟' },
  { value: '1h', label: '1 小时' },
  { value: '24h', label: '24 小时' },
] as const

export function OverviewPage() {
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
  const [metricsWindow, setMetricsWindow] = useState(() =>
    normalizeMetricsWindow(loadPreferences().metricsWindow),
  )
  const [sseWindow, setSseWindow] = useState(() =>
    normalizeMetricsWindow(loadPreferences().metricsWindow),
  )
  const windowFetchRef = useRef(0)
  const snapshotRef = useRef<OverviewSnapshot | null>(null)
  const prevSseConnectedRef = useRef(false)

  const { overviewPatch, connected: sseConnected, reconnecting: sseReconnecting, fallbackPolling } = useSSE(
    ['overview'],
    { window: sseWindow },
  )
  const { setStream } = useOverviewStream()

  useEffect(() => {
    setStream({
      connected: sseConnected,
      reconnecting: sseReconnecting,
      fallbackPolling,
      metricsSource: metrics?.source,
      windowStale: metrics?.window_stale,
    })
    return () => setStream(null)
  }, [
    setStream,
    sseConnected,
    sseReconnecting,
    fallbackPolling,
    metrics?.source,
    metrics?.window_stale,
  ])

  const applySnapshot = useCallback((snap: OverviewSnapshot, forWindow?: string) => {
    const expected = normalizeMetricsWindow(forWindow ?? metricsWindow)
    if (snap.window && normalizeMetricsWindow(snap.window) !== expected) {
      return
    }
    if (snap.status) {
      setStatus(snap.status)
    }
    setMetrics(snap.metrics)
    setSystemMetrics(snap.system)
    setCerts(Array.isArray(snap.certs) ? snap.certs : [])
    setHealthChecks(snap.health_checks || [])
    setHealthSummary(snap.health_summary || { total: 0, up: 0, down: 0, unknown: 0 })
    setWafBlocks(Array.isArray(snap.waf_blocks) ? snap.waf_blocks : [])
    setParseIssues(Array.isArray(snap.parse_issues) ? snap.parse_issues : [])
    setRevisions(Array.isArray(snap.revisions) ? snap.revisions : [])
    setMetricsLoading(false)
    snapshotRef.current = {
      window: snap.window || expected,
      status: snap.status,
      metrics: snap.metrics,
      system: snap.system,
      certs: Array.isArray(snap.certs) ? snap.certs : [],
      health_checks: snap.health_checks || [],
      health_summary: snap.health_summary || { total: 0, up: 0, down: 0, unknown: 0 },
      waf_blocks: Array.isArray(snap.waf_blocks) ? snap.waf_blocks : [],
      parse_issues: Array.isArray(snap.parse_issues) ? snap.parse_issues : [],
      revisions: Array.isArray(snap.revisions) ? snap.revisions : [],
    }
  }, [metricsWindow])

  const refreshSnapshot = useCallback(() => {
    api
      .overviewSnapshot(metricsWindow)
      .then(applySnapshot)
      .catch(() => {
        setMetricsLoading(false)
      })
  }, [metricsWindow, applySnapshot])

  useEffect(() => {
    const fetchId = ++windowFetchRef.current
    const window = metricsWindow
    setMetricsLoading(true)
    api
      .overviewSnapshot(window)
      .then((snap) => {
        if (fetchId !== windowFetchRef.current) return
        applySnapshot(snap, window)
      })
      .catch((e: Error) => {
        if (fetchId !== windowFetchRef.current) return
        setMetricsLoading(false)
        setErr(e.message)
      })
  }, [metricsWindow, applySnapshot])

  useEffect(() => {
    if (!overviewPatch) return
    const merged = mergeOverviewPatch(snapshotRef.current ?? undefined, overviewPatch)
    applySnapshot(merged, merged.window)
  }, [overviewPatch, applySnapshot])

  useEffect(() => {
    if (!fallbackPolling) return
    refreshSnapshot()
    const timer = window.setInterval(refreshSnapshot, 5000)
    return () => window.clearInterval(timer)
  }, [fallbackPolling, refreshSnapshot])

  useEffect(() => {
    if (sseConnected && !prevSseConnectedRef.current && snapshotRef.current) {
      refreshSnapshot()
    }
    prevSseConnectedRef.current = sseConnected
  }, [sseConnected, refreshSnapshot])

  const onWindowChange = (value: string) => {
    const normalized = normalizeMetricsWindow(value)
    setMetricsWindow(normalized)
    savePreferences({ ...loadPreferences(), metricsWindow: normalized })
  }

  useEffect(() => {
    if (sseWindow === metricsWindow) return
    const timer = window.setTimeout(() => setSseWindow(metricsWindow), 1000)
    return () => window.clearTimeout(timer)
  }, [metricsWindow, sseWindow])

  const handleParseIssueStatus = useCallback(
    async (id: number, nextStatus: 'ignored' | 'resolved') => {
      try {
        await api.updateParseIssueStatus(id, nextStatus)
        refreshSnapshot()
      } catch {
        // keep current list on failure
      }
    },
    [refreshSnapshot],
  )

  const handleWafEventStatus = useCallback(
    async (id: number, nextStatus: 'ignored' | 'resolved') => {
      try {
        await api.updateWafEventStatus(id, nextStatus)
        refreshSnapshot()
      } catch {
        // keep current list on failure
      }
    },
    [refreshSnapshot],
  )

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

  if (!status && metricsLoading) {
    return (
      <div className="page overview-page">
        <PageLoading />
      </div>
    )
  }

  const reloadReady = Boolean(status?.reload_ready)
  const wafLabel = status?.waf_enabled ? (status.waf_log_only ? 'WAF 记录' : 'WAF 拦截') : 'WAF 关'
  const fileHash = String(status?.file_hash || status?.config_hash || '')
  const runtimeHash = String(status?.runtime_hash || '')
  const latestHash = String(status?.latest_revision_hash || (revisions.length > 0 ? revisions[0].hash : ''))

  return (
    <div className="page overview-page">
      {!reloadReady ? (
        <p className="err" style={{ marginBottom: 16 }}>
          无法热加载：未找到可发送 SIGHUP 的 ingress 进程。请执行{' '}
          <code>ingress run -c {String(status?.config_path)}</code> 且 pid 文件与 admin 一致。
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
      </div>

      <OverviewCharts
        metrics={metrics}
        loading={metricsLoading && metrics === null}
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
            loading={metricsLoading && systemMetrics === null}
          />
          <div className="overview-system-divider" />
          <div className="overview-system-grid">
            <SystemBadge label="热加载" value={reloadReady ? '就绪' : '不可用'} tone={reloadReady ? 'ok' : 'warn'} />
            <SystemBadge
              label="版本"
              value={String(status?.version || '—')}
              sub={
                <>
                  hash {fileHash.slice(0, 8)}…
                  <ConfigGovernanceBadges
                    fileHash={fileHash}
                    runtimeHash={runtimeHash}
                    latestRevisionHash={latestHash}
                    runtimeDrift={status?.runtime_drift}
                    revisionDrift={status?.revision_drift}
                  />
                </>
              }
            />
            <SystemBadge
              label="监听"
              value={String(status?.listen_http ?? '—')}
              sub={`HTTPS ${String(status?.listen_https || '—')}`}
            />
            <SystemBadge label="路由" value={String(status?.rules_count ?? '—')} sub="条规则" />
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
            <SystemBadge label="最近 reload" value={String(status?.last_reload || '—')} />
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

function formatTime(iso: string) {
  try {
    return new Date(iso).toLocaleString('zh-CN', { hour12: false })
  } catch {
    return iso
  }
}
