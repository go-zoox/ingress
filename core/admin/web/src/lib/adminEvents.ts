import type {
  AccessLogParseIssue,
  HealthCheckResult,
  TLSCert,
  WAFEvent,
} from '../api/client'

export type EventStatusTab = 'open' | 'resolved' | 'ignored'

/** Keep in sync with events page list fetch. */
export const OPEN_EVENTS_LIST_LIMIT = 200

export type AdminEventKind = 'waf' | 'parse' | 'health' | 'tls'

export type AdminEvent = {
  key: string
  kind: AdminEventKind
  /** Database id for waf / parse rows */
  entityId?: number
  level: 'danger' | 'warn' | 'info'
  time: string
  timeMs: number
  title: string
  detail: string
  note?: string
  /** Can select for batch resolve / ignore */
  actionable: boolean
}

function formatHostPath(host: string, path: string) {
  const p = (path || '/').trim()
  if (!p || p === '/') return host || '—'
  return p.startsWith('/') ? `${host}${p}` : `${host}/${p}`
}

export function buildAdminEvents(input: {
  statusTab: EventStatusTab
  wafEvents: WAFEvent[]
  parseIssues: AccessLogParseIssue[]
  healthChecks: HealthCheckResult[]
  certs: TLSCert[]
}): AdminEvent[] {
  const out: AdminEvent[] = []

  if (input.statusTab === 'open') {
    for (const e of input.wafEvents) {
      if (e.action !== 'block') continue
      if (!wafIsOpen(e.status)) continue
      const t = new Date(e.created_at).getTime()
      out.push({
        key: `waf-${e.id}`,
        kind: 'waf',
        entityId: e.id,
        level: 'warn',
        time: e.created_at,
        timeMs: t,
        title: `WAF block · ${e.rule}`,
        detail: formatHostPath(e.host, e.path),
        note: e.note,
        actionable: true,
      })
    }
    for (const issue of input.parseIssues) {
      if (issue.status !== 'open') continue
      const t = new Date(issue.last_seen_at || issue.first_seen_at).getTime()
      out.push({
        key: `parse-${issue.id}`,
        kind: 'parse',
        entityId: issue.id,
        level: 'warn',
        time: issue.last_seen_at || issue.first_seen_at,
        timeMs: t,
        title: `日志解析 · ${parseIssueReasonLabel(issue.reason)}`,
        detail: `${issue.hit_count} 次 · ${truncateLine(issue.sample_line)}`,
        note: issue.note,
        actionable: true,
      })
    }
    for (const h of input.healthChecks) {
      if (h.status !== 'down') continue
      const t = h.last_check ? new Date(h.last_check).getTime() : 0
      out.push({
        key: `health-${h.key}`,
        kind: 'health',
        level: 'danger',
        time: h.last_check || '',
        timeMs: t,
        title: `健康检查 DOWN · ${h.host}`,
        detail: h.error || h.url || h.backend,
        actionable: false,
      })
    }
    for (const c of input.certs) {
      if (c.days_remaining >= 30) continue
      const level = c.days_remaining < 7 ? 'danger' : 'warn'
      out.push({
        key: `tls-${c.domain}`,
        kind: 'tls',
        level,
        time: c.expires_at || '',
        timeMs: c.expires_at ? new Date(c.expires_at).getTime() : 0,
        title: `证书 ${level === 'danger' ? '即将过期' : '需关注'} · ${c.domain}`,
        detail: `剩余 ${c.days_remaining} 天`,
        actionable: false,
      })
    }
  } else if (input.statusTab === 'resolved') {
    for (const e of input.wafEvents) {
      if (e.action !== 'block' || e.status !== 'resolved') continue
      const t = new Date(e.created_at).getTime()
      out.push({
        key: `waf-${e.id}`,
        kind: 'waf',
        entityId: e.id,
        level: 'warn',
        time: e.created_at,
        timeMs: t,
        title: `WAF block · ${e.rule}`,
        detail: formatHostPath(e.host, e.path),
        note: e.note,
        actionable: false,
      })
    }
    for (const issue of input.parseIssues) {
      if (issue.status !== 'resolved') continue
      const t = new Date(issue.last_seen_at).getTime()
      out.push({
        key: `parse-${issue.id}`,
        kind: 'parse',
        entityId: issue.id,
        level: 'warn',
        time: issue.last_seen_at,
        timeMs: t,
        title: `日志解析 · ${parseIssueReasonLabel(issue.reason)}`,
        detail: `${issue.hit_count} 次`,
        note: issue.note,
        actionable: false,
      })
    }
  } else {
    for (const e of input.wafEvents) {
      if (e.action !== 'block' || e.status !== 'ignored') continue
      const t = new Date(e.created_at).getTime()
      out.push({
        key: `waf-${e.id}`,
        kind: 'waf',
        entityId: e.id,
        level: 'warn',
        time: e.created_at,
        timeMs: t,
        title: `WAF block · ${e.rule}`,
        detail: formatHostPath(e.host, e.path),
        note: e.note,
        actionable: false,
      })
    }
    for (const issue of input.parseIssues) {
      if (issue.status !== 'ignored') continue
      const t = new Date(issue.last_seen_at || issue.first_seen_at).getTime()
      out.push({
        key: `parse-${issue.id}`,
        kind: 'parse',
        entityId: issue.id,
        level: 'warn',
        time: issue.last_seen_at || issue.first_seen_at,
        timeMs: t,
        title: `日志解析 · ${parseIssueReasonLabel(issue.reason)}`,
        detail: `${issue.hit_count} 次`,
        note: issue.note,
        actionable: false,
      })
    }
  }

  out.sort((a, b) => b.timeMs - a.timeMs)
  return out
}

/** Pending-tab count for nav badge (same inputs as EventsPage open tab). */
export function countOpenAdminEvents(input: {
  wafEvents: WAFEvent[]
  parseIssues: AccessLogParseIssue[]
  healthChecks: HealthCheckResult[]
  certs: TLSCert[]
}): number {
  return buildAdminEvents({ statusTab: 'open', ...input }).length
}

export function wafIsOpen(status?: string) {
  const s = (status || 'open').trim()
  return s === '' || s === 'open'
}

function parseIssueReasonLabel(reason: string) {
  switch (reason) {
    case 'missing_host':
      return '缺少 host'
    case 'missing_request':
      return '缺少 HTTP 请求段'
    case 'empty_after_prefix':
      return '去掉前缀后为空'
    default:
      return '格式不兼容'
  }
}

function truncateLine(line: string) {
  const trimmed = String(line || '').trim()
  if (trimmed.length <= 120) return trimmed || '—'
  return `${trimmed.slice(0, 120)}…`
}

export function kindLabel(kind: AdminEventKind) {
  switch (kind) {
    case 'waf':
      return 'WAF'
    case 'parse':
      return '日志解析'
    case 'health':
      return '健康'
    case 'tls':
      return 'TLS'
  }
}
