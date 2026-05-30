import type { EChartsOption } from 'echarts'
import type { WAFAttackPoint, WAFVisualization } from '../api/client'

function lineColor(pt: WAFAttackPoint) {
  return pt.block > pt.audit ? '#ff6b6b' : '#5eb8ff'
}

/** 同一路径多束流星，错开 delay，形成连续流动感 */
function meteorStreams(
  points: WAFAttackPoint[],
  server: { lng: number; lat: number; label: string },
  filter: (pt: WAFAttackPoint) => boolean,
) {
  const rows: Array<{
    coords: [number, number][]
    lineStyle: { width: number; opacity: number; curveness: number }
    effect: { delay: number; constantSpeed: number }
    fromName: string
    toName: string
  }> = []

  points.filter(filter).forEach((pt, i) => {
    const streams = Math.min(4, 1 + Math.floor(pt.count / 6))
    for (let n = 0; n < streams; n++) {
      rows.push({
        coords: [
          [pt.lng, pt.lat],
          [server.lng, server.lat],
        ],
        // 底轨隐藏，只看到 effect 拖尾（流星）
        lineStyle: {
          width: 1,
          opacity: 0.01,
          curveness: 0.22 + (i % 4) * 0.05 + n * 0.02,
        },
        effect: {
          delay: n * 1.6 + (i % 5) * 0.35,
          constantSpeed: 26 + (i % 6) * 4 + n * 2,
        },
        fromName: pt.label,
        toName: server.label,
      })
    }
  })

  return rows
}

function meteorLinesSeries(
  name: string,
  color: string,
  data: ReturnType<typeof meteorStreams>,
  zlevel: number,
): EChartsOption['series'] {
  return {
    name,
    type: 'lines',
    coordinateSystem: 'geo',
    zlevel,
    silent: true,
    polyline: false,
    effect: {
      show: true,
      loop: true,
      trailLength: 0.62,
      symbol: 'circle',
      symbolSize: 5,
      color,
    },
    lineStyle: {
      width: 1,
      opacity: 0.01,
      curveness: 0.28,
    },
    data,
  }
}

export function buildWafAttackMapOption(data: WAFVisualization | null): EChartsOption {
  const server = data?.server ?? { lng: 121.4737, lat: 31.2304, label: 'Ingress' }
  const points = data?.points ?? []

  const blockMeteors = meteorStreams(points, server, (pt) => pt.block >= pt.audit)
  const auditMeteors = meteorStreams(points, server, (pt) => pt.audit > pt.block)

  const sourceScatter = points.map((pt) => ({
    name: pt.label,
    value: [pt.lng, pt.lat, pt.count] as [number, number, number],
    itemStyle: { color: lineColor(pt) },
  }))

  const serverScatter = [
    {
      name: server.label,
      value: [server.lng, server.lat, 100] as [number, number, number],
      itemStyle: {
        color: '#3dd6c6',
        shadowBlur: 16,
        shadowColor: 'rgba(61, 214, 198, 0.85)',
      },
    },
  ]

  return {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'item',
      backgroundColor: 'rgba(10, 16, 26, 0.94)',
      borderColor: 'rgba(61, 214, 198, 0.35)',
      textStyle: { color: '#e8eef5', fontSize: 12 },
      formatter: (params: unknown) => {
        const p = params as {
          seriesType?: string
          name?: string
          value?: number[]
          data?: { fromName?: string; toName?: string }
        }
        if (p.seriesType === 'lines' && p.data) {
          return `${p.data.fromName ?? '来源'} → ${p.data.toName ?? server.label}`
        }
        if (p.seriesType === 'effectScatter' && Array.isArray(p.value)) {
          if (p.name === server.label) return `${p.name}<br/>Ingress 节点`
          return `${p.name}<br/>事件 ${p.value[2]}`
        }
        return p.name ?? ''
      },
    },
    geo: {
      map: 'world',
      roam: true,
      zoom: 1.12,
      center: [server.lng, 26],
      aspectScale: 0.82,
      label: { show: false },
      itemStyle: {
        areaColor: '#060b12',
        borderColor: '#2dd4bf',
        borderWidth: 0.7,
        shadowColor: 'rgba(45, 212, 191, 0.12)',
        shadowBlur: 10,
      },
      emphasis: {
        label: { show: false },
        itemStyle: { areaColor: '#0c1520' },
      },
    },
    series: [
      meteorLinesSeries('block-meteor', '#ff6b6b', blockMeteors, 2),
      meteorLinesSeries('audit-meteor', '#5eb8ff', auditMeteors, 2),
      {
        type: 'effectScatter',
        coordinateSystem: 'geo',
        zlevel: 3,
        rippleEffect: {
          brushType: 'stroke',
          scale: 4.5,
          period: 4,
          number: 2,
        },
        symbol: 'circle',
        symbolSize: (val: number[]) => Math.max(7, Math.min(24, 5 + (val[2] ?? 0) / 3)),
        data: sourceScatter,
      },
      {
        type: 'effectScatter',
        coordinateSystem: 'geo',
        zlevel: 4,
        rippleEffect: {
          brushType: 'fill',
          scale: 4,
          period: 2.8,
          number: 3,
        },
        symbol: 'circle',
        symbolSize: 16,
        itemStyle: {
          color: '#3dd6c6',
          shadowBlur: 14,
          shadowColor: 'rgba(61, 214, 198, 0.9)',
        },
        data: serverScatter,
        label: {
          show: true,
          position: 'right',
          formatter: server.label,
          color: '#3dd6c6',
          fontSize: 11,
        },
      },
    ],
  } as EChartsOption
}
