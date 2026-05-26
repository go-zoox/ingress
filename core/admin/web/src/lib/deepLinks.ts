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

export function routeDetailLink(
  ruleIndex: number,
  pathIndex: number,
  scope?: { host?: string; path?: string; path_match?: 'prefix' | 'exact' },
) {
  const q = new URLSearchParams()
  if (scope?.host) q.set('host', scope.host)
  if (scope?.path) q.set('path', scope.path)
  if (scope?.path_match) q.set('path_match', scope.path_match)
  const qs = q.toString()
  return `/routes/${ruleIndex}/${pathIndex}${qs ? `?${qs}` : ''}`
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

export function investigateLink(params: {
  host: string
  path: string
  method?: string
  status?: string | number
  ri?: number
  pi?: number
  client_ip?: string
}) {
  const q = new URLSearchParams()
  q.set('host', params.host)
  q.set('path', params.path)
  if (params.method) q.set('method', params.method)
  if (params.status != null && params.status !== '') q.set('status', String(params.status))
  if (params.ri != null) q.set('ri', String(params.ri))
  if (params.pi != null) q.set('pi', String(params.pi))
  if (params.client_ip) q.set('client_ip', params.client_ip)
  return `/investigate?${q.toString()}`
}
