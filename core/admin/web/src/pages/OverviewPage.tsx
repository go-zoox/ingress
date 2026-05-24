import { useEffect, useState, useRef, useCallback } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { OverviewCharts } from '../components/OverviewCharts'
import { PageHeader } from '../components/PageHeader'
import { VersionConsistencyBadge } from '../components/VersionConsistencyBadge'
import { api, type OverviewMetrics, type WAFEvent, type TLSCert, type ConfigRevisionSummary } from '../api/client'
import { useSSE } from '../hooks/useSSE'
import { loadPreferences } from '../lib/preferences'

export function OverviewPage() {
  const navigate = useNavigate()
  const [status, setStatus] = useState<Record<string, unknown> | null>(null)
  const [events, setEvents] = useState<WAFEvent[]>([])
  const [metrics, setMetrics] = useState<OverviewMetrics | null>(null)
  const [metricsLoading, setMetricsLoading] = useState(true)
  const [err, setErr] = useState('')
  const [certs, setCerts] = useState<TLSCert[]>([])
  const [revisions, setRevisions] = useState<ConfigRevisionSummary[]>([])
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  // Track whether the first fetch has completed so we only show full-page loading once.
  const mountedRef = useRef(false)

  // SSE for real-time metrics
  const { data: sseData, connected: sseConnected } = useSSE(['metrics'])

  const fetchMetrics = useCallback(() => {
    const window = loadPreferences().metricsWindow
    if (!mountedRef.current) {
      setMetricsLoading(true)
    }
    api
      .overviewMetrics(window)
      .then((data) => {
        setMetrics(data)
        setMetricsLoading(false)
        mountedRef.current = true
      })
      .catch(() => {
        if (!mountedRef.current) {
          setMetrics(null)
          setMetricsLoading(false)
          mountedRef.current = true
        }
        // On subsequent errors, keep the last known metrics — no flash.
      })
  }, [])

  useEffect(() => {
    api
      .status()
      .then(setStatus)
      .catch((e: Error) => setErr(e.message))
    api
      .wafEvents()
      .then((d) => setEvents(Array.isArray(d) ? d.slice(0, 4) : []))
      .catch(() => setEvents([]))
    api
      .tlsCerts()
      .then((d) => setCerts(Array.isArray(d) ? d : []))
      .catch(() => setCerts([]))
    api
      .configRevisions(5)
      .then((d) => setRevisions(Array.isArray(d) ? d : []))
      .catch(() => setRevisions([]))
    fetchMetrics()

    // Use SSE for metrics if connected, otherwise fall back to polling
    const refreshMs = loadPreferences().metricsRefreshMs
    if (refreshMs > 0) {
      timerRef.current = window.setInterval(fetchMetrics, refreshMs)
    }

    return () => {
      if (timerRef.current != null) {
        window.clearInterval(timerRef.current)
      }
    }
  }, [fetchMetrics])

  // Update metrics from SSE data
  useEffect(() => {
    if (sseData.metrics) {
      setMetrics(sseData.metrics as OverviewMetrics)
      setMetricsLoading(false)
    }
  }, [sseData.metrics])

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
        <PageHeader title="总览" desc="运行状态、请求指标（access log）与近期运维事件" />
        <p style={{ color: 'var(--text-muted)' }}>加载中…</p>
      </div>
    )
  }

  // Compute cert warning count from real data
  const certWarn = certs.filter((c) => c.days_remaining < 30).length
  const certCritical = certs.filter((c) => c.days_remaining < 7).length

  const wafLabel = status.waf_enabled ? (status.waf_log_only ? '审计' : '拦截') : '关'
  const wafCardClass = status.waf_log_only ? 'warn' : ''
  const reloadReady = Boolean(status.reload_ready)

  // Version consistency
  const runningHash = String(status.config_hash || '')
  const latestHash = revisions.length > 0 ? revisions[0].hash : ''

  // SSE status indicator
  const sseStatusClass = sseConnected ? 'ok' : ''

  return (
    <div className="page">
      <PageHeader title="总览" desc="运行状态、请求指标（access log）与近期运维事件" />
      {!reloadReady ? (
        <p className="err" style={{ marginBottom: 16 }}>
          无法热加载：未找到可发送 SIGHUP 的 ingress 进程（pid 文件{' '}
          <code>{String(status.pid_file || '/tmp/gozoox.ingress.pid')}</code>）。请另开终端执行{' '}
          <code>ingress run -c {String(status.config_path)}</code>，且 <code>--pid-file</code>{' '}
          与 admin 配置一致后再发布。
        </p>
      ) : null}
      <div className="cards">
        <div className={`card ${reloadReady ? 'ok' : 'warn'}`}>
          <div className="label">热加载</div>
          <div className="value">{reloadReady ? '就绪' : '不可用'}</div>
          <div className="sub">{reloadReady ? '可 SIGHUP reload' : '需 ingress run'}</div>
        </div>
        <div className="card">
          <div className="label">版本</div>
          <div className="value">{String(status.version || 'ingress')}</div>
          <div className="sub">
            配置 hash {String(status.config_hash || '—')}
            <VersionConsistencyBadge runningHash={runningHash} latestHash={latestHash} />
          </div>
        </div>
        <div className="card">
          <div className="label">监听</div>
          <div className="value">{String(status.listen_http)}</div>
          <div className="sub">HTTPS {String(status.listen_https || '—')}</div>
        </div>
        <div className="card">
          <div className="label">路由规则</div>
          <div className="value">{String(status.rules_count)}</div>
          <div className="sub">上次 reload {String(status.last_reload || '—')}</div>
        </div>
        <div className={`card ${wafCardClass}`}>
          <div className="label">WAF</div>
          <div className="value">{wafLabel}</div>
          <div className="sub">log_only={String(status.waf_log_only)}</div>
        </div>
        <div className={`card ${certCritical > 0 ? 'danger' : certWarn > 0 ? 'warn' : 'ok'}`}>
          <div className="label">证书</div>
          <div className="value">
            {certCritical > 0
              ? `${certCritical} 即将过期`
              : certWarn > 0
                ? `${certWarn} 需关注`
                : '正常'}
          </div>
          <div className="sub">TLS 证书有效期</div>
        </div>
        <div className={`card ${sseStatusClass}`}>
          <div className="label">实时推送</div>
          <div className="value">{sseConnected ? '已连接' : '未连接'}</div>
          <div className="sub">SSE 事件流</div>
        </div>
      </div>
      <OverviewCharts metrics={metrics} loading={metricsLoading} />
      <div className="panel">
        <div className="panel-head">
          <h2>最近事件</h2>
          <Link to="/logs" className="btn btn-ghost">
            查看全部日志
          </Link>
        </div>
        <div className="panel-body panel-table-wrap">
          <table className="data">
            <thead>
              <tr>
                <th>时间</th>
                <th>动作</th>
                <th>规则</th>
                <th>Host</th>
                <th>Path</th>
              </tr>
            </thead>
            <tbody>
              {events.length === 0 ? (
                <tr>
                  <td colSpan={5} className="empty-hint">
                    暂无 WAF 事件
                  </td>
                </tr>
              ) : (
                events.map((e) => (
                  <tr key={e.id}>
                    <td>{formatTime(e.created_at)}</td>
                    <td>
                      <span className={`badge badge-${e.action}`}>{e.action}</span>
                    </td>
                    <td>{e.rule}</td>
                    <td>{e.host}</td>
                    <td>
                      <code>{e.path}</code>
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
          <h2>快捷操作</h2>
        </div>
        <div className="panel-body toolbar">
          <button type="button" className="btn" onClick={() => navigate('/routes')}>
            路由试匹配
          </button>
          <button type="button" className="btn" onClick={() => navigate('/config')}>
            编辑配置
          </button>
          <button type="button" className="btn btn-primary" onClick={() => navigate('/config')}>
            校验并发布
          </button>
        </div>
      </div>
    </div>
  )
}

function formatTime(iso: string) {
  try {
    const d = new Date(iso)
    return d.toLocaleTimeString('zh-CN', { hour12: false })
  } catch {
    return iso
  }
}
