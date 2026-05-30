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
  const res = await fetch(`/api/v1${path}`, { ...init, headers, credentials: 'include' })
  const data = (await res.json()) as ApiEnvelope<T>
  if (!res.ok || data.code >= 400) {
    throw new Error(data.message || res.statusText)
  }
  return data.result
}

export const api = {
  authConfig: () => request<AuthConfigView>('/auth/config'),
  authLogin: (username: string, password: string) =>
    request<{ ok: boolean; user: AuthUser }>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    }),
  authLogout: () => request<{ ok: boolean }>('/auth/logout', { method: 'POST' }),
  status: () => request<IngressStatus>('/status'),
  routes: () => request<RouteRow[]>('/routes'),
  match: (host: string, path: string) =>
    request<MatchPreview>('/routes/match', {
      method: 'POST',
      body: JSON.stringify({ host, path }),
    }),
  wafEvent: (id: number) => request<WAFEventDetail>(`/waf/events/${id}`),
  updateWafEventStatus: (id: number, status: 'ignored' | 'resolved' | 'open', note = '') =>
    request<WAFEvent>(`/waf/events/${id}/status`, {
      method: 'POST',
      body: JSON.stringify({ status, note }),
    }),
  batchUpdateWafEventStatus: (ids: number[], status: 'ignored' | 'resolved' | 'open', note = '') =>
    request<{ ok: boolean; updated: number }>('/waf/events/batch-status', {
      method: 'POST',
      body: JSON.stringify({ ids, status, note }),
    }),
  wafMatch: (body: WAFTrialInput) =>
    request<WAFTrialResult>('/waf/match', {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  wafEvents: (params?: {
    action?: string
    host?: string
    path?: string
    path_match?: 'prefix' | 'exact'
    client_ip?: string
    rule?: string
    status?: string
    time_start?: string
    time_end?: string
    limit?: number
    ri?: number
    pi?: number
  }) => {
    const q = new URLSearchParams()
    if (params?.action) q.set('action', params.action)
    if (params?.host) q.set('host', params.host)
    if (params?.path) q.set('path', params.path)
    if (params?.path_match) q.set('path_match', params.path_match)
    if (params?.ri != null) q.set('ri', String(params.ri))
    if (params?.pi != null) q.set('pi', String(params.pi))
    if (params?.client_ip) q.set('client_ip', params.client_ip)
    if (params?.rule) q.set('rule', params.rule)
    if (params?.status) q.set('status', params.status)
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
  wafVisualization: (params?: {
    action?: string
    host?: string
    path?: string
    client_ip?: string
    rule?: string
    time_start?: string
    time_end?: string
  }) => {
    const q = new URLSearchParams()
    if (params?.action) q.set('action', params.action)
    if (params?.host) q.set('host', params.host)
    if (params?.path) q.set('path', params.path)
    if (params?.client_ip) q.set('client_ip', params.client_ip)
    if (params?.rule) q.set('rule', params.rule)
    if (params?.time_start) q.set('time_start', params.time_start)
    if (params?.time_end) q.set('time_end', params.time_end)
    const qs = q.toString()
    return request<WAFVisualization>(`/waf/visualization${qs ? `?${qs}` : ''}`)
  },
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
  auditLogs: (limit = 50) => request<AuditLogRow[]>(`/audit/logs?limit=${limit}`),
  validateConfig: (content: string) =>
    request<{ valid: boolean }>('/config/validate', {
      method: 'POST',
      body: JSON.stringify({ content }),
    }),
  reload: () => request<{ ok: boolean }>('/reload', { method: 'POST' }),
  scenarios: () => request<ScenariosResult>('/scenarios'),
  setScenarioActive: (id: string) =>
    request<ScenariosResult>('/scenarios/active', {
      method: 'PUT',
      body: JSON.stringify({ id }),
    }),
  overviewMetrics: (window = '15m') =>
    request<OverviewMetrics>(`/metrics/overview?window=${encodeURIComponent(window)}`),
  overviewSnapshot: (window = '15m') =>
    request<OverviewSnapshot>(`/overview/snapshot?window=${encodeURIComponent(window)}`),
  systemMetrics: (window = '15m') =>
    request<SystemMetrics>(`/metrics/system?window=${encodeURIComponent(window)}`),
  parseIssues: (status = 'open', limit = 20) =>
    request<AccessLogParseIssue[]>(
      `/logs/parse-issues?status=${encodeURIComponent(status)}&limit=${limit}`,
    ),
  updateParseIssueStatus: (id: number, status: 'ignored' | 'resolved' | 'open', note = '') =>
    request<AccessLogParseIssue>(`/logs/parse-issues/${id}/status`, {
      method: 'POST',
      body: JSON.stringify({ status, note }),
    }),
  batchUpdateParseIssueStatus: (ids: number[], status: 'ignored' | 'resolved' | 'open', note = '') =>
    request<{ ok: boolean; updated: number }>('/logs/parse-issues/batch-status', {
      method: 'POST',
      body: JSON.stringify({ ids, status, note }),
    }),
  parseIssueDetail: (id: number) => request<AccessLogParseIssueDetail>(`/logs/parse-issues/${id}`),
  logs: (params: {
    log?: 'access' | 'error'
    q?: string
    host?: string
    path?: string
    path_match?: 'prefix' | 'exact'
    status?: string
    cache_hit?: string
    waf_block?: string
    offset?: number
    limit?: number
    ri?: number
    pi?: number
  }) => {
    const q = new URLSearchParams()
    if (params.log) q.set('log', params.log)
    if (params.q) q.set('q', params.q)
    if (params.host) q.set('host', params.host)
    if (params.path) q.set('path', params.path)
    if (params.path_match) q.set('path_match', params.path_match)
    if (params.ri != null) q.set('ri', String(params.ri))
    if (params.pi != null) q.set('pi', String(params.pi))
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
  routeMetrics: (
    ri: number,
    pi: number,
    window?: string,
    scope?: { host?: string; path?: string; path_match?: 'prefix' | 'exact' },
  ) => {
    const q = new URLSearchParams()
    if (window) q.set('window', window)
    if (scope?.host) q.set('host', scope.host)
    if (scope?.path) q.set('path', scope.path)
    if (scope?.path_match) q.set('path_match', scope.path_match)
    const qs = q.toString()
    return request<RouteMetrics>(`/routes/${ri}/${pi}/metrics${qs ? `?${qs}` : ''}`)
  },
  serviceDetail: (name: string) =>
    request<ServiceDetail>(`/services/${encodeURIComponent(name)}`),
  serviceMetrics: (name: string, window?: string) => {
    const q = new URLSearchParams()
    if (window) q.set('window', window)
    const qs = q.toString()
    return request<ServiceMetrics>(
      `/services/${encodeURIComponent(name)}/metrics${qs ? `?${qs}` : ''}`,
    )
  },
  healthCheck: () =>
    request<{ checks: HealthCheckResult[]; summary: HealthSummary }>('/healthcheck'),
  jobs: () => request<JobsListResult>('/jobs'),
  jobsCapabilities: () => request<JobsCapabilities>('/jobs/capabilities'),
  jobRunsForJob: (source: 'builtin' | 'config', id: string, limit = 30) => {
    const q = new URLSearchParams()
    if (limit) q.set('limit', String(limit))
    const qs = q.toString()
    return request<JobRunRow[]>(
      `/jobs/${source}/${encodeURIComponent(id)}/runs${qs ? `?${qs}` : ''}`,
    )
  },
  jobRunDetail: (runId: number) => request<JobRunRow>(`/jobs/runs/${runId}`),
  updateBuiltinJob: (id: string, body: BuiltinJobPatch) =>
    request<{ ok: boolean }>(`/jobs/builtins/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: JSON.stringify(body),
    }),
  createJobItem: (body: JobItemInput) =>
    request<{ ok: boolean }>('/jobs/items', {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  updateJobItem: (id: string, body: JobItemInput) =>
    request<{ ok: boolean }>(`/jobs/items/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: JSON.stringify(body),
    }),
  deleteJobItem: (id: string) =>
    request<{ ok: boolean }>(`/jobs/items/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  runJob: (source: 'builtin' | 'config', id: string) =>
    request<JobRunRow>(`/jobs/${source}/${encodeURIComponent(id)}/run`, { method: 'POST' }),
  rbacPermissions: () => request<RBACPermissionRow[]>('/rbac/permissions'),
  createRbacPermission: (body: RBACPermissionInput) =>
    request<RBACPermissionRow>('/rbac/permissions', {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  updateRbacPermission: (id: number, body: RBACPermissionInput) =>
    request<RBACPermissionRow>(`/rbac/permissions/${id}`, {
      method: 'PUT',
      body: JSON.stringify(body),
    }),
  deleteRbacPermission: (id: number) =>
    request<{ ok: boolean }>(`/rbac/permissions/${id}`, { method: 'DELETE' }),
  rbacRoles: () => request<RBACRoleRow[]>('/rbac/roles'),
  createRbacRole: (body: RBACRoleInput) =>
    request<RBACRoleRow>('/rbac/roles', {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  updateRbacRole: (id: number, body: RBACRoleInput) =>
    request<RBACRoleRow>(`/rbac/roles/${id}`, {
      method: 'PUT',
      body: JSON.stringify(body),
    }),
  deleteRbacRole: (id: number) =>
    request<{ ok: boolean }>(`/rbac/roles/${id}`, { method: 'DELETE' }),
  rbacUsers: () => request<RBACUserRow[]>('/rbac/users'),
  createRbacUser: (body: RBACUserInput) =>
    request<RBACUserRow>('/rbac/users', {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  updateRbacUser: (id: number, body: RBACUserInput) =>
    request<RBACUserRow>(`/rbac/users/${id}`, {
      method: 'PUT',
      body: JSON.stringify(body),
    }),
  updateRbacUserPassword: (id: number, password: string) =>
    request<{ ok: boolean }>(`/rbac/users/${id}/password`, {
      method: 'PUT',
      body: JSON.stringify({ password }),
    }),
  deleteRbacUser: (id: number) =>
    request<{ ok: boolean }>(`/rbac/users/${id}`, { method: 'DELETE' }),
  rbacMenus: () => request<NavMenuResult>('/rbac/menus'),
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
  auth?: string
  health_check?: string
  maintenance?: string
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
  user_agent?: string
  status?: 'open' | 'ignored' | 'resolved' | ''
  note?: string
  created_at: string
}

export type WAFAttackPoint = {
  lat: number
  lng: number
  label: string
  count: number
  block: number
  audit: number
  ips: string[]
  approx?: boolean
}

export type WAFVisualization = {
  points: WAFAttackPoint[]
  total: number
  unknown_ips: number
  server: { lat: number; lng: number; label: string }
  geoip?: {
    configured?: boolean
    enabled: boolean
    loaded: boolean
    source: 'maxmind' | 'fallback' | string
    database?: string
    error?: string
    reason?: string
  }
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
  action?: string
  enabled: boolean
  builtin?: boolean
}

export type WAFEventDetail = WAFEvent & {
  rule_detail?: WAFRuleDetail | null
  replay_note?: string
  status?: 'open' | 'ignored' | 'resolved' | ''
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

export type IngressStatus = {
  version?: string
  config_path?: string
  pid_file?: string
  reload_ready?: boolean
  config_hash?: string
  file_hash?: string
  runtime_hash?: string
  latest_revision_hash?: string
  runtime_drift?: boolean
  revision_drift?: boolean
  listen_http?: number | string
  listen_https?: number | string
  rules_count?: number
  waf_enabled?: boolean
  waf_log_only?: boolean
  waf_runtime_enabled?: boolean
  last_reload?: string
}

export type ConfigModule = {
  id: string
  label: string
  keys: string[]
  yaml: string
}

export type ConfigRouteImpact = {
  kind: 'added' | 'removed' | 'changed'
  host: string
  path: string
  rule_index: number
  path_index: number
  fields?: string[]
  before?: string
  after?: string
}

export type ConfigPreview = {
  valid: boolean
  hash: string
  published_hash: string
  changed: boolean
  error?: string
  modules_changed: string[]
  global_touches?: string[]
  route_impacts?: ConfigRouteImpact[]
}

export type AuditLogRow = {
  id: number
  action: string
  detail: string
  actor: string
  created_at: string
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
  geoip: {
    database?: string
    ingress_label?: string
    ingress_lat?: number
    ingress_lng?: number
    database_exists: boolean
    database_readable: boolean
    runtime: {
      configured?: boolean
      enabled: boolean
      loaded: boolean
      source: string
      database?: string
      error?: string
      reason?: string
    }
    ingress: {
      lat: number
      lng: number
      label: string
    }
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

export type BackendStat = {
  name: string
  count: number
  rpm: number
  upstream_p95_ms: number
  upstream_error_pct: number
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
    p50_ms?: number
    p95_ms?: number
    upstream_p95_ms?: number
  }>
  top_hosts: Array<{ name: string; count: number }>
  top_hosts_error: Array<{
    name: string
    count: number
    errors: number
    error_rate: number
  }>
  host_traffic?: Array<{ name: string; pv: number; uv: number }>
  top_paths: Array<{ name: string; count: number }>
  top_backends?: BackendStat[]
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
  parse_skipped?: number
  parse_issue_open?: number
  parseable_in_tail?: number
  window_stale?: boolean
  slowest?: Array<{
    host: string
    method: string
    path: string
    status: number
    duration_ms: number
  }>
}

export type SystemMetrics = {
  window: string
  cpu_pct: number
  memory_mb: number
  goroutines: number
  num_cpu: number
  timeline: Array<{
    label: string
    cpu_pct: number
    memory_mb: number
  }>
}

export type OverviewSnapshot = {
  window: string
  status: IngressStatus
  metrics: OverviewMetrics
  system: SystemMetrics
  certs: TLSCert[]
  health_checks: HealthCheckResult[]
  health_summary: HealthSummary
  waf_blocks: WAFEvent[]
  parse_issues: AccessLogParseIssue[]
  revisions: ConfigRevisionSummary[]
}

export type AccessLogParseIssue = {
  id: number
  fingerprint: string
  sample_line: string
  reason: string
  hit_count: number
  status: 'open' | 'ignored' | 'resolved'
  first_seen_at: string
  last_seen_at: string
  note?: string
}

export type AccessLogParseIssueDetail = AccessLogParseIssue & {
  diagnosis: {
    reason: string
    reason_label: string
    hint: string
    has_host: boolean
    has_request: boolean
    sample_line: string
  }
  context: Array<{
    line: string
    match: boolean
    parsed: boolean
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
    default?: string
    path_rules?: number
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

export type MetricsTimelineBucket = {
  label: string
  count: number
  '2xx': number
  '3xx': number
  '4xx': number
  '5xx': number
  error_rate?: number
  cache_hit_rate?: number
  waf_blocks?: number
  p50_ms?: number
  p95_ms?: number
  upstream_p95_ms?: number
}

export type RouteSampleRow = {
  host: string
  method: string
  path: string
  status: number
  duration_ms: number
}

export type MetricsDelta = OverviewMetrics['delta']

export type RouteUpstreamStats = {
  samples: number
  avg_total_ms: number
  avg_upstream_ms: number
  avg_gateway_ms: number
  upstream_error_pct: number
}

export type RoutePathBreakdown = {
  path_index: number
  path: string
  count: number
  error_rate: number
}

export type RouteCompareStats = {
  site_rpm: number
  site_error_rate: number
  route_share_pct: number
  error_rate_vs_site: number
}

export type ServiceRouteRef = {
  rule_index: number
  path_index: number
  host: string
  path: string
  target: string
  backend_type: string
}

export type ServiceDetail = {
  name: string
  catalog_index: number
  target: string
  protocol: string
  port: number
  mode: string
  note: string
  health_check: {
    enabled: boolean
    method: string
    path: string
    status: number[]
    ok: boolean
  } | null
  route_refs: ServiceRouteRef[]
  route_ref_count: number
  target_aliases: string[]
}

export type ServiceCompareStats = {
  site_rpm: number
  site_error_rate: number
  service_share_pct: number
  error_rate_vs_site: number
}

export type ServiceMetrics = {
  window: string
  source?: string
  rpm: number
  error_rate: number
  p50_ms: number
  p95_ms: number
  cache_hit_rate: number
  waf_blocks?: number
  total: number
  status_counts: Record<string, number>
  timeline: MetricsTimelineBucket[]
  slowest?: RouteSampleRow[]
  error_samples?: RouteSampleRow[]
  latency_histogram?: Array<{ label: string; count: number }>
  top_hosts?: Array<{ name: string; count: number }>
  top_paths?: Array<{ name: string; count: number }>
  delta?: MetricsDelta
  upstream?: RouteUpstreamStats
  compare?: ServiceCompareStats
  target_aliases?: string[]
  health_checks?: HealthCheckResult[]
  health_summary?: HealthSummary
}

export type RouteCacheStats = {
  enabled: boolean
  ttl: number
  max_body_kb: number
  hits: number
  total: number
  hit_rate: number
}

export type HealthProbePoint = {
  at: string
  status: string
  response_ms: number
}

export type RelatedRouteRow = {
  rule_index: number
  path_index: number
  host: string
  path: string
  target: string
  relation: string
}

export type RouteMetrics = {
  window: string
  source?: string
  rpm: number
  error_rate: number
  p50_ms: number
  p95_ms: number
  cache_hit_rate: number
  waf_blocks?: number
  total: number
  status_counts: Record<string, number>
  timeline: MetricsTimelineBucket[]
  slowest?: RouteSampleRow[]
  error_samples?: RouteSampleRow[]
  latency_histogram?: Array<{ label: string; count: number }>
  top_hosts?: Array<{ name: string; count: number }>
  top_paths?: Array<{ name: string; count: number }>
  host_traffic?: Array<{ name: string; pv: number; uv: number }>
  scope_hosts?: Array<{ name: string; count: number }>
  scope_paths?: Array<{ name: string; count: number }>
  scope_host_traffic?: Array<{ name: string; pv: number; uv: number }>
  delta?: MetricsDelta
  upstream?: RouteUpstreamStats
  path_breakdown?: RoutePathBreakdown[]
  waf_top_rules?: Array<{ name: string; count: number }>
  health_history?: HealthProbePoint[]
  compare?: RouteCompareStats
  related_routes?: RelatedRouteRow[]
  route_cache?: RouteCacheStats
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

export type JobParams = {
  method?: string
  url?: string
  headers?: Record<string, string>
  body?: string
  expect_status?: number[]
  insecure_tls?: boolean
  script?: string
  engine?: string
  shell?: string
  workdir?: string
  env?: Record<string, string>
  /** @deprecated legacy script migration */
  command?: string
  /** @deprecated legacy script migration */
  args?: string[]
  retain_days?: number
}

export type JobView = {
  id: string
  name: string
  source: 'builtin' | 'config'
  kind: string
  description?: string
  schedule: string
  enabled: boolean
  timeout_sec?: number
  on_failure?: string
  params: JobParams
  deletable: boolean
  editable: boolean
  last_run?: JobRunRow
}

export type JobsCapabilities = {
  http_call: boolean
  command: boolean
  allow_command: boolean
  command_restricted?: boolean
  command_allowlist?: string[]
  command_reason?: string
}

export type JobsListResult = {
  capabilities: JobsCapabilities
  builtins: JobView[]
  items: JobView[]
}

export type JobRunResult = {
  http?: {
    status_code: number
    headers: Record<string, string>
    body: string
  }
  command?: {
    log: string
  }
  message?: string
}

export type JobRunRow = {
  id: number
  job_id: string
  source: string
  kind: string
  status: 'running' | 'success' | 'failed'
  duration_ms: number
  output_preview?: string
  result?: JobRunResult
  error?: string
  trigger: string
  started_at: string
  finished_at: string
}

export type BuiltinJobPatch = {
  enabled?: boolean
  schedule?: string
  params?: JobParams
}

export type JobItemInput = {
  id?: string
  name: string
  kind: 'http_call' | 'script'
  schedule: string
  enabled: boolean
  timeout_sec?: number
  on_failure?: string
  params: JobParams
}

export type RBACPermissionRow = {
  id: number
  code: string
  name: string
  group: string
  description?: string
  builtin: boolean
  role_count: number
}

export type RBACPermissionInput = {
  code: string
  name: string
  group: string
  description?: string
}

export type RBACRoleRow = {
  id: number
  code: string
  name: string
  description?: string
  builtin: boolean
  permission_ids: number[]
  permissions?: string[]
  user_count: number
}

export type RBACRoleInput = {
  code: string
  name: string
  description?: string
  permission_ids: number[]
}

export type RBACUserRow = {
  id: number
  username: string
  display_name: string
  email?: string
  enabled: boolean
  builtin: boolean
  role_ids: number[]
  roles?: string[]
}

export type RBACUserInput = {
  username: string
  display_name: string
  email?: string
  password?: string
  enabled: boolean
  role_ids: number[]
}

export type NavMenuItem = {
  to: string
  label: string
  icon: string
  end?: boolean
  badge_key?: string
  permission: string
}

export type NavMenuGroup = {
  label: string
  items: NavMenuItem[]
}

export type NavMenuResult = {
  username?: string
  groups: NavMenuGroup[]
}

export type ScenarioSummary = {
  id: string
  label: string
  description?: string
  active: boolean
}

export type ScenariosResult = {
  active: string
  scenarios: ScenarioSummary[]
}

export type AuthUser = {
  username: string
  display_name: string
}

export type AuthConfigView = {
  type: 'none' | 'basic' | 'oauth'
  authenticated: boolean
  user?: AuthUser
  oauth_login_url?: string
}
