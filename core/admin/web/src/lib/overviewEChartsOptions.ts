import type { EChartsOption } from 'echarts'
import type { OverviewMetrics, MetricsTimelineBucket } from '../api/client'
import type { ChartColors } from '../components/charts/chartTheme'

type AxisMaxOpts = {
  yMax?: number
  y2Max?: number
}

/** Round axis max to readable ticks with ~10% headroom (tracks current data, no sticky ceiling). */
export function niceAxisMax(peak: number, floor = 1): number {
  if (!Number.isFinite(peak) || peak <= 0) return floor
  const padded = peak * 1.1
  if (padded <= 10) {
    return Math.max(floor, Math.ceil(padded * 10) / 10)
  }
  const pow = 10 ** Math.floor(Math.log10(padded))
  const n = padded / pow
  const steps = [1, 1.2, 1.5, 2, 2.5, 3, 4, 5, 6, 7.5, 8, 10]
  for (const step of steps) {
    if (n <= step) {
      return Math.max(floor, step * pow)
    }
  }
  return Math.max(floor, 10 * pow)
}

/** Live overview charts: no entry animation (SSE refresh); axis smoothing is handled separately. */
export function overviewChartMotion(): Pick<EChartsOption, 'animation'> {
  return { animation: false }
}

/** One decimal for percentages (matches overview KPI). */
export function formatChartPercent(value: number): string {
  if (!Number.isFinite(value)) return '—'
  return `${value.toFixed(1)}%`
}

/** Compact axis tick for percent scales. */
export function formatChartPercentAxis(value: number): string {
  if (!Number.isFinite(value)) return ''
  return value >= 10 ? String(Math.round(value)) : value.toFixed(1)
}

/** Whole numbers for counts (WAF blocks, requests). */
export function formatChartInteger(value: number): string {
  if (!Number.isFinite(value)) return '—'
  return String(Math.round(value))
}

function baseGrid(right = 16) {
  return {
    left: 44,
    right,
    top: 24,
    bottom: 36,
    containLabel: false,
  }
}

function categoryAxis(labels: string[], c: ChartColors): EChartsOption['xAxis'] {
  return {
    type: 'category',
    data: labels,
    boundaryGap: false,
    axisLine: { lineStyle: { color: c.grid } },
    axisTick: { show: false },
    axisLabel: { color: c.muted, fontSize: 11 },
  }
}

function countAxis(
  c: ChartColors,
  opts?: { min?: number; max?: number; name?: string; integerTicks?: boolean },
): EChartsOption['yAxis'] {
  return {
    type: 'value',
    min: opts?.min,
    max: opts?.max,
    scale: opts?.max == null,
    name: opts?.name,
    nameTextStyle: { color: c.muted, fontSize: 11 },
    axisLine: { show: false },
    axisTick: { show: false },
    axisLabel: {
      color: c.muted,
      fontSize: 11,
      formatter: opts?.integerTicks ? (v: number) => formatChartInteger(v) : undefined,
    },
    splitLine: { lineStyle: { color: c.grid } },
  }
}

function msAxisLabel(value: number) {
  return value >= 1000 ? `${(value / 1000).toFixed(1)}s` : `${Math.round(value)}ms`
}

function legendBottom(c: ChartColors): EChartsOption['legend'] {
  return {
    bottom: 0,
    itemWidth: 10,
    itemHeight: 8,
    textStyle: { color: c.text, fontSize: 11 },
  }
}

