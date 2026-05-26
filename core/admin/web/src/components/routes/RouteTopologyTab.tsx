import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { TopologySVG } from '../TopologySVG'
import { api, type RouteRow, type HealthCheckResult, type TLSCert } from '../../api/client'
import {
  computeTopologyLayout,
  topologyHighlightNodeIds,
  type TopologyNode,
} from '../../lib/topologyLayout'

type Props = {
  highlightRi?: number
  highlightPi?: number
}

export function RouteTopologyTab({ highlightRi, highlightPi }: Props) {
  const navigate = useNavigate()
  const [rows, setRows] = useState<RouteRow[]>([])
  const [healthChecks, setHealthChecks] = useState<HealthCheckResult[]>([])
  const [certs, setCerts] = useState<TLSCert[]>([])
  const [err, setErr] = useState('')

  useEffect(() => {
    Promise.all([
      api.routes().catch(() => []),
      api.healthCheck().catch(() => ({ checks: [], summary: { total: 0, up: 0, down: 0, unknown: 0 } })),
      api.tlsCerts().catch(() => []),
    ])
      .then(([routes, health, tls]) => {
        setRows(Array.isArray(routes) ? routes : [])
        setHealthChecks(health?.checks || [])
        setCerts(Array.isArray(tls) ? tls : [])
      })
      .catch((e: Error) => setErr(e.message))
  }, [])

  const healthMap = useMemo(() => {
    const m = new Map<string, string>()
    for (const h of healthChecks) {
      m.set(h.backend, h.status)
    }
    return m
  }, [healthChecks])

  const certWarnMap = useMemo(() => {
    const m = new Map<string, string>()
    for (const c of certs) {
      if (c.days_remaining < 7) m.set(c.domain, 'critical')
      else if (c.days_remaining < 30) m.set(c.domain, 'warning')
    }
    return m
  }, [certs])

  const layout = useMemo(() => computeTopologyLayout(rows, healthMap, certWarnMap), [rows, healthMap, certWarnMap])

  const highlightIds = useMemo(() => {
    if (highlightRi == null || highlightPi == null) return undefined
    return topologyHighlightNodeIds(layout, highlightRi, highlightPi)
  }, [layout, highlightRi, highlightPi])

  const handleNodeClick = (node: TopologyNode) => {
    const ri = node.meta.ruleIndex as number | undefined
    const pi = node.meta.pathIndex as number | undefined
    if (ri != null && pi != null) {
      navigate(`/routes/${ri}/${pi}`)
      return
    }
    if (node.type === 'host') {
      const first = rows.find((r) => r.host === node.label)
      if (first) navigate(`/routes/${first.rule_index}/${first.path_index}`)
    }
  }

  return (
    <>
      {err ? <p className="err">{err}</p> : null}
      <div className="topology-container">
        <TopologySVG layout={layout} onNodeClick={handleNodeClick} highlightIds={highlightIds} />
        <div className="topology-legend">
          <span className="legend-item">
            <span className="legend-dot" style={{ background: 'var(--ok)' }} /> 健康
          </span>
          <span className="legend-item">
            <span className="legend-dot" style={{ background: 'var(--warn)' }} /> 告警
          </span>
          <span className="legend-item">
            <span className="legend-dot" style={{ background: 'var(--danger)' }} /> 故障
          </span>
          {highlightIds && highlightIds.size > 0 ? (
            <span className="legend-item chart-hint">高亮为当前路由</span>
          ) : null}
        </div>
      </div>
    </>
  )
}
