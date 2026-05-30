import { arr, obj, str, num, bool } from './ingressModuleForms'
import {
  maintenanceHostEntriesFromYAML,
  maintenanceHostEntriesToYAML,
  type MaintenanceHostFormEntry,
} from './maintenance'

export type SSLForm = {
  domain: string
  certificate: string
  certificate_key: string
}

export type RuleBackendType = 'service' | 'redirect' | 'handler'

export type HandlerType = 'static_response' | 'file_server' | 'templates' | 'script'
export type HandlerEngine = 'javascript' | 'go'

export const DEFAULT_HANDLER_SCRIPT: Record<HandlerEngine, string> = {
  javascript: `ctx.status = 200
ctx.type = "application/json"
ctx.body = JSON.stringify({ method: ctx.method, path: ctx.path })
ctx.setHeader("X-Handler-Engine", "javascript")
`,
  go: `ctx.SetHeader("Content-Type", "application/json")
ctx.SetHeader("X-Handler-Engine", "go")
ctx.String(200, ctx.Method+" "+ctx.Path)
`,
}

/** Placeholder hints (includes optional APIs). */
export const HANDLER_SCRIPT_PLACEHOLDER: Record<HandlerEngine, string> = {
  javascript: `// ctx.status / ctx.type / ctx.body — response aliases
// ctx.method / ctx.path / ctx.headers — request aliases
// ctx.setHeader(key, value)  ·  ctx.response.setHeader(...)
// await ctx.fetch(url) → { status, ok, text(), json() }
ctx.status = 200
ctx.type = "application/json"
ctx.body = JSON.stringify({ method: ctx.method, path: ctx.path })`,
  go: `// ctx is *zoox.Context (yaegi): SetHeader, String, JSON, Fetch, ...
// ctx.Method  ctx.Path  ctx.Request  ctx.Writer
// res, err := ctx.Fetch().Get(url, nil).Execute()
ctx.SetHeader("Content-Type", "application/json")
ctx.String(200, ctx.Method+" "+ctx.Path)`,
}

export type AuthFormType = '' | 'basic' | 'bearer' | 'oauth2' | 'jwt' | 'oidc'

export type RateLimitKey = 'global' | 'route' | 'ip' | 'header'

export type RateLimitFormSlice = {
  rate_limit_enabled: boolean | undefined
  rate_limit_requests: number
  rate_limit_period: number
  rate_limit_key: RateLimitKey | ''
  rate_limit_header: string
  rate_limit_trust_proxy: boolean
  rate_limit_xff_index: number
}

export type SecurityProfile = 'off' | 'strict' | 'api' | 'embeddable'

export type SecurityFormSlice = {
  security_profile: SecurityProfile | ''
  security_hsts: 'auto' | 'on' | 'off' | ''
  security_frame: 'inherit' | 'deny' | 'sameorigin' | 'off' | ''
  security_referrer_policy: string
  security_csp: string
  security_content_type_options: boolean
  security_cors_enabled: boolean
  security_cors_origins: string
  security_cors_methods: string
  security_cors_headers: string
  security_cors_credentials: boolean
  security_cors_max_age: number
}

/** Host / path level: when false, omit security from YAML and inherit upstream layer. */
export type SecurityLayerFormSlice = SecurityFormSlice & {
  security_override: boolean
}

export type AuthBasicUserForm = {
  username: string
  password: string
}

export type StringKVForm = {
  key: string
  value: string
}

export type ServiceRequestHostRewrite = 'default' | 'true' | 'false'

export type AuthOAuth2ConnectJWTForm = {
  secret: string
  algorithm: string
  expires_in: string
}

export type AuthOAuth2ConnectForm = {
  enabled: boolean
  jwt: AuthOAuth2ConnectJWTForm
}

export type AuthOAuth2Form = {
  provider: string
  client_id: string
  client_secret: string
  redirect_url: string
  scopes: string
  connect: AuthOAuth2ConnectForm
}

export type BackendForm = {
  backend_type: RuleBackendType
  service_name: string
  service_port: number
  service_protocol: string
  service_mode: string
  service_strip_prefix: boolean
  redirect_url: string
  redirect_permanent: boolean
  handler_type: HandlerType
  handler_status_code: number
  handler_content_type: string
  handler_body: string
  handler_root_dir: string
  handler_index_file: string
  handler_engine: HandlerEngine
  handler_script: string
  cache_enabled: boolean
  cache_ttl: number
  cache_max_body_bytes: number
  cache_key_hash: string
  cache_skip_vary: boolean
  cache_skip_when_set_cookie: boolean
  cache_ignore_response_private: boolean
  cache_honor_pragma_no_cache: boolean
  cache_key_headers: string[]
  cache_methods: string
  cache_default: 'cache' | 'bypass'
  cache_path_rules: CachePathRuleForm[]
  // auth fields
  auth_enabled: boolean | undefined  // undefined = not set (default by type), true = explicit enable, false = explicit disable
  auth_type: AuthFormType
  auth_basic_users: AuthBasicUserForm[]
  auth_bearer_tokens: string
  auth_oauth2_provider: string
  auth_oauth2_client_id: string
  auth_oauth2_client_secret: string
  auth_oauth2_redirect_url: string
  auth_oauth2_scopes: string
  auth_oauth2_connect_enabled: boolean
  auth_oauth2_connect_jwt_secret: string
  auth_oauth2_connect_jwt_algorithm: string
  auth_oauth2_connect_jwt_expires_in: string
  auth_jwt_secret: string
  auth_jwt_public_key: string
  auth_jwt_algorithm: string
  auth_jwt_issuer: string
  auth_jwt_audience: string
  auth_oidc_provider: string
  auth_oidc_client_id: string
  auth_oidc_client_secret: string
  auth_oidc_redirect_url: string
  auth_oidc_scopes: string
  auth_oidc_issuer: string
  auth_oidc_audience: string
  // healthcheck fields
  health_check_enable: boolean
  health_check_method: string
  health_check_path: string
  health_check_status: string
  health_check_ok: boolean
  // maintenance (host-level service only)
  maintenance_enabled: boolean
  maintenance_scope: 'all' | 'listed' | ''
  maintenance_host_entries: MaintenanceHostFormEntry[]
  maintenance_retry_after: number
  maintenance_title: string
  maintenance_subtitle: string
  maintenance_bypass_paths: string
  maintenance_bypass_allow_ips: string
  maintenance_bypass_header_name: string
  maintenance_bypass_header_value: string
  maintenance_response_header_name: string
  maintenance_response_header_value: string
  // service request / response
  service_request_host_rewrite: ServiceRequestHostRewrite
  service_request_path_rewrites: string[]
  service_request_headers: StringKVForm[]
  service_request_query: StringKVForm[]
  service_request_delay: number
  service_request_timeout: number
  service_response_headers: StringKVForm[]
}

export type PathForm = {
  path: string
} & BackendForm & SecurityLayerFormSlice

export type CachePathRuleForm = {
  match: string
  match_type: 'auto' | 'prefix' | 'exact' | 'regex' | ''
  action: 'cache' | 'bypass'
  ttl: number
  max_body_bytes: number
  /** Path-level HTTP methods (e.g. POST). Empty inherits backend.cache.methods. */
  methods: string[]
  /** Dot paths into request JSON for cache fingerprint when POST is selected. */
  key_json: string[]
  key_body_max_bytes: number
}

