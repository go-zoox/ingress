import { useCallback, useEffect, useRef, useState } from 'react'
import * as echarts from 'echarts'
import type { ECharts } from 'echarts'
import type { WAFVisualization } from '../api/client'
import { buildWafAttackMapOption } from '../lib/wafAttackMapChart'
import { geoipStatusLabel } from '../lib/geoipStatus'

type Props = {
  data: WAFVisualization | null
  loading?: boolean
}

type MapPoint = WAFVisualization['points'][number]

let worldMapReady: Promise<void> | null = null

function ensureWorldMapRegistered() {
  if (!worldMapReady) {
    worldMapReady = import('../assets/world.geo.json').then((mod) => {
      echarts.registerMap('world', mod.default as unknown as Parameters<typeof echarts.registerMap>[1])
    })
  }
  return worldMapReady
}

export function WafAttackMap({ data, loading }: Props) {
  const chartRef = useRef<HTMLDivElement>(null)
  const chartInst = useRef<ECharts | null>(null)
  const [mapReady, setMapReady] = useState(false)
  const [hover, setHover] = useState<MapPoint | null>(null)

  const points = data?.points ?? []
  const topPoints = points.slice(0, 8)

  const geoipHint = geoipStatusLabel(data?.geoip)

  const highlightLabel = useCallback(
    (label: string | null) => {
      const chart = chartInst.current
      if (!chart) return
      chart.dispatchAction({ type: 'downplay', seriesIndex: 2 })
      if (!label) return
      const idx = points.findIndex((p) => p.label === label)
      if (idx >= 0) {
        chart.dispatchAction({ type: 'highlight', seriesIndex: 2, dataIndex: idx })
        chart.dispatchAction({ type: 'showTip', seriesIndex: 2, dataIndex: idx })
      }
    },
    [points],
  )

  useEffect(() => {
    let cancelled = false
    ensureWorldMapRegistered()
      .then(() => {
        if (!cancelled) setMapReady(true)
      })
      .catch(() => {
        if (!cancelled) setMapReady(false)
      })
    return () => {
      cancelled = true
    }
  }, [])

  useEffect(() => {
    const el = chartRef.current
    if (!el || !mapReady) return

    const chart = echarts.init(el, undefined, { renderer: 'canvas' })
    chartInst.current = chart

    const ro = new ResizeObserver(() => chart.resize())
    ro.observe(el)

    return () => {
      ro.disconnect()
      chart.dispose()
      chartInst.current = null
    }
  }, [mapReady])

  useEffect(() => {
    const chart = chartInst.current
    if (!chart || !mapReady) return
    chart.setOption(buildWafAttackMapOption(data), { notMerge: true, lazyUpdate: false })
  }, [data, mapReady])

  return (
    <div className="waf-map-panel">
      <div className="waf-map-layout">
        <div className="waf-map-canvas-wrap">
          <div ref={chartRef} className="waf-map-chart" aria-label="WAF 攻击世界地图可视化" role="img" />
          {loading || !mapReady ? <div className="waf-map-overlay">加载中…</div> : null}
          {mapReady && !loading && points.length === 0 ? (
            <div className="waf-map-overlay waf-map-overlay--empty">暂无地理定位的攻击数据</div>
          ) : null}
        </div>

        <div className="waf-map-side">
          <div className="waf-map-stats">
            <div className="waf-map-stat">
              <span className="waf-map-stat-val">{data?.total ?? 0}</span>
              <span className="waf-map-stat-label">事件</span>
            </div>
            <div className="waf-map-stat">
              <span className="waf-map-stat-val">{points.length}</span>
              <span className="waf-map-stat-label">来源地</span>
            </div>
            {(data?.unknown_ips ?? 0) > 0 ? (
              <div className="waf-map-stat">
                <span className="waf-map-stat-val">{data?.unknown_ips}</span>
                <span className="waf-map-stat-label">内网/未知</span>
              </div>
            ) : null}
          </div>
          <ul className="waf-map-legend">
            <li><span className="waf-map-dot waf-map-dot--block" /> Block</li>
            <li><span className="waf-map-dot waf-map-dot--audit" /> Audit</li>
            <li><span className="waf-map-dot waf-map-dot--server" /> Ingress</li>
          </ul>
          {topPoints.length > 0 ? (
            <ol className="waf-map-rank">
              {topPoints.map((pt) => (
                <li
                  key={`${pt.lat}-${pt.lng}-${pt.label}`}
                  onMouseEnter={() => {
                    setHover(pt)
                    highlightLabel(pt.label)
                  }}
                  onMouseLeave={() => {
                    setHover(null)
                    highlightLabel(null)
                    chartInst.current?.dispatchAction({ type: 'hideTip' })
                  }}
                  className={hover?.label === pt.label ? 'active' : ''}
                >
                  <span className="waf-map-rank-label">{pt.label}{pt.approx ? ' *' : ''}</span>
                  <span className="waf-map-rank-count">{pt.count}</span>
                </li>
              ))}
            </ol>
          ) : null}
          <p className="waf-map-hint">拖拽缩放 · 流星飞线从来源射向 Ingress</p>
          <p className="waf-map-hint waf-map-hint--geoip">{geoipHint}</p>
        </div>
      </div>
    </div>
  )
}
