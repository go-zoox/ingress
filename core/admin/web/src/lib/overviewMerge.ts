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

/** Merge a field-level overview patch onto a base snapshot. */
export function mergeOverviewPatch(
  base: OverviewSnapshot | undefined,
  patch: OverviewSSEPatch,
): OverviewSnapshot {
  if (!base) {
    return {
      window: patch.window || '15m',
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
    window: patch.window || base.window,
    status: mergePartial(base.status, patch.status),
    metrics: mergePartial(base.metrics, patch.metrics),
    system: mergePartial(base.system, patch.system),
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