export function buildTrafficTimelineOption(
  timeline: OverviewMetrics['timeline'],
  c: ChartColors,
  axis?: AxisMaxOpts,
): EChartsOption {
  const labels = timeline.map((b) => b.label)
  const stackPeak = timeline.reduce((max, b) => {
    const sum = b['2xx'] + b['3xx'] + b['4xx'] + b['5xx']
    return sum > max ? sum : max
  }, 0)
  const yMax = axis?.yMax ?? niceAxisMax(stackPeak)
  return {
    ...overviewChartMotion(),
    color: [c.ok, c.accent, c.warn, c.danger],
    grid: baseGrid(),
    legend: { ...legendBottom(c), data: ['2xx', '3xx', '4xx', '5xx'] },
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'line' },
      valueFormatter: (v) => formatChartInteger(Number(v)),
    },
    xAxis: categoryAxis(labels, c),
    yAxis: countAxis(c, { integerTicks: true, min: 0, max: yMax }),
    series: [
      {
        name: '2xx',
        type: 'line',
        stack: 'traffic',
        areaStyle: { opacity: 1 },
        lineStyle: { width: 0 },
        symbol: 'none',
        data: timeline.map((b) => b['2xx']),
      },
      {
        name: '3xx',
        type: 'line',
        stack: 'traffic',
        areaStyle: { opacity: 1 },
        lineStyle: { width: 0 },
        symbol: 'none',
        data: timeline.map((b) => b['3xx']),
      },
      {
        name: '4xx',
        type: 'line',
        stack: 'traffic',
        areaStyle: { opacity: 1 },
        lineStyle: { width: 0 },
        symbol: 'none',
        data: timeline.map((b) => b['4xx']),
      },
      {
        name: '5xx',
        type: 'line',
        stack: 'traffic',
        areaStyle: { opacity: 1 },
        lineStyle: { width: 0 },
        symbol: 'none',
        data: timeline.map((b) => b['5xx']),
      },
    ],
  }
}

export function buildQualityTimelineOption(
  timeline: OverviewMetrics['timeline'],
  c: ChartColors,
  axis?: AxisMaxOpts,
): EChartsOption {
  const labels = timeline.map((b) => b.label)
  const errorRates = timeline.map((b) => b.error_rate)
  const wafBlocks = timeline.map((b) => b.waf_blocks)
  const errorPeak = Math.max(0, ...errorRates, 0)
  const wafPeak = Math.max(0, ...wafBlocks, 0)
  const yMax = axis?.yMax ?? niceAxisMax(errorPeak * 1.15, 1)
  const y2Max = axis?.y2Max ?? niceAxisMax(wafPeak * 1.15, 1)

  return {
    ...overviewChartMotion(),
    color: [c.warn, c.danger],
    grid: baseGrid(52),
    legend: { ...legendBottom(c), data: ['错误率 %', 'WAF 拦截'] },
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'line' },
      formatter: (params) => {
        const items = (Array.isArray(params) ? params : [params]) as Array<{
          axisValue?: string
          seriesName?: string
          marker?: string
          value?: number | string
        }>
        if (items.length === 0) return ''
        const head = items[0].axisValue ?? ''
        const lines = items.map((p) => {
          const n = Number(p.value)
          const text =
            p.seriesName === '错误率 %' ? formatChartPercent(n) : formatChartInteger(n)
          return `${p.marker ?? ''}${p.seriesName ?? ''}: ${text}`
        })
        return [head, ...lines].join('<br/>')
      },
    },
    xAxis: categoryAxis(labels, c),
    yAxis: [
      {
        type: 'value',
        min: 0,
        max: yMax,
        name: '错误率 %',
        nameTextStyle: { color: c.warn, fontSize: 11 },
        axisLine: { show: false },
        axisTick: { show: false },
        axisLabel: {
          color: c.warn,
          fontSize: 11,
          formatter: (v: number) => formatChartPercentAxis(v),
        },
        splitLine: { lineStyle: { color: c.grid } },
      },
      {
        type: 'value',
        min: 0,
        max: y2Max,
        name: 'WAF',
        nameTextStyle: { color: c.danger, fontSize: 11 },
        axisLine: { show: false },
        axisTick: { show: false },
        axisLabel: {
          color: c.danger,
          fontSize: 11,
          formatter: (v: number) => formatChartInteger(v),
        },
        splitLine: { show: false },
      },
    ],
    series: [
      {
        name: '错误率 %',
        type: 'line',
        yAxisIndex: 0,
        symbol: 'none',
        lineStyle: { width: 2, color: c.warn },
        data: errorRates,
      },
      {
        name: 'WAF 拦截',
        type: 'line',
        yAxisIndex: 1,
        symbol: 'none',
        lineStyle: { width: 1, color: c.danger },
        areaStyle: { color: c.danger + '55' },
        data: wafBlocks,
      },
    ],
  }
}