export function emptyCachePathRule(): CachePathRuleForm {
  return {
    match: '',
    match_type: 'auto',
    action: 'cache',
    ttl: 0,
    max_body_bytes: 0,
    methods: [],
    key_json: [],
    key_body_max_bytes: 0,
  }
}

function splitConfigList(s: string): string[] {
  return s.split(/[,\n]/).map((x) => x.trim()).filter(Boolean)
}

/** UI default when enabling backend.cache; not applied by the ingress runtime unless saved. */
export const DEFAULT_CACHE_KEY_HEADERS = [
  'Authorization',
  'Cookie',
  'Token',
  'X-API-Key',
  'Api-Key',
  'X-Tenant-ID',
] as const

/** Dedupe cache key header names case-insensitively; preserve first spelling. */
export function normalizeCacheKeyHeaders(headers: string[]): string[] {
  const seen = new Set<string>()
  const out: string[] = []
  for (const h of headers) {
    const trimmed = h.trim()
    if (!trimmed) continue
    const key = trimmed.toLowerCase()
    if (seen.has(key)) continue
    seen.add(key)
    out.push(trimmed)
  }
  return out.sort((a, b) => a.localeCompare(b, undefined, { sensitivity: 'base' }))
}

function cachePathRulesToForm(cache: Record<string, unknown>): CachePathRuleForm[] {
  return arr<Record<string, unknown>>(cache.paths).map((row) => ({
    match: str(row.match),
    match_type: (str(row.match_type, 'auto') || 'auto') as CachePathRuleForm['match_type'],
    action: str(row.action, 'cache') === 'bypass' ? 'bypass' : 'cache',
    ttl: num(row.ttl, 0),
    max_body_bytes: num(row.max_body_bytes, 0),
    methods: arr<string>(row.methods).map((m) => m.toUpperCase()),
    key_json: arr<string>(row.key_json),
    key_body_max_bytes: num(row.key_body_max_bytes, 0),
  }))
}

export type RuleForm = {
  host: string
  host_type: string
  paths: PathForm[]
} & BackendForm & RateLimitFormSlice & SecurityLayerFormSlice

export type FallbackForm = {
  backend_type: 'service' | 'redirect'
  service_name: string
  service_port: number
  service_protocol: string
  redirect_url: string
  redirect_permanent: boolean
}

export function sslFromRow(row: Record<string, unknown>): SSLForm {
  const cert = obj(row.cert)
  return {
    domain: str(row.domain),
    certificate: str(cert.certificate),
    certificate_key: str(cert.certificate_key),
  }
}

export function sslToRow(form: SSLForm): Record<string, unknown> {
  return {
    domain: form.domain.trim(),
    cert: {
      certificate: form.certificate.trim(),
      certificate_key: form.certificate_key.trim(),
    },
  }
}

export function emptySSLForm(): SSLForm {
  return {
    domain: '',
    certificate: './certs/example.com.pem',
    certificate_key: './certs/example.com.key.pem',
  }
}

function handlerLooksConfigured(handler: Record<string, unknown>): boolean {
  const h = obj(handler)
  const ht = str(h.type)
  if (ht && ht !== 'static_response') return true
  if (str(h.body) || str(h.script) || str(h.root_dir)) return true
  if (Object.keys(obj(h.headers)).length > 0) return true
  if (num(h.status_code, 200) !== 200) return true
  if (str(h.index_file) && str(h.index_file) !== 'index.html') return true
  return false
}

export function inferBackendType(backend: Record<string, unknown>): RuleBackendType {
  const explicit = str(backend.type)
  if (explicit === 'redirect' || explicit === 'handler' || explicit === 'service') {
    return explicit
  }
  if (str(obj(backend.redirect).url)) return 'redirect'
  if (handlerLooksConfigured(obj(backend.handler))) return 'handler'
  return 'service'
}

function handlerTypeFromYAML(handler: Record<string, unknown>): HandlerType {
  const ht = str(handler.type, 'static_response')
  if (ht === 'file_server' || ht === 'templates' || ht === 'script') return ht
  return 'static_response'
}

function handlerToForm(handler: Record<string, unknown>): Pick<
  BackendForm,
  | 'handler_type'
  | 'handler_status_code'
  | 'handler_content_type'
  | 'handler_body'
  | 'handler_root_dir'
  | 'handler_index_file'
  | 'handler_engine'
  | 'handler_script'
> {
  const h = obj(handler)
  const headers = obj(h.headers)
  const engine = str(h.engine, 'javascript')
  return {
    handler_type: handlerTypeFromYAML(h),
    handler_status_code: num(h.status_code, 200),
    handler_content_type: str(headers['Content-Type']),
    handler_body: str(h.body),
    handler_root_dir: str(h.root_dir),
    handler_index_file: str(h.index_file, 'index.html'),
    handler_engine: engine === 'go' ? 'go' : 'javascript',
    handler_script: str(h.script),
  }
}

function cacheToForm(backend: Record<string, unknown>): Pick<
  BackendForm,
  | 'cache_enabled'
  | 'cache_ttl'
  | 'cache_max_body_bytes'
  | 'cache_key_hash'
  | 'cache_skip_vary'
  | 'cache_skip_when_set_cookie'
  | 'cache_ignore_response_private'
  | 'cache_honor_pragma_no_cache'
  | 'cache_key_headers'
  | 'cache_methods'
  | 'cache_default'
  | 'cache_path_rules'
> {
  const c = obj(backend.cache)
  const keyHeaders = arr<string>(c.key_headers)
  const methods = arr<string>(c.methods)
  const defaultAction = str(c.default, 'cache')
  return {
    cache_enabled: bool(c.enabled),
    cache_ttl: num(c.ttl, 300),
    cache_max_body_bytes: num(c.max_body_bytes, 2097152),
    cache_key_hash: str(c.key_hash, 'md5') || 'md5',
    cache_skip_vary: bool(c.skip_vary),
    cache_skip_when_set_cookie: c.skip_when_set_cookie === undefined ? true : bool(c.skip_when_set_cookie, true),
    cache_ignore_response_private: bool(c.ignore_response_private),
    cache_honor_pragma_no_cache: c.honor_pragma_no_cache === undefined ? true : bool(c.honor_pragma_no_cache, true),
    cache_key_headers: keyHeaders,
    cache_methods: methods.length ? methods.join(', ') : '',
    cache_default: defaultAction === 'bypass' ? 'bypass' : 'cache',
    cache_path_rules: cachePathRulesToForm(c),
  }
}

function rateLimitToForm(rateLimit: Record<string, unknown>): RateLimitFormSlice {
  const rl = obj(rateLimit)
  let enabled: boolean | undefined
  if (rl.enabled !== undefined && rl.enabled !== null) {
    enabled = bool(rl.enabled)
  }
  const key = str(rl.key, 'ip') || 'ip'
  return {
    rate_limit_enabled: enabled,
    rate_limit_requests: num(rl.requests),
    rate_limit_period: num(rl.period),
    rate_limit_key: (key === 'global' || key === 'route' || key === 'header' ? key : 'ip') as RateLimitKey,
    rate_limit_header: str(rl.header),
    rate_limit_trust_proxy: bool(rl.trust_proxy),
    rate_limit_xff_index: num(rl.xff_index),
  }
}

