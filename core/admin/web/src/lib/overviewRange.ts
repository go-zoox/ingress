import { normalizeMetricsWindow, normalizeOverviewView } from './metricsWindow'

export type OverviewRangePreset = '5m' | '15m' | '1h' | '6h' | '24h'

export type OverviewRange =
  | { kind: 'preset'; preset: OverviewRangePreset }
  | { kind: 'absolute'; from: string; to: string }

export const OVERVIEW_RANGE_PRESETS: { value: OverviewRangePreset; label: string }[] = [
  { value: '5m', label: '最近 5 分钟' },
  { value: '15m', label: '最近 15 分钟' },
  { value: '1h', label: '最近 1 小时' },
  { value: '6h', label: '最近 6 小时' },
  { value: '24h', label: '最近 24 小时' },
]

const PRESET_SET = new Set<string>(OVERVIEW_RANGE_PRESETS.map((p) => p.value))

/** Slack for treating an absolute range end as "now" (live-eligible). */
export const LIVE_RANGE_END_SLACK_MS = 120_000

export function isOverviewRangePreset(value: string): value is OverviewRangePreset {
  return PRESET_SET.has(value)
}

export function defaultOverviewRange(): OverviewRange {
  return { kind: 'preset', preset: '5m' }
}

/** Parse persisted JSON or legacy metricsWindow string. */
export function parseOverviewRange(raw: string | undefined, legacyWindow?: string): OverviewRange {
  if (raw) {
    try {
      const parsed = JSON.parse(raw) as { kind?: string; preset?: string; from?: string; to?: string }
      // Migrate removed live kind → default rolling 5m preset.
      if (parsed?.kind === 'live') return { kind: 'preset', preset: '5m' }
      if (parsed?.kind === 'preset' && parsed.preset && isOverviewRangePreset(parsed.preset)) {
        return { kind: 'preset', preset: parsed.preset }
      }
      if (parsed?.kind === 'absolute' && parsed.from && parsed.to) {
        return { kind: 'absolute', from: parsed.from, to: parsed.to }
      }
    } catch {
      // fall through
    }
  }
  const legacy = normalizeOverviewView(legacyWindow || '5m')
  if (legacy === 'live') return { kind: 'preset', preset: '5m' }
  if (isOverviewRangePreset(legacy)) return { kind: 'preset', preset: legacy }
  return defaultOverviewRange()
}

export function parseOverviewLiveEnabled(raw: boolean | undefined, legacyWindow?: string): boolean {
  if (raw !== undefined) return raw
  // Legacy "live" view implied streaming updates.
  if (legacyWindow?.trim() === 'live') return true
  return true
}

export function serializeOverviewRange(range: OverviewRange): string {
  return JSON.stringify(range)
}

export function rangeQueryKey(range: OverviewRange): string {
  if (range.kind === 'preset') return range.preset
  return `abs:${range.from}|${range.to}`
}

export function resolveApiWindow(range: OverviewRange): string {
  if (range.kind === 'preset') return range.preset
  return 'custom'
}

const PRESET_DURATION_MS: Record<OverviewRangePreset, number> = {
  '5m': 5 * 60 * 1000,
  '15m': 15 * 60 * 1000,
  '1h': 60 * 60 * 1000,
  '6h': 6 * 60 * 60 * 1000,
  '24h': 24 * 60 * 60 * 1000,
}

/** Resolve any UI range to API { from, to } (RFC3339). */
export function resolveOverviewRangeToQuery(
  range: OverviewRange,
  now = new Date(),
): { from: string; to: string } {
  if (range.kind === 'absolute') {
    return { from: range.from, to: range.to }
  }
  const to = now
  const from = new Date(to.getTime() - PRESET_DURATION_MS[range.preset])
  return { from: from.toISOString(), to: to.toISOString() }
}

export function rangeToQueryParams(range: OverviewRange): { from: string; to: string } {
  return resolveOverviewRangeToQuery(range)
}

/** True when incremental SSE updates make sense for this range. */
export function isRangeLiveEligible(range: OverviewRange, now = new Date()): boolean {
  if (range.kind === 'preset') return true
  const toMs = new Date(range.to).getTime()
  const nowMs = now.getTime()
  if (Number.isNaN(toMs)) return false
  return nowMs - toMs <= LIVE_RANGE_END_SLACK_MS
}

/** SSE subscription params for the selected range. */
export function sseParamsForRange(range: OverviewRange): {
  window?: string
  from?: string
  to?: string
} {
  if (range.kind === 'preset') {
    return { window: range.preset }
  }
  const q = resolveOverviewRangeToQuery(range)
  return { from: q.from, to: q.to }
}

/** Rolling window hint for merging SSE patches onto REST range snapshots. */
export function mergeRollingWindowForRange(range: OverviewRange): string | undefined {
  if (range.kind === 'preset') return range.preset
  return undefined
}

export function formatOverviewRangeLabel(range: OverviewRange): string {
  if (range.kind === 'preset') {
    return OVERVIEW_RANGE_PRESETS.find((p) => p.value === range.preset)?.label ?? '时间范围'
  }
  return `${formatShortDateTime(range.from)} – ${formatShortDateTime(range.to)}`
}

export function formatShortDateTime(iso: string): string {
  try {
    return new Date(iso).toLocaleString('zh-CN', {
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      hour12: false,
    })
  } catch {
    return iso
  }
}

/** Start/end of local calendar day as RFC3339. */
export function localDayBounds(dayOffset: number): { from: string; to: string } {
  const now = new Date()
  const start = new Date(now.getFullYear(), now.getMonth(), now.getDate() + dayOffset, 0, 0, 0, 0)
  const end =
    dayOffset === 0
      ? now
      : new Date(now.getFullYear(), now.getMonth(), now.getDate() + dayOffset, 23, 59, 59, 999)
  return { from: start.toISOString(), to: end.toISOString() }
}

export function toDatetimeLocalValue(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export function fromDatetimeLocalValue(value: string): string {
  if (!value) return ''
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return ''
  return d.toISOString()
}

type RangeMatchable = {
  window?: string
  range_from?: string
  range_to?: string
}

function rangeTimesClose(a: string, b: string, slackMs = 120_000) {
  const ta = new Date(a).getTime()
  const tb = new Date(b).getTime()
  if (Number.isNaN(ta) || Number.isNaN(tb)) return false
  return Math.abs(ta - tb) <= slackMs
}

export function snapshotMatchesRange(
  value: RangeMatchable | null | undefined,
  range: OverviewRange,
): boolean {
  if (!value?.range_from || !value?.range_to) return false
  const expected = resolveOverviewRangeToQuery(range)
  const slack = range.kind === 'absolute' ? 180_000 : 120_000
  return (
    rangeTimesClose(value.range_from, expected.from, slack) &&
    rangeTimesClose(value.range_to, expected.to, slack)
  )
}

/** Legacy bridge for components still using metricsWindow string. */
export function overviewRangeToLegacyView(range: OverviewRange): string {
  if (range.kind === 'preset') return range.preset
  return `custom:${range.from}|${range.to}`
}

export function legacyViewToRangeKey(view: string): string {
  if (view.startsWith('custom:')) return view
  return normalizeMetricsWindow(view)
}