export function buildCacheTimelineOption(
  timeline: OverviewMetrics['timeline'],
  c: ChartColors,
): EChartsOption {
  const labels = timeline.map((b) => b.label)
  return {
    ...overviewChartMotion(),
    color: [c.ok],
    grid: baseGrid(),
    legend: { ...legendBottom(c), data: ['缓存命中 %'] },
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'line' },
      valueFormatter: (v) => formatChartPercent(Number(v)),
    },
    xAxis: categoryAxis(labels, c),
    yAxis: {
      type: 'value',
      min: 0,
      max: 100,
      name: '命中率 %',
      nameTextStyle: { color: c.muted, fontSize: 11 },
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: {
        color: c.muted,
        fontSize: 11,
        formatter: (v: number) => formatChartPercentAxis(v),
      },
      splitLine: { lineStyle: { color: c.grid } },
    },
    series: [
      {
        name: '缓存命中 %',
        type: 'line',
        symbol: 'none',
        lineStyle: { width: 2, color: c.ok },
        areaStyle: { color: c.ok + '44' },
        data: timeline.map((b) => b.cache_hit_rate),
      },
    ],
  }
}

export function buildLatencyHistogramOption(
  histogram: OverviewMetrics['latency_histogram'],
  c: ChartColors,
  axis?: AxisMaxOpts,
): EChartsOption {
  const labels = histogram.map((b) => b.label)
  const peak = histogram.reduce((max, b) => (b.count > max ? b.count : max), 0)
  const yMax = axis?.yMax ?? niceAxisMax(peak)
  return {
    ...overviewChartMotion(),
    color: [c.accent],
    grid: baseGrid(),
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      valueFormatter: (v) => formatChartInteger(Number(v)),
    },
    xAxis: categoryAxis(labels, c),
    yAxis: countAxis(c, { integerTicks: true, min: 0, max: yMax }),
    series: [
      {
        name: '请求数',
        type: 'bar',
        barMaxWidth: 28,
        data: histogram.map((b) => b.count),
      },
    ],
  }
}

export function buildUpstreamLatencyTrendOption(
  timeline: MetricsTimelineBucket[],
  c: ChartColors,
): EChartsOption {
  const labels = timeline.map((b) => b.label)
  return {
    ...overviewChartMotion(),
    color: [c.warn],
    grid: baseGrid(),
    legend: { ...legendBottom(c), data: ['上游 P95'] },
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'line' },
      valueFormatter: (v) => msAxisLabel(Number(v)),
    },
    xAxis: categoryAxis(labels, c),
    yAxis: {
      type: 'value',
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: {
        color: c.muted,
        fontSize: 11,
        formatter: (v: number) => msAxisLabel(v),
      },
      splitLine: { lineStyle: { color: c.grid } },
    },
    series: [
      {
        name: '上游 P95',
        type: 'line',
        symbol: 'none',
        lineStyle: { width: 2, color: c.warn },
        areaStyle: { color: c.warn, opacity: 0.35 },
        data: timeline.map((b) => b.upstream_p95_ms ?? 0),
      },
    ],
  }
}

const STATUS_ORDER = ['2xx', '3xx', '4xx', '5xx'] as const

export function buildStatusDonutOption(
  counts: Record<string, number>,
  c: ChartColors,
): EChartsOption | null {
  const total = STATUS_ORDER.reduce((s, k) => s + (counts[k] ?? 0), 0)
  if (total === 0) return null

  const data = STATUS_ORDER.filter((k) => (counts[k] ?? 0) > 0).map((k) => ({
    name: k,
    value: counts[k] ?? 0,
    itemStyle: {
      color: k === '2xx' ? c.ok : k === '3xx' ? c.accent : k === '4xx' ? c.warn : c.danger,
    },
  }))

  return {
    ...overviewChartMotion(),
    series: [
      {
        type: 'pie',
        radius: ['58%', '78%'],
        center: ['50%', '50%'],
        label: { show: false },
        labelLine: { show: false },
        data,
      },
    ],
  }
}
