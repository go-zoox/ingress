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

interface TopologySVGProps {
  layout: TopologyLayout
  onNodeClick: (node: TopologyNode) => void
}

const NODE_W = 180
const NODE_H = 32
const NODE_RX = 6

/** Map status to CSS color variable */
function statusColor(status: string): string {
  switch (status) {
    case 'up':
    case 'ok':
      return 'var(--ok)'
    case 'warn':
      return 'var(--warn)'
    case 'down':
      return 'var(--danger)'
    default:
      return 'var(--text-muted)'
  }
}

/** Compute a cubic bezier path between two points */
function bezierPath(x1: number, y1: number, x2: number, y2: number): string {
  const cx = (x1 + x2) / 2
  return `M${x1},${y1} C${cx},${y1} ${cx},${y2} ${x2},${y2}`
}

export function TopologySVG({ layout, onNodeClick }: TopologySVGProps) {
  const { nodes, edges, width, height } = layout

  if (nodes.length === 0) {
    return (
      <div className="topology-empty">
        <p className="empty-hint">暂无路由数据，无法生成拓扑图</p>
      </div>
    )
  }

  // Build node position map for edge rendering
  const nodeMap = new Map<string, TopologyNode>()
  for (const n of nodes) {
    nodeMap.set(n.id, n)
  }

  return (
    <svg
      viewBox={`0 0 ${width} ${height}`}
      className="topology-svg"
      width="100%"
      style={{ maxWidth: width, minHeight: 300 }}
    >
      {/* Render edges first (below nodes) */}
      {edges.map((e, i) => {
        const from = nodeMap.get(e.from)
        const to = nodeMap.get(e.to)
        if (!from || !to) return null

        const x1 = from.x + NODE_W
        const y1 = from.y + NODE_H / 2
        const x2 = to.x
        const y2 = to.y + NODE_H / 2

        const strokeColor = e.style === 'danger' ? 'var(--danger)' : 'var(--border)'
        const dashArray = e.style === 'danger' ? '6,3' : undefined

        return (
          <path
            key={`edge-${i}`}
            d={bezierPath(x1, y1, x2, y2)}
            fill="none"
            stroke={strokeColor}
            strokeWidth={1.5}
            strokeDasharray={dashArray}
            className={e.style === 'danger' ? 'topology-edge-danger' : ''}
          />
        )
      })}

      {/* Render nodes */}
      {nodes.map((n) => {
        const borderColor = statusColor(n.status)
        const icon = n.type === 'host' ? '◉' : n.type === 'path' ? '/' : '■'
        return (
          <g
            key={n.id}
            className="topology-node"
            onClick={() => onNodeClick(n)}
            style={{ cursor: 'pointer' }}
          >
            <rect
              x={n.x}
              y={n.y}
              width={NODE_W}
              height={NODE_H}
              rx={NODE_RX}
              fill="var(--bg-elevated)"
              stroke={borderColor}
              strokeWidth={1.5}
              className="topology-node-rect"
            />
            <text
              x={n.x + 10}
              y={n.y + NODE_H / 2}
              dominantBaseline="central"
              fill={borderColor}
              fontSize={12}
              fontFamily="var(--mono)"
            >
              {icon}
            </text>
            <text
              x={n.x + 28}
              y={n.y + NODE_H / 2}
              dominantBaseline="central"
              fill="var(--text)"
              fontSize={12}
              fontFamily="var(--mono)"
            >
              {n.label.length > 20 ? n.label.slice(0, 18) + '…' : n.label}
            </text>
          </g>
        )
      })}
    </svg>
  )
}
