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
  window: string,
): OverviewMetrics {
  if (!patch) return base
  const merged = mergePartial(base, patch)
  if (patch.timeline) {
    const expected = timelineBucketsForWindow(window)
    if (patch.timeline.length !== expected) {
      merged.timeline = base.timeline
    }
  }
  return merged
}

function mergeSystem(
  base: SystemMetrics,
  patch: Partial<SystemMetrics> | undefined,
  window: string,
): SystemMetrics {
  if (!patch) return base
  const merged = mergePartial(base, patch)
  if (patch.timeline) {
    const expected = timelineBucketsForWindow(window)
    if (patch.timeline.length !== expected) {
      merged.timeline = base.timeline
    }
  }
  return merged
}

/** Merge a field-level overview patch onto a base snapshot. */
export function mergeOverviewPatch(
  base: OverviewSnapshot | undefined,
  patch: OverviewSSEPatch,
): OverviewSnapshot {
  const window = normalizeMetricsWindow(patch.window || base?.window || '15m')
  if (!base) {
    return {
      window,
      status: (patch.status || {}) as IngressStatus,
      metrics: (patch.metrics || {}) as OverviewMetrics,
      system: (patch.system || {}) as SystemMetrics,
      certs: patch.certs || [],
      health_checks: patch.health_checks || [],
      health_summary: (patch.health_summary || {
        total: 0,
        up: 0,
        down: 0,
        unknown: 0,
      }) as HealthSummary,
      waf_blocks: patch.waf_blocks || [],
      parse_issues: patch.parse_issues || [],
      revisions: patch.revisions || [],
    }
  }
  return {
    window,
    status: mergePartial(base.status, patch.status),
    metrics: mergeMetrics(base.metrics, patch.metrics, window),
    system: mergeSystem(base.system, patch.system, window),
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
  const expected = normalizeMetricsWindow(metricsWindow)
  const patchWindow = patch.window ? normalizeMetricsWindow(patch.window) : normalizeMetricsWindow(sseWindow)
  return patchWindow !== expected
}
