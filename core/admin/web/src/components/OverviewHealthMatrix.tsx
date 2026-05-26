import { memo } from 'react'
import { Link } from 'react-router-dom'
import { Activity, AlertCircle, CheckCircle2, HelpCircle } from 'lucide-react'
import type { HealthCheckResult, HealthSummary } from '../api/client'

type Props = {
  checks: HealthCheckResult[]
  summary: HealthSummary
}

export const OverviewHealthMatrix = memo(function OverviewHealthMatrix({ checks, summary }: Props) {
  if (summary.total === 0) {
    return (
      <p className="empty-hint">
        <HelpCircle size={14} style={{ verticalAlign: 'middle', marginRight: 6 }} />
        未配置健康检查的后端
      </p>
    )
  }

  return (
    <>
      <div className="health-matrix-summary">
        <span className="health-pill ok">
          <CheckCircle2 size={14} /> UP {summary.up}
        </span>
        <span className={`health-pill ${summary.down > 0 ? 'danger' : ''}`}>
          <AlertCircle size={14} /> DOWN {summary.down}
        </span>
        {summary.unknown > 0 ? (
          <span className="health-pill">
            <Activity size={14} /> 未知 {summary.unknown}
          </span>
        ) : null}
      </div>
      <div className="health-matrix-grid">
        {checks.slice(0, 12).map((h) => (
          <div key={h.key} className={`health-matrix-cell health-${h.status}`} title={h.url}>
            <StatusIcon status={h.status} />
            <div className="health-matrix-text">
              <span className="health-matrix-host">{h.host}</span>
              <span className="health-matrix-path">{h.path}</span>
            </div>
          </div>
        ))}
      </div>
      {checks.length > 12 ? (
        <Link to="/healths" className="btn btn-ghost btn-sm" style={{ marginTop: 8 }}>
          查看全部 {checks.length} 项
        </Link>
      ) : (
        <Link to="/healths" className="btn btn-ghost btn-sm" style={{ marginTop: 8 }}>
          健康检查详情
        </Link>
      )}
    </>
  )
})

function StatusIcon({ status }: { status: string }) {
  if (status === 'up') return <CheckCircle2 size={16} className="icon-ok" />
  if (status === 'down') return <AlertCircle size={16} className="icon-danger" />
  return <HelpCircle size={16} className="icon-muted" />
}
