/** Build in-app paths with query params for cross-page deep linking. */

export function logsLink(params: {
  host?: string
  path?: string
  waf_block?: string
  cache_hit?: string
  status?: string
  log?: 'access' | 'error'
}) {
  const q = new URLSearchParams()
  if (params.log) q.set('log', params.log)
  if (params.host) q.set('host', params.host)
  if (params.path) q.set('q', params.path)
  if (params.status) q.set('status', params.status)
  if (params.waf_block) q.set('waf_block', params.waf_block)
  if (params.cache_hit) q.set('cache_hit', params.cache_hit)
  const s = q.toString()
  return `/logs${s ? `?${s}` : ''}`
}

export function wafLink(params: {
  action?: string
  host?: string
  path?: string
  rule?: string
  trial?: boolean
  eventId?: number
}) {
  const q = new URLSearchParams()
  if (params.action) q.set('action', params.action)
  if (params.host) q.set('host', params.host)
  if (params.path) q.set('path', params.path)
  if (params.rule) q.set('rule', params.rule)
  if (params.trial) q.set('trial', '1')
  if (params.eventId != null) q.set('event_id', String(params.eventId))
  const s = q.toString()
  return `/waf${s ? `?${s}` : ''}`
}

export function healthLink(params?: { status?: string; host?: string }) {
  const q = new URLSearchParams()
  if (params?.status) q.set('status', params.status)
  if (params?.host) q.set('host', params.host)
  const s = q.toString()
  return `/healths${s ? `?${s}` : ''}`
}

export function routeDetailLink(ruleIndex: number, pathIndex: number) {
  return `/routes/${ruleIndex}/${pathIndex}`
}

export function routesTabLink(
  tab: 'list' | 'topology' | 'match',
  extra?: { highlight_ri?: number; highlight_pi?: number; host?: string },
) {
  const q = new URLSearchParams({ tab })
  if (extra?.highlight_ri != null) q.set('ri', String(extra.highlight_ri))
  if (extra?.highlight_pi != null) q.set('pi', String(extra.highlight_pi))
  if (extra?.host) q.set('host', extra.host)
  return `/routes?${q.toString()}`
}

export function configLink(params?: { focus?: string }) {
  if (!params?.focus) return '/config'
  return `/config?focus=${encodeURIComponent(params.focus)}`
}
