import type {
  AccessLogParseIssue,
  ConfigRevisionSummary,
  HealthCheckResult,
  HealthSummary,
  IngressStatus,
  OverviewMetrics,
  OverviewSnapshot,
  SystemMetrics,
  TLSCert,
  WAFEvent,
} from '../api/client'
import { normalizeMetricsWindow, timelineBucketsForWindow } from './metricsWindow'

export type MergeOverviewPatchOptions = {
  /** Rolling window for live SSE (e.g. 5m) when snapshots use window "range". */
  rollingWindow?: string
}

function resolveMergeMetricsWindow(window: string, rollingWindow?: string): string {
  if (window === 'range') {
    if (rollingWindow) return normalizeMetricsWindow(rollingWindow)
    return 'range'
  }
  return normalizeMetricsWindow(window)
}

function timelineBucketExpectation(
  snapshotWindow: string,
  rollingWindow: string | undefined,
  existingLen: number | undefined,
): number {
  const resolved = resolveMergeMetricsWindow(snapshotWindow, rollingWindow)
  if (resolved === 'range' && existingLen != null && existingLen > 0) {
    return existingLen
  }
  return timelineBucketsForWindow(resolved)
}

/** Field-level SSE patch; each section only contains changed keys. */
export type OverviewSSEPatch = {
  window?: string
  seq?: number
  status?: Partial<IngressStatus>
  metrics?: Partial<OverviewMetrics>
  system?: Partial<SystemMetrics>
  health_summary?: Partial<HealthSummary>
  certs?: TLSCert[]
  health_checks?: HealthCheckResult[]
  waf_blocks?: WAFEvent[]
  parse_issues?: AccessLogParseIssue[]
  revisions?: ConfigRevisionSummary[]
}

function mergePartial<T extends object>(base: T, patch: Partial<T> | undefined): T {
  if (!patch) return base
  return { ...base, ...patch }
}

function mergeMetrics(
  base: OverviewMetrics,
  patch: Partial<OverviewMetrics> | undefined,
  snapshotWindow: string,
  rollingWindow?: string,
): OverviewMetrics {
  if (!patch) return base
  const scalarKeys: (keyof OverviewMetrics)[] = [
    'total',
    'rpm',
    'error_rate',
    'p50_ms',
    'p95_ms',
    'cache_hit_rate',
    'waf_blocks',
  ]
  const scalarPatch = scalarKeys.some((key) => patch[key] !== undefined)
  if (scalarPatch && patch.timeline === undefined && !rollingWindow) {
    return base
  }
  const merged = mergePartial(base, patch)
  const expectedBuckets = timelineBucketExpectation(snapshotWindow, rollingWindow, base.timeline?.length)
  if (patch.timeline) {
    if (patch.timeline.length !== expectedBuckets) {
      merged.timeline = base.timeline
    }
  }
  return merged
}

function mergeSystem(
  base: SystemMetrics,
  patch: Partial<SystemMetrics> | undefined,
  snapshotWindow: string,
  rollingWindow?: string,
): SystemMetrics {
  if (!patch) return base
  const merged = mergePartial(base, patch)
  const expectedBuckets = timelineBucketExpectation(snapshotWindow, rollingWindow, base.timeline?.length)
  if (patch.timeline) {
    if (patch.timeline.length !== expectedBuckets) {
      merged.timeline = base.timeline
    }
  }
  return merged
}

/** Merge a field-level overview patch onto a base snapshot. Requires a REST snapshot base. */
export function mergeOverviewPatch(
  base: OverviewSnapshot | undefined,
  patch: OverviewSSEPatch,
  options?: MergeOverviewPatchOptions,
): OverviewSnapshot | null {
  if (!base) {
    return null
  }
  const rollingWindow = options?.rollingWindow
  const baseWindow = resolveMergeMetricsWindow(base.window || '15m', rollingWindow)
  if (patch.window && resolveMergeMetricsWindow(patch.window, rollingWindow) !== baseWindow) {
    return null
  }
  const window = baseWindow
  return {
    window: base.window || window,
    status: mergePartial(base.status, patch.status),
    metrics: mergeMetrics(base.metrics, patch.metrics, base.window || window, rollingWindow),
    system: mergeSystem(base.system, patch.system, base.window || window, rollingWindow),
    certs: patch.certs ?? base.certs,
    health_checks: patch.health_checks ?? base.health_checks,
    health_summary: mergePartial(base.health_summary, patch.health_summary),
    waf_blocks: patch.waf_blocks ?? base.waf_blocks,
    parse_issues: patch.parse_issues ?? base.parse_issues,
    revisions: patch.revisions ?? base.revisions,
  }
}

export function isOverviewSSEPatch(payload: unknown): payload is OverviewSSEPatch {
  if (!payload || typeof payload !== 'object') return false
  return typeof (payload as OverviewSSEPatch).seq === 'number'
}

/** True when patch targets a different metrics window than the UI selection. */
export function overviewPatchWindowMismatch(
  patch: OverviewSSEPatch,
  metricsWindow: string,
  sseWindow: string,
): boolean {
  if (patch.window === 'range') {
    return false
  }
  const expected = normalizeMetricsWindow(metricsWindow)
  const patchWindow = patch.window ? normalizeMetricsWindow(patch.window) : normalizeMetricsWindow(sseWindow)
  return patchWindow !== expected
}
