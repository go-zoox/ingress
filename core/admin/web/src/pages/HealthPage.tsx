import { useEffect, useState } from 'react'
import { RefreshCw } from 'lucide-react'
import { PageHeader } from '../components/PageHeader'
import { EmptyStateGuide } from '../components/EmptyStateGuide'
import { api, type HealthCheckResult, type HealthSummary } from '../api/client'

export function HealthPage() {
  const [checks, setChecks] = useState<HealthCheckResult[]>([])
  const [summary, setSummary] = useState<HealthSummary>({ total: 0, up: 0, down: 0, unknown: 0 })
  const [loading, setLoading] = useState(true)
  const [err, setErr] = useState('')

  const loadChecks = () => {
    setLoading(true)
    api
      .healthCheck()
      .then((data) => {
        setChecks(data.checks || [])
        setSummary(data.summary || { total: 0, up: 0, down: 0, unknown: 0 })
        setLoading(false)
      })
      .catch((e: Error) => {
        setErr(e.message)
        setLoading(false)
      })
  }

  useEffect(() => {
    loadChecks()
    // Auto-refresh every 30s
    const timer = window.setInterval(loadChecks, 30000)
    return () => window.clearInterval(timer)
  }, [])

  return (
    <div className="page">
      <PageHeader
        title="健康检查"
        desc="探测路由 backend.service.healthcheck 配置的后端可用性"
        actions={
          <button type="button" className="btn btn-sm" onClick={loadChecks}>
            <RefreshCw size={14} aria-hidden /> 刷新
          </button>
        }
      />
      {err && <p className="err">{err}</p>}

      {/* Summary Cards */}
      <div className="cards">
        <div className="card">
          <div className="label">总数</div>
          <div className="value">{summary.total}</div>
        </div>
        <div className="card ok">
          <div className="label">UP</div>
          <div className="value">{summary.up}</div>
        </div>
        <div className="card danger">
          <div className="label">DOWN</div>
          <div className="value">{summary.down}</div>
        </div>
        <div className="card">
          <div className="label">未知</div>
          <div className="value">{summary.unknown}</div>
        </div>
      </div>

      {/* Results Table */}
      <div className="panel">
        <div className="panel-head">
          <h2>探测结果</h2>
        </div>
        <div className="panel-body panel-table-wrap">
          {loading ? (
            <p className="empty-hint">加载中…</p>
          ) : checks.length === 0 ? (
            <EmptyStateGuide
              title="暂无健康检查目标"
              configModule="healthcheck"
              linkLabel="打开配置中心"
            >
              在路由的 service 上启用 <code>healthcheck.enable</code> 后，ingress 会按间隔探测后端并在此展示
              UP/DOWN。总览与拓扑图也会引用相同状态。
            </EmptyStateGuide>
          ) : (
            <table className="data">
              <thead>
                <tr>
                  <th>Host</th>
                  <th>Path</th>
                  <th>Backend</th>
                  <th>探测 URL</th>
                  <th>状态</th>
                  <th>上次探测</th>
                  <th>响应时间</th>
                  <th>错误</th>
                </tr>
              </thead>
              <tbody>
                {checks.map((c) => (
                  <tr key={c.key} className={c.status === 'down' ? 'health-row-down' : ''}>
                    <td>{c.host}</td>
                    <td><code>{c.path}</code></td>
                    <td><code>{c.backend}</code></td>
                    <td><code className="health-url">{c.url}</code></td>
                    <td>
                      <span className={`badge ${c.status === 'up' ? 'badge-exact' : c.status === 'down' ? 'badge-block' : 'badge-audit'}`}>
                        {c.status.toUpperCase()}
                      </span>
                    </td>
                    <td>{c.last_check ? new Date(c.last_check).toLocaleString() : '—'}</td>
                    <td>{c.response_ms > 0 ? `${c.response_ms}ms` : '—'}</td>
                    <td className="health-error-cell">{c.error || '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>
    </div>
  )
}
