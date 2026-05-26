import type { HealthCheckResult, TLSCert, WAFEvent } from '../api/client'
import { healthLink, logsLink, wafLink } from './deepLinks'

export type FeedEvent = {
  id: string
  kind: 'waf' | 'health' | 'tls'
  level: 'danger' | 'warn' | 'info'
  time: string
  timeMs: number
  title: string
  detail: string
  href: string
  actions?: Array<{ label: string; href: string }>
}

export function buildEventsFeed(
  wafEvents: WAFEvent[],
  healthChecks: HealthCheckResult[],
  certs: TLSCert[],
): FeedEvent[] {
  const out: FeedEvent[] = []

  for (const e of wafEvents) {
    if (e.action !== 'block') continue
    const t = new Date(e.created_at).getTime()
    out.push({
      id: `waf-${e.id}`,
      kind: 'waf',
      level: 'warn',
      time: e.created_at,
      timeMs: t,
      title: `WAF block · ${e.rule}`,
      detail: `${e.host}${e.path}`,
      href: wafLink({ action: 'block', host: e.host, path: e.path }),
      actions: [
        { label: '查日志', href: logsLink({ host: e.host, waf_block: '1', log: 'access' }) },
        { label: '试匹配', href: wafLink({ host: e.host, path: e.path, trial: true, eventId: e.id }) },
      ],
    })
  }

  for (const h of healthChecks) {
    if (h.status !== 'down') continue
    const t = h.last_check ? new Date(h.last_check).getTime() : 0
    out.push({
      id: `health-${h.key}`,
      kind: 'health',
      level: 'danger',
      time: h.last_check || '',
      timeMs: t,
      title: `健康检查 DOWN · ${h.host}`,
      detail: h.error || h.url || h.backend,
      href: healthLink({ status: 'down', host: h.host }),
      actions: [
        { label: '查日志', href: logsLink({ host: h.host, log: 'access' }) },
        { label: '健康检查', href: healthLink({ status: 'down' }) },
      ],
    })
  }

  for (const c of certs) {
    if (c.days_remaining >= 30) continue
    const level = c.days_remaining < 7 ? 'danger' : 'warn'
    out.push({
      id: `tls-${c.domain}`,
      kind: 'tls',
      level,
      time: c.expires_at || '',
      timeMs: c.expires_at ? new Date(c.expires_at).getTime() : 0,
      title: `证书 ${level === 'danger' ? '即将过期' : '需关注'} · ${c.domain}`,
      detail: `剩余 ${c.days_remaining} 天`,
      href: '/tls',
      actions: [{ label: '证书管理', href: '/tls' }],
    })
  }

  out.sort((a, b) => b.timeMs - a.timeMs)
  return out
}
