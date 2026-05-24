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
    if (params.status) q.set('status', params.status)
    if (params.cache_hit) q.set('cache_hit', params.cache_hit)
    if (params.waf_block) q.set('waf_block', params.waf_block)
    if (params.offset != null && params.offset > 0) q.set('offset', String(params.offset))
    if (params.limit) q.set('limit', String(params.limit))
    const qs = q.toString()
    return request<LogResult>(`/logs${qs ? `?${qs}` : ''}`)
  },
}

export type RouteRow = {
  id: number
  rule_index: number
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
  }>
  top_hosts: Array<{ name: string; count: number }>
  slowest: Array<{
    host: string
    method: string
    path: string
    status: number
    duration_ms: number
  }>
}
