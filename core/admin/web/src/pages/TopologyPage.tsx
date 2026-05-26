import { useEffect, useState, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { PageHeader } from '../components/PageHeader'
import { TopologySVG } from '../components/TopologySVG'
import { api, type RouteRow, type HealthCheckResult, type TLSCert } from '../api/client'

interface TopologyNode {
  id: string
  type: 'host' | 'path' | 'backend'
  label: string
  x: number
  y: number
  status: string
  meta: Record<string, unknown>
}

interface TopologyEdge {
  from: string
  to: string
  style: string
}

interface TopologyLayout {
  nodes: TopologyNode[]
  edges: TopologyEdge[]
  width: number
  height: number
}

export function TopologyPage() {
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

  const layout = useMemo(() => computeLayout(rows, healthMap, certWarnMap), [rows, healthMap, certWarnMap])

  const handleNodeClick = (node: TopologyNode) => {
    const ri = node.meta.ruleIndex as number | undefined
    const pi = node.meta.pathIndex as number | undefined
    if (ri != null && pi != null) {
      navigate(`/routes/${ri}/${pi}`)
      return
    }
    if (node.type === 'host') {
      const host = node.label
      const first = rows.find((r) => r.host === host)
      if (first) {
        navigate(`/routes/${first.rule_index}/${first.path_index}`)
      }
    }
  }

  return (
    <div className="page">
      <PageHeader
        title="拓扑"
        desc="Host → Path → Backend 关系图；叠加健康检查与证书告警，点击进入路由详情"
      />
      {err && <p className="err">{err}</p>}
      <div className="topology-container">
        <TopologySVG layout={layout} onNodeClick={handleNodeClick} />
        <div className="topology-legend">
          <span className="legend-item">
            <span className="legend-dot" style={{ background: 'var(--ok)' }}></span> 健康
          </span>
          <span className="legend-item">
            <span className="legend-dot" style={{ background: 'var(--warn)' }}></span> 告警
          </span>
          <span className="legend-item">
            <span className="legend-dot" style={{ background: 'var(--danger)' }}></span> 故障
          </span>
          <span className="legend-item">
            <span className="legend-dot" style={{ background: 'var(--text-muted)' }}></span> 未知
          </span>
        </div>
      </div>
    </div>
  )
}

function computeLayout(
  rows: RouteRow[],
  healthMap: Map<string, string>,
  certWarnMap: Map<string, string>,
): TopologyLayout {
  if (rows.length === 0) {
    return { nodes: [], edges: [], width: 800, height: 300 }
  }

  const hostMap = new Map<string, RouteRow[]>()
  for (const r of rows) {
    if (!hostMap.has(r.host)) hostMap.set(r.host, [])
    hostMap.get(r.host)!.push(r)
  }

  const colX = [80, 360, 640]
  const nodeH = 36
  const gapY = 48

  const nodes: TopologyNode[] = []
  const edges: TopologyEdge[] = []

  type LayerNode = { id: string; label: string; status: string; meta: Record<string, unknown> }
  const hostNodes: LayerNode[] = []
  const pathNodes: LayerNode[] = []
  const backendNodes: LayerNode[] = []

  const seenHosts = new Set<string>()
  const seenPaths = new Set<string>()
  const seenBackends = new Set<string>()

  const hostPathEdges: { from: string; to: string; style: string }[] = []
  const pathBackendEdges: { from: string; to: string; style: string }[] = []

  for (const [host, hostRows] of hostMap.entries()) {
    const firstRow = hostRows[0]
    if (!seenHosts.has(host)) {
      seenHosts.add(host)
      const certStatus = certWarnMap.get(host)
      let status = 'unknown'
      if (certStatus === 'critical') status = 'down'
      else if (certStatus === 'warning') status = 'warn'
      else status = 'ok'
      hostNodes.push({
        id: `host:${host}`,
        label: host,
        status,
        meta: { ruleIndex: firstRow.rule_index, pathIndex: firstRow.path_index },
      })
    }

    for (const r of hostRows) {
      const pathLabel = r.path_index < 0 ? `${r.path} (规则级)` : r.path
      const pathKey = `path:${host}:${r.rule_index}:${r.path_index}`
      if (!seenPaths.has(pathKey)) {
        seenPaths.add(pathKey)
        pathNodes.push({
          id: pathKey,
          label: pathLabel,
          status: 'ok',
          meta: { ruleIndex: r.rule_index, pathIndex: r.path_index },
        })
      }
      hostPathEdges.push({ from: `host:${host}`, to: pathKey, style: 'solid' })

      const backendKey = `backend:${r.target}`
      if (!seenBackends.has(backendKey)) {
        seenBackends.add(backendKey)
        const healthStatus = healthMap.get(r.target) || 'unknown'
        backendNodes.push({
          id: backendKey,
          label: r.target || '(empty)',
          status: healthStatus,
          meta: { ruleIndex: r.rule_index, pathIndex: r.path_index },
        })
      }
      pathBackendEdges.push({
        from: pathKey,
        to: backendKey,
        style: healthMap.get(r.target) === 'down' ? 'danger' : 'solid',
      })
    }
  }

  const maxLayerSize = Math.max(hostNodes.length, pathNodes.length, backendNodes.length, 1)
  const height = Math.max(maxLayerSize * (nodeH + gapY) + 120, 400)
  const width = 920

  function assignPositions(layer: LayerNode[], colIdx: number) {
    const x = colX[colIdx]
    const startY = 60
    for (let i = 0; i < layer.length; i++) {
      const y = startY + i * (nodeH + gapY)
      nodes.push({
        id: layer[i].id,
        type: colIdx === 0 ? 'host' : colIdx === 1 ? 'path' : 'backend',
        label: layer[i].label,
        x,
        y,
        status: layer[i].status,
        meta: layer[i].meta,
      })
    }
  }

  assignPositions(hostNodes, 0)
  assignPositions(pathNodes, 1)
  assignPositions(backendNodes, 2)

  for (const e of hostPathEdges) edges.push(e)
  for (const e of pathBackendEdges) edges.push(e)

  return { nodes, edges, width, height }
}