function emptyRateLimitForm(): RateLimitFormSlice {
  return {
    rate_limit_enabled: undefined,
    rate_limit_requests: 0,
    rate_limit_period: 60,
    rate_limit_key: 'ip',
    rate_limit_header: '',
    rate_limit_trust_proxy: false,
    rate_limit_xff_index: 0,
  }
}

function buildRateLimit(form: RateLimitFormSlice): Record<string, unknown> | undefined {
  if (form.rate_limit_enabled === false) {
    return undefined
  }
  if (form.rate_limit_requests <= 0 && form.rate_limit_enabled !== true) {
    return undefined
  }

  const rl: Record<string, unknown> = {
    requests: form.rate_limit_requests,
    period: form.rate_limit_period,
  }
  const key = form.rate_limit_key || 'ip'
  if (key !== 'ip') rl.key = key
  if (key === 'header' && form.rate_limit_header.trim()) {
    rl.header = form.rate_limit_header.trim()
  }
  if (form.rate_limit_trust_proxy) rl.trust_proxy = true
  if (form.rate_limit_xff_index !== 0) rl.xff_index = form.rate_limit_xff_index
  if (form.rate_limit_enabled === true) rl.enabled = true
  return rl
}

function authToForm(service: Record<string, unknown>): Pick<BackendForm,
  'auth_enabled' | 'auth_type' | 'auth_basic_users' | 'auth_bearer_tokens' |
  'auth_oauth2_provider' | 'auth_oauth2_client_id' | 'auth_oauth2_client_secret' |
  'auth_oauth2_redirect_url' | 'auth_oauth2_scopes' |
  'auth_oauth2_connect_enabled' | 'auth_oauth2_connect_jwt_secret' |
  'auth_oauth2_connect_jwt_algorithm' | 'auth_oauth2_connect_jwt_expires_in' |
  'auth_jwt_secret' | 'auth_jwt_public_key' | 'auth_jwt_algorithm' |
  'auth_jwt_issuer' | 'auth_jwt_audience' |
  'auth_oidc_provider' | 'auth_oidc_client_id' | 'auth_oidc_client_secret' |
  'auth_oidc_redirect_url' | 'auth_oidc_scopes' | 'auth_oidc_issuer' | 'auth_oidc_audience'
> {
  const auth = obj(service.auth)
  const authType = str(auth.type) as AuthFormType

  // auth.enabled: undefined = not set (default by type), true = explicit enable, false = explicit disable
  let authEnabled: boolean | undefined
  if (auth.enabled !== undefined && auth.enabled !== null) {
    authEnabled = bool(auth.enabled)
  }

  // basic users
  const basicUsers = arr<Record<string, unknown>>(obj(auth.basic).users)
  const authBasicUsers = basicUsers.length > 0
    ? basicUsers.map(u => ({ username: str(u.username), password: str(u.password) }))
    : authType === 'basic' ? [{ username: '', password: '' }] : []

  // bearer tokens
  const bearerTokens = arr<string>(obj(auth.bearer).tokens)

  // oauth2
  const oauth2 = obj(auth.oauth2)
  const connect = obj(oauth2.connect)
  const jwtConnect = obj(connect.jwt)
  const jwtAuth = obj(auth.jwt)
  const oidc = obj(auth.oidc)

  const jwtSecret = str(jwtAuth.secret) || str(auth.secret)

  return {
    auth_enabled: authEnabled,
    auth_type: authType,
    auth_basic_users: authBasicUsers,
    auth_bearer_tokens: bearerTokens.join(', '),
    auth_oauth2_provider: str(oauth2.provider),
    auth_oauth2_client_id: str(oauth2.client_id),
    auth_oauth2_client_secret: str(oauth2.client_secret),
    auth_oauth2_redirect_url: str(oauth2.redirect_url),
    auth_oauth2_scopes: arr<string>(oauth2.scopes).join(', '),
    auth_oauth2_connect_enabled: bool(connect.enabled),
    auth_oauth2_connect_jwt_secret: str(jwtConnect.secret),
    auth_oauth2_connect_jwt_algorithm: str(jwtConnect.algorithm, 'hs256') || 'hs256',
    auth_oauth2_connect_jwt_expires_in: str(jwtConnect.expires_in, '5m') || '5m',
    auth_jwt_secret: jwtSecret,
    auth_jwt_public_key: str(jwtAuth.public_key),
    auth_jwt_algorithm: str(jwtAuth.algorithm, 'HS256') || 'HS256',
    auth_jwt_issuer: str(jwtAuth.issuer),
    auth_jwt_audience: str(jwtAuth.audience),
    auth_oidc_provider: str(oidc.provider),
    auth_oidc_client_id: str(oidc.client_id),
    auth_oidc_client_secret: str(oidc.client_secret),
    auth_oidc_redirect_url: str(oidc.redirect_url),
    auth_oidc_scopes: arr<string>(oidc.scopes).join(', '),
    auth_oidc_issuer: str(oidc.issuer),
    auth_oidc_audience: str(oidc.audience),
  }
}

function mapToKVForm(m: Record<string, unknown>): StringKVForm[] {
  const entries = Object.entries(m).map(([key, value]) => ({ key, value: str(value) }))
  return entries.length > 0 ? entries : [{ key: '', value: '' }]
}

function kvFormToMap(rows: StringKVForm[]): Record<string, string> | undefined {
  const out: Record<string, string> = {}
  for (const { key, value } of rows) {
    const k = key.trim()
    if (!k) continue
    out[k] = value
  }
  return Object.keys(out).length > 0 ? out : undefined
}

function requestToForm(service: Record<string, unknown>): Pick<BackendForm,
  'service_request_host_rewrite' | 'service_request_path_rewrites' |
  'service_request_headers' | 'service_request_query' |
  'service_request_delay' | 'service_request_timeout' | 'service_response_headers'
> {
  const req = obj(service.request)
  const host = obj(req.host)
  const path = obj(req.path)
  const resp = obj(service.response)

  let hostRewrite: ServiceRequestHostRewrite = 'default'
  if (host.rewrite === true) hostRewrite = 'true'
  else if (host.rewrite === false) hostRewrite = 'false'

  const rewrites = arr<string>(path.rewrites).filter(Boolean)

  return {
    service_request_host_rewrite: hostRewrite,
    service_request_path_rewrites: rewrites,
    service_request_headers: mapToKVForm(obj(req.headers)),
    service_request_query: mapToKVForm(obj(req.query)),
    service_request_delay: num(req.delay, 0),
    service_request_timeout: num(req.timeout, 0),
    service_response_headers: mapToKVForm(obj(resp.headers)),
  }
}

