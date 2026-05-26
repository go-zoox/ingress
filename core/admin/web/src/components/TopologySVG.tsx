import { useCallback, useRef, useState } from 'react'

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
  highlightIds?: Set<string>
}

const NODE_W = 180
const NODE_H = 32
const NODE_RX = 6
const MIN_SCALE = 0.35
const MAX_SCALE = 2.5

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

function bezierPath(x1: number, y1: number, x2: number, y2: number): string {
  const cx = (x1 + x2) / 2
  return `M${x1},${y1} C${cx},${y1} ${cx},${y2} ${x2},${y2}`
}

export function TopologySVG({ layout, onNodeClick, highlightIds }: TopologySVGProps) {
  const { nodes, edges, width, height } = layout
  const wrapRef = useRef<HTMLDivElement>(null)
  const [scale, setScale] = useState(1)
  const [pan, setPan] = useState({ x: 0, y: 0 })
  const dragRef = useRef<{ x: number; y: number; panX: number; panY: number } | null>(null)

  const zoomBy = useCallback((delta: number) => {
    setScale((s) => Math.min(MAX_SCALE, Math.max(MIN_SCALE, s + delta)))
  }, [])

  const onWheel = useCallback((e: React.WheelEvent) => {
    e.preventDefault()
    const delta = e.deltaY < 0 ? 0.12 : -0.12
    zoomBy(delta)
  }, [zoomBy])

  const onPointerDown = useCallback((e: React.PointerEvent) => {
    if (e.button !== 0) return
    dragRef.current = { x: e.clientX, y: e.clientY, panX: pan.x, panY: pan.y }
    ;(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId)
  }, [pan])

  const onPointerMove = useCallback((e: React.PointerEvent) => {
    if (!dragRef.current) return
    setPan({
      x: dragRef.current.panX + (e.clientX - dragRef.current.x),
      y: dragRef.current.panY + (e.clientY - dragRef.current.y),
    })
  }, [])

  const onPointerUp = useCallback(() => {
    dragRef.current = null
  }, [])

  if (nodes.length === 0) {
    return (
      <div className="topology-empty">
        <p className="empty-hint">暂无路由数据，无法生成拓扑图</p>
      </div>
    )
  }

  const nodeMap = new Map<string, TopologyNode>()
  for (const n of nodes) {
    nodeMap.set(n.id, n)
  }

  return (
    <div className="topology-viewport" ref={wrapRef}>
      <div className="topology-zoom-toolbar">
        <button type="button" className="btn btn-sm btn-ghost" onClick={() => zoomBy(0.15)} title="放大">
          +
        </button>
        <button type="button" className="btn btn-sm btn-ghost" onClick={() => zoomBy(-0.15)} title="缩小">
          −
        </button>
        <button
          type="button"
          className="btn btn-sm btn-ghost"
          onClick={() => {
            setScale(1)
            setPan({ x: 0, y: 0 })
          }}
          title="重置视图"
        >
          重置
        </button>
        <span className="topology-zoom-label">{Math.round(scale * 100)}%</span>
      </div>
      <div
        className="topology-canvas"
        onWheel={onWheel}
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={onPointerUp}
        onPointerLeave={onPointerUp}
      >
        <svg
          viewBox={`0 0 ${width} ${height}`}
          className="topology-svg"
          style={{
            width: width * scale,
            height: height * scale,
            minWidth: width * scale,
            minHeight: Math.max(300, height * scale * 0.5),
            transform: `translate(${pan.x}px, ${pan.y}px)`,
          }}
        >
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

          {nodes.map((n) => {
            const highlighted = highlightIds?.has(n.id)
            const borderColor = highlighted ? 'var(--accent)' : statusColor(n.status)
            const icon = n.type === 'host' ? '◉' : n.type === 'path' ? '/' : '■'
            return (
              <g
                key={n.id}
                className={`topology-node${highlighted ? ' topology-node-highlight' : ''}`}
                onClick={(ev) => {
                  ev.stopPropagation()
                  onNodeClick(n)
                }}
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
                  strokeWidth={highlighted ? 2.5 : 1.5}
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
                  {n.label.length > 24 ? n.label.slice(0, 22) + '…' : n.label}
                </text>
                <title>{n.label}</title>
              </g>
            )
          })}
        </svg>
      </div>
    </div>
  )
}
