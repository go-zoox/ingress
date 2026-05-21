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
  wafEvents: () => request<WAFEvent[]>('/waf/events'),
  tlsCerts: () => request<TLSCert[]>('/tls/certs'),
  getConfig: () => request<{ path: string; content: string }>('/config'),
  putConfig: (content: string) =>
    request<{ hash: string }>('/config', {
      method: 'PUT',
      body: JSON.stringify({ content }),
    }),
  validateConfig: (content: string) =>
    request<{ valid: boolean }>('/config/validate', {
      method: 'POST',
      body: JSON.stringify({ content }),
    }),
  reload: () => request<{ ok: boolean }>('/reload', { method: 'POST' }),
  overviewMetrics: (window = '15m') =>
    request<OverviewMetrics>(`/metrics/overview?window=${encodeURIComponent(window)}`),
  logs: (params: { log?: 'access' | 'error'; q?: string; host?: string; status?: string }) => {
    const q = new URLSearchParams()
    if (params.log) q.set('log', params.log)
    if (params.q) q.set('q', params.q)
    if (params.host) q.set('host', params.host)
    if (params.status) q.set('status', params.status)
    const qs = q.toString()
    return request<{ lines: string[]; count: number }>(`/logs${qs ? `?${qs}` : ''}`)
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
  status: string
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