export function serviceRequestConfigured(form: BackendForm): boolean {
  if (form.service_request_host_rewrite !== 'default') return true
  if (form.service_request_path_rewrites.some((r) => r.trim())) return true
  if (form.service_request_headers.some((r) => r.key.trim())) return true
  if (form.service_request_query.some((r) => r.key.trim())) return true
  if (form.service_request_delay > 0 || form.service_request_timeout > 0) return true
  if (form.service_response_headers.some((r) => r.key.trim())) return true
  return false
}

function healthCheckToForm(service: Record<string, unknown>): Pick<BackendForm,
  'health_check_enable' | 'health_check_method' | 'health_check_path' |
  'health_check_status' | 'health_check_ok'
> {
  const hc = obj(service.healthcheck)
  const statusArr = arr<number>(hc.status)
  return {
    health_check_enable: bool(hc.enable),
    health_check_method: str(hc.method),
    health_check_path: str(hc.path),
    health_check_status: statusArr.length > 0 ? statusArr.join(',') : '',
    health_check_ok: bool(hc.ok),
  }
}

function maintenanceToForm(service: Record<string, unknown>): Pick<BackendForm,
  'maintenance_enabled' | 'maintenance_scope' | 'maintenance_host_entries' |
  'maintenance_retry_after' | 'maintenance_title' | 'maintenance_subtitle' |
  'maintenance_bypass_paths' | 'maintenance_bypass_allow_ips' |
  'maintenance_bypass_header_name' | 'maintenance_bypass_header_value' |
  'maintenance_response_header_name' | 'maintenance_response_header_value'
> {
  const m = obj(service.maintenance)
  const bypass = obj(m.bypass)
  const header = obj(bypass.header)
  const responseHeader = obj(m.response_header)
  const paths = arr<string>(bypass.paths)
  const allowIPs = arr<string>(bypass.allow_ips)
  const scopeRaw = str(m.scope, 'all').toLowerCase()
  const scope = scopeRaw === 'listed' ? 'listed' : 'all'
  return {
    maintenance_enabled: bool(m.enabled),
    maintenance_scope: scope,
    maintenance_host_entries: maintenanceHostEntriesFromYAML(m.hosts),
    maintenance_retry_after: num(m.retry_after, 0),
    maintenance_title: str(m.title),
    maintenance_subtitle: str(m.subtitle),
    maintenance_bypass_paths: paths.join(', '),
    maintenance_bypass_allow_ips: allowIPs.join(', '),
    maintenance_bypass_header_name: str(header.name),
    maintenance_bypass_header_value: str(header.value),
    maintenance_response_header_name: str(responseHeader.name),
    maintenance_response_header_value: str(responseHeader.value),
  }
}

export function backendToForm(backend: Record<string, unknown>): BackendForm {
  const backendType = inferBackendType(backend)
  const service = obj(backend.service)
  const redirect = obj(backend.redirect)
  return {
    backend_type: backendType,
    service_name: str(service.name),
    service_port: num(service.port, 8080),
    service_protocol: str(service.protocol, 'http') || 'http',
    service_mode: str(service.mode),
    service_strip_prefix: bool(service.strip_prefix),
    redirect_url: str(redirect.url),
    redirect_permanent: bool(redirect.permanent),
    ...handlerToForm(obj(backend.handler)),
    ...cacheToForm(backend),
    ...authToForm(service),
    ...healthCheckToForm(service),
    ...maintenanceToForm(service),
    ...requestToForm(service),
  }
}

export function emptyBackendForm(): BackendForm {
  return {
    backend_type: 'service',
    service_name: 'backend.internal',
    service_port: 8080,
    service_protocol: 'http',
    service_mode: '',
    service_strip_prefix: false,
    redirect_url: '',
    redirect_permanent: false,
    handler_type: 'static_response',
    handler_status_code: 200,
    handler_content_type: 'text/plain; charset=utf-8',
    handler_body: 'ok\n',
    handler_root_dir: './static',
    handler_index_file: 'index.html',
    handler_engine: 'javascript',
    handler_script: DEFAULT_HANDLER_SCRIPT.javascript,
    cache_enabled: false,
    cache_ttl: 300,
    cache_max_body_bytes: 2097152,
    cache_key_hash: 'md5',
    cache_skip_vary: false,
    cache_skip_when_set_cookie: true,
    cache_ignore_response_private: false,
    cache_honor_pragma_no_cache: true,
    cache_key_headers: [],
    cache_methods: '',
    cache_default: 'cache',
    cache_path_rules: [],
    auth_enabled: undefined,
    auth_type: '' as AuthFormType,
    auth_basic_users: [],
    auth_bearer_tokens: '',
    auth_oauth2_provider: '',
    auth_oauth2_client_id: '',
    auth_oauth2_client_secret: '',
    auth_oauth2_redirect_url: '',
    auth_oauth2_scopes: '',
    auth_oauth2_connect_enabled: false,
    auth_oauth2_connect_jwt_secret: '',
    auth_oauth2_connect_jwt_algorithm: 'hs256',
    auth_oauth2_connect_jwt_expires_in: '5m',
    auth_jwt_secret: '',
    auth_jwt_public_key: '',
    auth_jwt_algorithm: 'HS256',
    auth_jwt_issuer: '',
    auth_jwt_audience: '',
    auth_oidc_provider: '',
    auth_oidc_client_id: '',
    auth_oidc_client_secret: '',
    auth_oidc_redirect_url: '',
    auth_oidc_scopes: '',
    auth_oidc_issuer: '',
    auth_oidc_audience: '',
    health_check_enable: false,
    health_check_method: '',
    health_check_path: '',
    health_check_status: '',
    health_check_ok: false,
    maintenance_enabled: false,
    maintenance_scope: 'all',
    maintenance_host_entries: [],
    maintenance_retry_after: 0,
    maintenance_title: '',
    maintenance_subtitle: '',
    maintenance_bypass_paths: '',
    maintenance_bypass_allow_ips: '',
    maintenance_bypass_header_name: '',
    maintenance_bypass_header_value: '',
    maintenance_response_header_name: '',
    maintenance_response_header_value: '',
    service_request_host_rewrite: 'default',
    service_request_path_rewrites: [],
    service_request_headers: [{ key: '', value: '' }],
    service_request_query: [{ key: '', value: '' }],
    service_request_delay: 0,
    service_request_timeout: 0,
    service_response_headers: [{ key: '', value: '' }],
  }
}

export function emptyPathForm(): PathForm {
  return {
    path: '/',
    ...emptyBackendForm(),
    ...emptySecurityLayerForm(),
  }
}

export function pathToForm(row: Record<string, unknown>): PathForm {
  return {
    path: str(row.path),
    ...backendToForm(obj(row.backend)),
    ...securityLayerFromDoc(obj(row.security)),
  }
}

export function ruleToForm(rule: Record<string, unknown>): RuleForm {
  return {
    host: str(rule.host),
    host_type: str(rule.host_type, 'auto'),
    paths: arr<Record<string, unknown>>(rule.paths).map(pathToForm),
    ...backendToForm(obj(rule.backend)),
    ...rateLimitToForm(obj(rule.rate_limit)),
    ...securityLayerFromDoc(obj(rule.security)),
  }
}

export function emptyRuleForm(): RuleForm {
  return {
    host: 'app.example.com',
    host_type: 'exact',
    paths: [],
    ...emptyBackendForm(),
    ...emptyRateLimitForm(),
    ...emptySecurityLayerForm(),
  }
}

