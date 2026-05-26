type ApiEnvelope<T> = {
  code: number
  message: string
  result: T
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers)
  if (!headers.has('Content-Type') && init?.body) {
    headers.set('Content-Type', 'application/json')
  }
  const res = await fetch(`/api/v1${path}`, { ...init, headers })
  const data = (await res.json()) as ApiEnvelope<T>
  if (!res.ok || data.code >= 400) {
    throw new Error(data.message || res.statusText)
  }
  return data.result
}

export const api = {
  status: () => request<Record<string, unknown>>('/status'),
  routes: () => request<RouteRow[]>('/routes'),
  match: (host: string, path: string) =>
    request<MatchPreview>('/routes/match', {
      method: 'POST',
      body: JSON.stringify({ host, path }),
    }),
  wafEvent: (id: number) => request<WAFEventDetail>(`/waf/events/${id}`),
  wafMatch: (body: WAFTrialInput) =>
    request<WAFTrialResult>('/waf/match', {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  wafEvents: (params?: {
    action?: string
    host?: string
    path?: string
    client_ip?: string
    rule?: string
    time_start?: string
    time_end?: string
    limit?: number
  }) => {
    const q = new URLSearchParams()
    if (params?.action) q.set('action', params.action)
    if (params?.host) q.set('host', params.host)
    if (params?.path) q.set('path', params.path)
    if (params?.client_ip) q.set('client_ip', params.client_ip)
    if (params?.rule) q.set('rule', params.rule)
    if (params?.time_start) q.set('time_start', params.time_start)
    if (params?.time_end) q.set('time_end', params.time_end)
    if (params?.limit) q.set('limit', String(params.limit))
    const qs = q.toString()
    return request<WAFEvent[]>(`/waf/events${qs ? `?${qs}` : ''}`)
  },
  wafToggle: (enabled: boolean | null) =>
    request<{ ok: boolean }>('/waf/toggle', {
      method: 'POST',
      body: JSON.stringify({ enabled }),
    }),
  clearDemoWafEvents: () =>
    request<{ ok: boolean; deleted: number }>('/waf/events/demo-seed', { method: 'DELETE' }),
  wafHosts: () => request<string[]>('/waf/hosts'),
  wafRules: () => request<string[]>('/waf/rules'),
  wafRulesCatalog: () => request<WAFRuleDetail[]>('/waf/rules/catalog'),
  tlsCerts: () => request<TLSCert[]>('/tls/certs'),
  tlsCheck: (domain: string) =>
    request<TLSCertCheck>('/tls/certs/check', {
      method: 'POST',
      body: JSON.stringify({ domain }),
    }),
  cacheOverview: () => request<CacheOverview>('/cache/overview'),
  settings: () => request<SettingsView>('/settings'),
  getConfig: () => request<{ path: string; content: string }>('/config'),
  putConfig: (content: string, note = 'save') =>
    request<{ hash: string }>('/config', {
      method: 'PUT',
      body: JSON.stringify({ content, note }),
    }),
  previewConfig: (content: string) =>
    request<ConfigPreview>('/config/preview', {
      method: 'POST',
      body: JSON.stringify({ content }),
    }),
  publishConfig: (content: string, note = 'publish') =>
    request<{ hash: string; ok: boolean }>('/config/publish', {
      method: 'POST',
      body: JSON.stringify({ content, note }),
    }),
  configModules: (content: string) =>
    request<ConfigModule[]>('/config/modules', {
      method: 'POST',
      body: JSON.stringify({ content }),
    }),
  mergeConfigModule: (content: string, moduleId: string, moduleYaml: string) =>
    request<{ content: string }>('/config/modules/merge', {
      method: 'POST',
      body: JSON.stringify({ content, module_id: moduleId, module_yaml: moduleYaml }),
    }),
  configRevisions: (limit = 50) =>
    request<ConfigRevisionSummary[]>(`/config/revisions?limit=${limit}`),
  configRevision: (id: number) => request<ConfigRevisionDetail>(`/config/revisions/${id}`),
  validateConfig: (content: string) =>
    request<{ valid: boolean }>('/config/validate', {
      method: 'POST',
      body: JSON.stringify({ content }),
    }),
  reload: () => request<{ ok: boolean }>('/reload', { method: 'POST' }),
  overviewMetrics: (window = '15m') =>
    request<OverviewMetrics>(`/metrics/overview?window=${encodeURIComponent(window)}`),
  logs: (params: {
    log?: 'access' | 'error'
    q?: string
    host?: string
    path?: string
    status?: string
    cache_hit?: string
    waf_block?: string
    offset?: number
    limit?: number
  }) => {
    const q = new URLSearchParams()
    if (params.log) q.set('log', params.log)
    if (params.q) q.set('q', params.q)
    if (params.host) q.set('host', params.host)
    if (params.path) q.set('path', params.path)
    if (params.status) q.set('status', params.status)
    if (params.cache_hit) q.set('cache_hit', params.cache_hit)
    if (params.waf_block) q.set('waf_block', params.waf_block)
    if (params.offset != null && params.offset > 0) q.set('offset', String(params.offset))
    if (params.limit) q.set('limit', String(params.limit))
    const qs = q.toString()
    return request<LogResult>(`/logs${qs ? `?${qs}` : ''}`)
  },
  logHosts: () => request<string[]>('/logs/hosts'),
  routeDetail: (ri: number, pi: number) =>
    request<RouteDetail>(`/routes/${ri}/${pi}`),
  routeMetrics: (ri: number, pi: number) =>
    request<RouteMetrics>(`/routes/${ri}/${pi}/metrics`),
  healthCheck: () =>
    request<{ checks: HealthCheckResult[]; summary: HealthSummary }>('/healthcheck'),
  investigate: (params: {
    host: string
    path?: string
    method?: string
    limit?: number
    ri?: number
    pi?: number
  }) => {
    const q = new URLSearchParams()
    q.set('host', params.host)
    q.set('path', params.path || '/')
    if (params.method) q.set('method', params.method)
    if (params.limit) q.set('limit', String(params.limit))
    if (params.ri != null) q.set('ri', String(params.ri))
    if (params.pi != null) q.set('pi', String(params.pi))
    return request<InvestigateResult>(`/investigate?${q.toString()}`)
  },
  sseURL: (channels: string[] = []) => {
    const ch = channels.join(',')
    return `/api/v1/events/stream?channels=${encodeURIComponent(ch)}`
  },
}

export type RouteRow = {
  id: number
  rule_index: number
  path_index: number
  host: string
  host_type: string
  path: string
  backend_type: string
  target: string
  waf: string
  cache: boolean
}

export type MatchPreview = {
  matched: boolean
  rule_index: number
  path_index: number
  host: string
  host_type: string
  path: string
  backend_type: string
  target: string
  fallback?: boolean
  message?: string
}

export type WAFEvent = {
  id: number
  action: string
  rule: string
  host: string
  path: string
  client_ip: string
  created_at: string
}

export type WAFTrialInput = {
  host: string
  path?: string
  method?: string
  client_ip?: string
  query?: string
  headers?: Record<string, string>
  rule_index?: number
  event_id?: number
  expected_rule?: string
}

export type WAFTrialHit = {
  action: string
  rule: string
  client_ip: string
}

export type WAFRuleDetail = {
  id: string
  name: string
  phase: string
  type: string
  pattern?: string
  targets?: string[]
  source: string
  description: string
  log_only?: boolean
}

export type WAFEventDetail = WAFEvent & {
  rule_detail?: WAFRuleDetail | null
  replay_note?: string
}

export type WAFTrialResult = {
  matched: boolean
  would_block: boolean
  rule_index: number
  path_index: number
  host: string
  path: string
  waf_enabled: boolean
  config_waf_enabled: boolean
  runtime_waf_enabled: boolean
  log_only: boolean
  hits: WAFTrialHit[]
  expected_rule?: string
  expected_rule_hit?: boolean
  message?: string
  hint?: string
}

export type TLSCert = {
  domain: string
  certificate: string
  certificate_key: string
  issuer: string
  expires_at: string
  days_remaining: number
  status: string
}

export type TLSCertCheck = {
  domain: string
  certificate: string
  certificate_key: string
  ok: boolean
  status: string
  issuer: string
  subject: string
  expires_at: string
  days_remaining: number
  dns_names: string[]
  checks: Array<{
    id: string
    label: string
    level: 'ok' | 'warn' | 'fail'
    message: string
  }>
}

export type LogResult = {
  lines: string[]
  count: number
  offset: number
}

export type ConfigModule = {
  id: string
  label: string
  keys: string[]
  yaml: string
}

export type ConfigPreview = {
  valid: boolean
  hash: string
  published_hash: string
  changed: boolean
  error?: string
  modules_changed: string[]
}

export type ConfigRevisionSummary = {
  id: number
  hash: string
  note: string
  created_at: string
}

export type ConfigRevisionDetail = ConfigRevisionSummary & {
  content: string
}

export type SettingsView = {
  admin: {
    enabled: boolean
    port: number
    dev_proxy: boolean
    ui_embedded: boolean
  }
  ingress: {
    config_path: string
    pid_file: string
    access_log_path: string
    error_log_path: string
    reload_ready: boolean
    config_hash: string
  }
  database: {
    driver: string
    dsn: string
    waf_events: number
    audit_logs: number
    config_revisions: number
  }
  logs: {
    access_configured: boolean
    access_exists: boolean
    error_configured: boolean
    error_exists: boolean
  }
}

export type CacheOverview = {
  global: {
    enabled: boolean
    engine: string
    ttl: number
    host: string
    port: number
    prefix: string
  }
  routes: Array<{
    id: number
    rule_index: number
    host: string
    path: string
    backend_type: string
    target: string
    ttl: number
    max_body_kb: number
    key_hash: string
  }>
  stats: {
    total_requests: number
    cache_hits: number
    hit_rate: number
    top_hosts: Array<{ host: string; hits: number; total: number; hit_rate: number }>
    top_paths: Array<{ path: string; hits: number; total: number; hit_rate: number }>
  }
}

export type OverviewMetrics = {
  window: string
  source: string
  total: number
  rpm: number
  error_rate: number
  p50_ms: number
  p95_ms: number
  cache_hit_rate: number
  waf_blocks: number
  status_counts: Record<string, number>
  timeline: Array<{
    label: string
    count: number
    '2xx': number
    '3xx': number
    '4xx': number
    '5xx': number
    error_rate: number
    cache_hit_rate: number
    waf_blocks: number
  }>
  top_hosts: Array<{ name: string; count: number }>
  top_hosts_error: Array<{
    name: string
    count: number
    errors: number
    error_rate: number
  }>
  top_paths: Array<{ name: string; count: number }>
  latency_histogram: Array<{ label: string; count: number }>
  delta: {
    total_pct: number
    rpm_pct: number
    error_rate_delta: number
    cache_hit_delta: number
    waf_blocks_delta: number
    p95_delta_ms: number
    has_previous: boolean
  }
  slowest: Array<{
    host: string
    method: string
    path: string
    status: number
    duration_ms: number
  }>
}

// --- New types for route detail, metrics, and health check ---

export type RouteDetail = {
  rule_index: number
  path_index: number
  host: string
  path: string
  backend: {
    type: string
    target: string
    service_name: string
    service_port: number
    service_protocol: string
  }
  auth: {
    type: string
    enabled: boolean
    summary: string
  } | null
  cache: {
    enabled: boolean
    ttl: number
    max_body_kb: number
    key_hash: string
  } | null
  health_check: {
    enabled: boolean
    method: string
    path: string
    status: number[]
    ok: boolean
  } | null
  waf: {
    enabled: boolean
    log_only: boolean
    patched: boolean
  } | null
}

export type RouteMetrics = {
  window: string
  rpm: number
  error_rate: number
  p50_ms: number
  p95_ms: number
  cache_hit_rate: number
  total: number
  timeline: Array<{
    label: string
    count: number
    '2xx': number
    '3xx': number
    '4xx': number
    '5xx': number
  }>
}

export type HealthCheckResult = {
  key: string
  host: string
  path: string
  backend: string
  url: string
  status: string
  last_check: string
  response_ms: number
  error: string
}

export type HealthSummary = {
  total: number
  up: number
  down: number
  unknown: number
}

export type InvestigateSample = {
  at?: string
  client_ip?: string
  method: string
  path: string
  status: number
  duration_ms: number
  target?: string
  upstream_status?: number
  upstream_duration_ms?: number
  cache_hit: boolean
  waf_block: boolean
}

export type InvestigateStats = {
  count: number
  error_rate: number
  p95_ms: number
  cache_hit_rate: number
}

export type InvestigateResult = {
  query: { host: string; path: string; method?: string }
  match: MatchPreview | null
  route: RouteDetail | null
  samples: InvestigateSample[]
  stats: InvestigateStats
  waf_recent: WAFEvent[]
  health_checks: HealthCheckResult[]
}