function buildHandler(form: BackendForm): Record<string, unknown> {
  switch (form.handler_type) {
    case 'file_server': {
      const handler: Record<string, unknown> = {
        type: 'file_server',
        root_dir: form.handler_root_dir.trim(),
      }
      const indexFile = form.handler_index_file.trim()
      if (indexFile && indexFile !== 'index.html') handler.index_file = indexFile
      return handler
    }
    case 'templates':
      return {
        type: 'templates',
        root_dir: form.handler_root_dir.trim(),
      }
    case 'script': {
      const handler: Record<string, unknown> = {
        type: 'script',
        engine: form.handler_engine,
        script: form.handler_script,
      }
      if (form.handler_engine === 'javascript') {
        if (form.handler_status_code !== 200) handler.status_code = form.handler_status_code
        const ct = form.handler_content_type.trim()
        if (ct) handler.headers = { 'Content-Type': ct }
        if (form.handler_body) handler.body = form.handler_body
      }
      return handler
    }
    default: {
      const handler: Record<string, unknown> = { type: 'static_response' }
      if (form.handler_status_code !== 200) handler.status_code = form.handler_status_code
      const ct = form.handler_content_type.trim()
      if (ct) handler.headers = { 'Content-Type': ct }
      if (form.handler_body) handler.body = form.handler_body
      return handler
    }
  }
}

function buildCache(form: BackendForm, original?: Record<string, unknown>): Record<string, unknown> {
  const orig = obj(original?.cache)
  const cache: Record<string, unknown> = {
    ...orig,
    enabled: true,
    ttl: form.cache_ttl,
  }
  if (form.cache_max_body_bytes > 0) cache.max_body_bytes = form.cache_max_body_bytes
  if (form.cache_key_hash && form.cache_key_hash !== 'md5') cache.key_hash = form.cache_key_hash
  else if (form.cache_key_hash === 'md5') delete cache.key_hash

  if (form.cache_skip_vary) cache.skip_vary = true
  else delete cache.skip_vary

  if (form.cache_ignore_response_private) cache.ignore_response_private = true
  else delete cache.ignore_response_private

  if (form.cache_skip_when_set_cookie) delete cache.skip_when_set_cookie
  else cache.skip_when_set_cookie = false

  if (form.cache_honor_pragma_no_cache) delete cache.honor_pragma_no_cache
  else cache.honor_pragma_no_cache = false

  const keyHeaders = normalizeCacheKeyHeaders(form.cache_key_headers)
  if (keyHeaders.length) cache.key_headers = keyHeaders
  else delete cache.key_headers

  const methods = splitConfigList(form.cache_methods)
    .map((m) => m.toUpperCase())
    .filter((m) => m !== 'POST')
  if (methods.length) cache.methods = methods
  else delete cache.methods

  const pathRules = form.cache_path_rules
    .map((r) => ({
      match: r.match.trim(),
      match_type: r.match_type,
      action: r.action,
      ttl: r.ttl,
      max_body_bytes: r.max_body_bytes,
      methods: r.methods.map((m) => m.toUpperCase()).filter(Boolean),
      key_json: r.key_json.map((p) => p.trim()).filter(Boolean),
      key_body_max_bytes: r.key_body_max_bytes,
    }))
    .filter((r) => r.match !== '')
  if (pathRules.length) {
    cache.paths = pathRules.map((r) => {
      const row: Record<string, unknown> = {
        match: r.match,
        action: r.action,
      }
      if (r.match_type && r.match_type !== 'auto') row.match_type = r.match_type
      if (r.ttl > 0) row.ttl = r.ttl
      if (r.max_body_bytes > 0) row.max_body_bytes = r.max_body_bytes
      if (r.methods.length) row.methods = r.methods
      if (r.key_json.length) row.key_json = r.key_json
      if (r.key_body_max_bytes > 0) row.key_body_max_bytes = r.key_body_max_bytes
      return row
    })
    if (form.cache_default === 'bypass') cache.default = 'bypass'
    else delete cache.default
  } else {
    delete cache.paths
    delete cache.default
  }

  return cache
}

function buildBackendCore(form: BackendForm): Record<string, unknown> {
  if (form.backend_type === 'redirect') {
    const redirect: Record<string, unknown> = { url: form.redirect_url.trim() }
    if (form.redirect_permanent) redirect.permanent = true
    return { type: 'redirect', redirect }
  }
  if (form.backend_type === 'handler') {
    return { type: 'handler', handler: buildHandler(form) }
  }
  const service: Record<string, unknown> = {
    name: form.service_name.trim(),
    port: form.service_port,
  }
  if (form.service_protocol && form.service_protocol !== 'http') {
    service.protocol = form.service_protocol
  }
  if (form.service_mode === 'internal' || form.service_mode === 'external') {
    service.mode = form.service_mode
  }
  if (form.service_strip_prefix) service.strip_prefix = true
  return { type: 'service', service }
}

function buildAuth(form: BackendForm): Record<string, unknown> | undefined {
  if (!form.auth_type) return undefined

  const auth: Record<string, unknown> = { type: form.auth_type }

  // Only write enabled when explicitly set (false = disabled, true = enabled)
  // When undefined, omit the field so the core engine defaults to "enabled when type is set"
  if (form.auth_enabled === false) auth.enabled = false
  else if (form.auth_enabled === true) auth.enabled = true

  if (form.auth_type === 'basic') {
    const users = form.auth_basic_users
      .filter(u => u.username.trim())
      .map(u => ({ username: u.username.trim(), password: u.password }))
    if (users.length > 0) auth.basic = { users }
  }

  if (form.auth_type === 'bearer') {
    const tokens = form.auth_bearer_tokens.split(',').map(s => s.trim()).filter(Boolean)
    if (tokens.length > 0) auth.bearer = { tokens }
  }

  if (form.auth_type === 'oauth2') {
    const oauth2: Record<string, unknown> = {
      provider: form.auth_oauth2_provider.trim(),
      client_id: form.auth_oauth2_client_id.trim(),
      client_secret: form.auth_oauth2_client_secret.trim(),
    }
    if (form.auth_oauth2_redirect_url.trim()) {
      oauth2.redirect_url = form.auth_oauth2_redirect_url.trim()
    }
    const scopes = form.auth_oauth2_scopes.split(',').map(s => s.trim()).filter(Boolean)
    if (scopes.length > 0) oauth2.scopes = scopes

    if (form.auth_oauth2_connect_enabled) {
      const connect: Record<string, unknown> = { enabled: true }
      const jwtObj: Record<string, unknown> = {
        secret: form.auth_oauth2_connect_jwt_secret.trim(),
      }
      if (form.auth_oauth2_connect_jwt_algorithm && form.auth_oauth2_connect_jwt_algorithm !== 'hs256') {
        jwtObj.algorithm = form.auth_oauth2_connect_jwt_algorithm
      }
      if (form.auth_oauth2_connect_jwt_expires_in && form.auth_oauth2_connect_jwt_expires_in !== '5m') {
        jwtObj.expires_in = form.auth_oauth2_connect_jwt_expires_in
      }
      connect.jwt = jwtObj
      oauth2.connect = connect
    }
    auth.oauth2 = oauth2
  }

  if (form.auth_type === 'jwt') {
    const jwtObj: Record<string, unknown> = {}
    const secret = form.auth_jwt_secret.trim()
    if (secret) {
      auth.secret = secret
      jwtObj.secret = secret
    }
    if (form.auth_jwt_public_key.trim()) jwtObj.public_key = form.auth_jwt_public_key.trim()
    const alg = form.auth_jwt_algorithm.trim()
    if (alg && alg !== 'HS256') jwtObj.algorithm = alg
    if (form.auth_jwt_issuer.trim()) jwtObj.issuer = form.auth_jwt_issuer.trim()
    if (form.auth_jwt_audience.trim()) jwtObj.audience = form.auth_jwt_audience.trim()
    auth.jwt = jwtObj
  }

  if (form.auth_type === 'oidc') {
    const oidc: Record<string, unknown> = {}
    if (form.auth_oidc_provider.trim()) oidc.provider = form.auth_oidc_provider.trim()
    if (form.auth_oidc_client_id.trim()) oidc.client_id = form.auth_oidc_client_id.trim()
    if (form.auth_oidc_client_secret.trim()) oidc.client_secret = form.auth_oidc_client_secret.trim()
    if (form.auth_oidc_redirect_url.trim()) oidc.redirect_url = form.auth_oidc_redirect_url.trim()
    const scopes = form.auth_oidc_scopes.split(',').map(s => s.trim()).filter(Boolean)
    if (scopes.length > 0) oidc.scopes = scopes
    if (form.auth_oidc_issuer.trim()) oidc.issuer = form.auth_oidc_issuer.trim()
    if (form.auth_oidc_audience.trim()) oidc.audience = form.auth_oidc_audience.trim()
    auth.oidc = oidc
  }

  return auth
}

function buildServiceRequest(form: BackendForm): Record<string, unknown> | undefined {
  const request: Record<string, unknown> = {}

  if (form.service_request_host_rewrite === 'true') {
    request.host = { rewrite: true }
  } else if (form.service_request_host_rewrite === 'false') {
    request.host = { rewrite: false }
  }

  const rewrites = form.service_request_path_rewrites.map((r) => r.trim()).filter(Boolean)
  if (rewrites.length > 0) request.path = { rewrites }

  const headers = kvFormToMap(form.service_request_headers)
  if (headers) request.headers = headers

  const query = kvFormToMap(form.service_request_query)
  if (query) request.query = query

  if (form.service_request_delay > 0) request.delay = form.service_request_delay
  if (form.service_request_timeout > 0) request.timeout = form.service_request_timeout

  return Object.keys(request).length > 0 ? request : undefined
}

function buildServiceResponse(form: BackendForm): Record<string, unknown> | undefined {
  const headers = kvFormToMap(form.service_response_headers)
  if (!headers) return undefined
  return { headers }
}

function buildHealthCheck(form: BackendForm): Record<string, unknown> | undefined {
  if (!form.health_check_enable && !form.health_check_method && !form.health_check_path &&
      !form.health_check_status && !form.health_check_ok) {
    return undefined
  }

  if (!form.health_check_enable) return undefined

  const hc: Record<string, unknown> = { enable: true }

  if (form.health_check_method && form.health_check_method !== 'GET') {
    hc.method = form.health_check_method
  }
  if (form.health_check_path && form.health_check_path !== '/health') {
    hc.path = form.health_check_path
  }
  if (form.health_check_status) {
    const codes = form.health_check_status.split(',').map(s => s.trim()).filter(Boolean).map(Number).filter(n => !isNaN(n))
    const nonDefault = codes.length > 0 && !(codes.length === 1 && codes[0] === 200)
    if (nonDefault) hc.status = codes
  }
  if (form.health_check_ok) hc.ok = true

  return hc
}

function buildMaintenance(form: BackendForm): Record<string, unknown> | undefined {
  if (!form.maintenance_enabled &&
      form.maintenance_retry_after <= 0 &&
      !form.maintenance_title.trim() &&
      !form.maintenance_subtitle.trim() &&
      !form.maintenance_bypass_paths.trim() &&
      !form.maintenance_bypass_allow_ips.trim() &&
      !form.maintenance_bypass_header_name.trim() &&
      !form.maintenance_bypass_header_value.trim() &&
      !form.maintenance_response_header_name.trim() &&
      !form.maintenance_response_header_value.trim() &&
      !(form.maintenance_scope === 'listed' && form.maintenance_host_entries.length > 0)) {
    return undefined
  }

  const block: Record<string, unknown> = {}
  if (form.maintenance_enabled) block.enabled = true
  const scope = form.maintenance_scope === 'listed' ? 'listed' : 'all'
  if (form.maintenance_enabled && scope === 'listed') {
    block.scope = 'listed'
    const hosts = maintenanceHostEntriesToYAML(form.maintenance_host_entries)
    if (hosts.length) block.hosts = hosts
  }
  if (form.maintenance_retry_after > 0) block.retry_after = form.maintenance_retry_after
  if (form.maintenance_title.trim()) block.title = form.maintenance_title.trim()
  if (form.maintenance_subtitle.trim()) block.subtitle = form.maintenance_subtitle.trim()

  const bypass: Record<string, unknown> = {}
  const paths = form.maintenance_bypass_paths.split(',').map((s) => s.trim()).filter(Boolean)
  if (paths.length) bypass.paths = paths
  const allowIPs = form.maintenance_bypass_allow_ips.split(',').map((s) => s.trim()).filter(Boolean)
  if (allowIPs.length) bypass.allow_ips = allowIPs
  if (form.maintenance_bypass_header_name.trim() || form.maintenance_bypass_header_value.trim()) {
    bypass.header = {
      name: form.maintenance_bypass_header_name.trim(),
      value: form.maintenance_bypass_header_value,
    }
  }
  if (Object.keys(bypass).length) block.bypass = bypass
  if (form.maintenance_response_header_name.trim() || form.maintenance_response_header_value.trim()) {
    const responseHeader: Record<string, unknown> = {}
    if (form.maintenance_response_header_name.trim()) {
      responseHeader.name = form.maintenance_response_header_name.trim()
    }
    if (form.maintenance_response_header_value.trim()) {
      responseHeader.value = form.maintenance_response_header_value
    }
    block.response_header = responseHeader
  }
  return block
}

export function formToBackend(form: BackendForm, original?: Record<string, unknown>): Record<string, unknown> {
  const orig = original ? { ...original } : {}
  const core = buildBackendCore(form)
  const next: Record<string, unknown> = { ...orig }

  if (core.type) next.type = core.type

  if (form.backend_type === 'service') {
    const svc = { ...obj(orig.service), ...obj(core.service) }
    if (!form.service_strip_prefix) delete svc.strip_prefix
    if (form.service_mode !== 'internal' && form.service_mode !== 'external') delete svc.mode
    const authBlock = buildAuth(form)
    if (authBlock) svc.auth = authBlock
    else delete svc.auth
    const hcBlock = buildHealthCheck(form)
    if (hcBlock) svc.healthcheck = hcBlock
    else delete svc.healthcheck
    const reqBlock = buildServiceRequest(form)
    if (reqBlock) svc.request = reqBlock
    else delete svc.request
    const respBlock = buildServiceResponse(form)
    if (respBlock) svc.response = respBlock
    else delete svc.response
    const maintBlock = buildMaintenance(form)
    if (maintBlock) svc.maintenance = maintBlock
    else delete svc.maintenance
    next.service = svc
    delete next.handler
    delete next.redirect
  } else if (form.backend_type === 'handler') {
    next.handler = core.handler
    delete next.service
    delete next.redirect
  } else if (form.backend_type === 'redirect') {
    next.redirect = core.redirect
    delete next.service
    delete next.handler
  }

  if (form.cache_enabled) next.cache = buildCache(form, orig)
  else delete next.cache

  return next
}

export function formToPath(form: PathForm, original?: Record<string, unknown>): Record<string, unknown> {
  const next: Record<string, unknown> = original ? { ...original } : {}
  next.path = form.path.trim()
  next.backend = formToBackend(form, obj(original?.backend))
  const secBlock = buildSecurityLayer(form)
  if (secBlock) next.security = secBlock
  else delete next.security
  return next
}

export function formToRule(form: RuleForm, original?: Record<string, unknown>): Record<string, unknown> {
  const next: Record<string, unknown> = original ? { ...original } : {}
  next.host = form.host.trim()
  if (form.host_type && form.host_type !== 'auto') next.host_type = form.host_type
  else delete next.host_type
  next.backend = formToBackend(form, obj(original?.backend))
  const rlBlock = buildRateLimit(form)
  if (rlBlock) next.rate_limit = rlBlock
  else delete next.rate_limit
  const secBlock = buildSecurityLayer(form)
  if (secBlock) next.security = secBlock
  else delete next.security
  const origPaths = arr<Record<string, unknown>>(original?.paths)
  next.paths = form.paths.map((p, i) => formToPath(p, origPaths[i]))
  return next
}

export function pathsFromRule(rule: Record<string, unknown>): PathForm[] {
  return arr<Record<string, unknown>>(rule.paths).map(pathToForm)
}

export function applyPathsToRule(
  rule: Record<string, unknown>,
  paths: PathForm[],
): Record<string, unknown> {
  const origPaths = arr<Record<string, unknown>>(rule.paths)
  return {
    ...rule,
    paths: paths.map((p, i) => formToPath(p, origPaths[i])),
  }
}

export function handlerSummary(handler: Record<string, unknown>): string {
  const h = obj(handler)
  const ht = str(h.type, 'static_response')
  if (ht === 'static_response') {
    const body = str(h.body)
    if (body) return `static (${body.split('\n')[0].slice(0, 24)}${body.length > 24 ? '…' : ''})`
    return 'static_response'
  }
  if (ht === 'file_server') return `file_server (${str(h.root_dir) || '?'})`
  if (ht === 'templates') return `templates (${str(h.root_dir) || '?'})`
  if (ht === 'script') return `script/${str(h.engine, 'javascript')}`
  return ht
}

export function backendSummary(backend: Record<string, unknown>): string {
  const backendType = inferBackendType(backend)
  const service = obj(backend.service)
  const redirect = obj(backend.redirect)
  const cache = obj(backend.cache)
  let base: string
  if (backendType === 'handler') base = `handler → ${handlerSummary(obj(backend.handler))}`
  else if (service.name) {
    base = `service → ${str(service.name)}:${num(service.port, 80)}`
    if (str(service.mode)) base += ` (${str(service.mode)})`
    if (bool(service.strip_prefix)) base += ' strip'
  }
  else if (redirect.url) base = `redirect → ${str(redirect.url)}`
  else base = str(backend.type, 'service')
  if (bool(cache.enabled)) {
    base += ` · cache ${num(cache.ttl, 300)}s`
    const paths = arr<unknown>(cache.paths)
    if (paths.length) base += ` · ${paths.length} path rules`
  }
  const auth = obj(service.auth)
  const authType = str(auth.type)
  if (authType === 'basic') base += ' · auth:basic'
  else if (authType === 'bearer') base += ' · auth:bearer'
  else if (authType === 'oauth2') base += ` · auth:oauth2(${str(obj(auth.oauth2).provider)})`
  else if (authType === 'jwt') base += ' · auth:jwt'
  else if (authType === 'oidc') {
    const oidc = obj(auth.oidc)
    if (str(oidc.provider)) base += ` · auth:oidc(${str(oidc.provider)})`
    else if (str(oidc.issuer)) base += ` · auth:oidc(${str(oidc.issuer)})`
    else base += ' · auth:oidc'
  }
  if (authType && auth.enabled === false) base += ' (disabled)'
  const hc = obj(service.healthcheck)
  if (bool(hc.ok)) base += ' · HC: ✓(ok)'
  else if (bool(hc.enable)) base += ' · HC: ✓'
  const req = obj(service.request)
  const reqParts: string[] = []
  const host = obj(req.host)
  if (host.rewrite === true) reqParts.push('Host→')
  else if (host.rewrite === false) reqParts.push('Host✗')
  const rewrites = arr<string>(obj(req.path).rewrites)
  if (rewrites.length) reqParts.push(`${rewrites.length} rewrite`)
  if (Object.keys(obj(req.headers)).length) reqParts.push('req hdr')
  if (num(req.timeout) > 0) reqParts.push(`${num(req.timeout)}s`)
  if (reqParts.length) base += ` · ${reqParts.join(',')}`
  return base
}

export function pathSummary(row: Record<string, unknown>): string {
  let base = backendSummary(obj(row.backend))
  const sec = obj(row.security)
  const profile = str(sec.profile)
  if (profile) {
    base += profile === 'off' ? ' · sec:off' : ` · sec:${profile}`
  }
  return base
}

export function ruleSummary(rule: Record<string, unknown>): string {
  let base = backendSummary(obj(rule.backend))
  const rl = obj(rule.rate_limit)
  if (num(rl.requests) > 0 || rl.enabled === true) {
    base += ` · RL ${num(rl.requests)}/${num(rl.period, 1)}s`
    const key = str(rl.key, 'ip')
    if (key && key !== 'ip') base += `(${key})`
  }
  const sec = obj(rule.security)
  const profile = str(sec.profile)
  if (profile) {
    base += profile === 'off' ? ' · sec:off' : ` · sec:${profile}`
  }
  return base
}

export function rateLimitFromDoc(doc: Record<string, unknown>): RateLimitFormSlice {
  return rateLimitToForm(obj(doc.rate_limit))
}

export function patchGlobalRateLimit(doc: Record<string, unknown>, form: RateLimitFormSlice): Record<string, unknown> {
  const next = { ...doc }
  const rlBlock = buildRateLimit(form)
  if (rlBlock) next.rate_limit = rlBlock
  else delete next.rate_limit
  return next
}

function splitLines(s: string): string[] {
  return s
    .split(/[\n,]+/)
    .map((x) => x.trim())
    .filter(Boolean)
}

function splitCSV(s: string): string[] {
  return s
    .split(',')
    .map((x) => x.trim())
    .filter(Boolean)
}

export function emptySecurityForm(): SecurityFormSlice {
  return {
    security_profile: 'strict',
    security_hsts: 'auto',
    security_frame: 'inherit',
    security_referrer_policy: '',
    security_csp: '',
    security_content_type_options: true,
    security_cors_enabled: false,
    security_cors_origins: '',
    security_cors_methods: '',
    security_cors_headers: '',
    security_cors_credentials: false,
    security_cors_max_age: 0,
  }
}

export function emptySecurityLayerForm(): SecurityLayerFormSlice {
  return {
    security_override: false,
    ...emptySecurityForm(),
  }
}

function securityDocConfigured(sec: Record<string, unknown>): boolean {
  return Object.keys(obj(sec)).length > 0
}

function securityLayerFromDoc(sec: Record<string, unknown>): SecurityLayerFormSlice {
  const block = obj(sec)
  if (!securityDocConfigured(block)) {
    return emptySecurityLayerForm()
  }
  return {
    security_override: true,
    ...securityToForm(block),
  }
}

export function securityLayerConfigured(form: SecurityLayerFormSlice): boolean {
  return form.security_override
}

export function securityLayerBadge(form: SecurityLayerFormSlice): string | undefined {
  if (!form.security_override) return undefined
  if (form.security_profile === 'off') return '关'
  return form.security_profile || '自定义'
}

export function buildSecurityBlock(form: SecurityFormSlice): Record<string, unknown> | null {
  const profile = (form.security_profile || 'off').trim()
  if (profile === 'off') return { profile: 'off' }
  if (profile === '') return null

  const block: Record<string, unknown> = { profile }
  if (form.security_hsts && form.security_hsts !== 'auto') block.hsts = form.security_hsts
  if (form.security_frame && form.security_frame !== 'inherit') block.frame = form.security_frame
  if (form.security_referrer_policy.trim()) block.referrer_policy = form.security_referrer_policy.trim()
  if (form.security_csp.trim()) block.csp = form.security_csp.trim()
  if (form.security_content_type_options === false) block.content_type_options = false

  const origins = splitLines(form.security_cors_origins)
  const methods = splitCSV(form.security_cors_methods)
  const headers = splitCSV(form.security_cors_headers)
  const needsCors = profile === 'api' || form.security_cors_enabled || origins.length > 0
  if (needsCors) {
    const cors: Record<string, unknown> = {}
    if (!form.security_cors_enabled) cors.enabled = false
    if (origins.length) cors.origins = origins
    if (methods.length) cors.methods = methods
    if (headers.length) cors.headers = headers
    if (form.security_cors_credentials) cors.credentials = true
    if (form.security_cors_max_age > 0) cors.max_age = form.security_cors_max_age
    block.cors = cors
  }
  return block
}

export function buildSecurityLayer(form: SecurityLayerFormSlice): Record<string, unknown> | undefined {
  if (!form.security_override) return undefined
  const block = buildSecurityBlock(form)
  return block ?? undefined
}

export function securityToForm(sec: Record<string, unknown>): SecurityFormSlice {
  const cors = obj(sec.cors)
  const origins = arr<string>(cors.origins)
  const profile = (str(sec.profile, 'off') as SecurityProfile) || 'off'
  let corsEnabled = false
  if (cors.enabled === false) {
    corsEnabled = false
  } else if (cors.enabled === true || origins.length > 0) {
    corsEnabled = true
  } else if (profile === 'api') {
    corsEnabled = true
  }
  return {
    security_profile: profile,
    security_hsts: (str(sec.hsts, 'auto') as SecurityFormSlice['security_hsts']) || 'auto',
    security_frame: (str(sec.frame, 'inherit') as SecurityFormSlice['security_frame']) || 'inherit',
    security_referrer_policy: str(sec.referrer_policy),
    security_csp: str(sec.csp),
    security_content_type_options: sec.content_type_options !== false,
    security_cors_enabled: corsEnabled,
    security_cors_origins: origins.join('\n'),
    security_cors_methods: arr<string>(cors.methods).join(', '),
    security_cors_headers: arr<string>(cors.headers).join(', '),
    security_cors_credentials: bool(cors.credentials),
    security_cors_max_age: num(cors.max_age, 0),
  }
}

export function buildSecurity(form: SecurityFormSlice): Record<string, unknown> | null {
  const block = buildSecurityBlock(form)
  if (!block || block.profile === 'off') return null
  return block
}

export function securityFromDoc(doc: Record<string, unknown>): SecurityFormSlice {
  return securityToForm(obj(doc.security))
}

export function patchGlobalSecurity(doc: Record<string, unknown>, form: SecurityFormSlice): Record<string, unknown> {
  const next = { ...doc }
  const block = buildSecurity(form)
  if (block) next.security = block
  else delete next.security
  return next
}

export function fallbackToForm(doc: Record<string, unknown>): FallbackForm {
  const fallback = obj(doc.fallback)
  const backend = obj(fallback)
  const backendType = inferBackendType(backend) === 'redirect' ? 'redirect' : 'service'
  const service = obj(backend.service)
  const redirect = obj(backend.redirect)
  return {
    backend_type: backendType,
    service_name: str(service.name),
    service_port: num(service.port, 8080),
    service_protocol: str(service.protocol, 'http') || 'http',
    redirect_url: str(redirect.url),
    redirect_permanent: bool(redirect.permanent),
  }
}

export function formToFallback(form: FallbackForm): Record<string, unknown> {
  if (form.backend_type === 'redirect') {
    const redirect: Record<string, unknown> = { url: form.redirect_url.trim() }
    if (form.redirect_permanent) redirect.permanent = true
    return { fallback: { type: 'redirect', redirect } }
  }
  const service: Record<string, unknown> = {
    name: form.service_name.trim(),
    port: form.service_port,
  }
  if (form.service_protocol && form.service_protocol !== 'http') {
    service.protocol = form.service_protocol
  }
  return { fallback: { type: 'service', service } }
}

export function rulesFromDoc(doc: Record<string, unknown>): Record<string, unknown>[] {
  return arr<Record<string, unknown>>(doc.rules)
}

export function sslFromDoc(doc: Record<string, unknown>): Record<string, unknown>[] {
  return arr<Record<string, unknown>>(obj(doc.https).ssl)
}

function handlerSaveDisabled(form: BackendForm): boolean {
  if (form.handler_type === 'file_server' || form.handler_type === 'templates') {
    return !form.handler_root_dir.trim()
  }
  if (form.handler_type === 'script') {
    return !form.handler_script.trim()
  }
  return false
}

export function backendSaveDisabled(form: BackendForm): boolean {
  if (form.backend_type === 'handler') return handlerSaveDisabled(form)
  if (form.backend_type === 'redirect') return !form.redirect_url.trim()
  if (form.backend_type === 'service') return !form.service_name.trim()
  return false
}

export function pathSaveDisabled(form: PathForm): boolean {
  if (!form.path.trim()) return true
  return backendSaveDisabled(form)
}

export function ruleSaveDisabled(form: RuleForm): boolean {
  if (!form.host.trim()) return true
  return backendSaveDisabled(form)
}
