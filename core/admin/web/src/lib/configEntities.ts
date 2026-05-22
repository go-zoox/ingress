import { arr, obj, str, num, bool } from './ingressModuleForms'

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
  cache_key_headers: string
  cache_methods: string
}

export type PathForm = {
  path: string
} & BackendForm

export type RuleForm = {
  host: string
  host_type: string
  paths: PathForm[]
} & BackendForm

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
> {
  const c = obj(backend.cache)
  const keyHeaders = arr<string>(c.key_headers)
  const methods = arr<string>(c.methods)
  return {
    cache_enabled: bool(c.enabled),
    cache_ttl: num(c.ttl, 300),
    cache_max_body_bytes: num(c.max_body_bytes, 2097152),
    cache_key_hash: str(c.key_hash, 'md5') || 'md5',
    cache_skip_vary: bool(c.skip_vary),
    cache_skip_when_set_cookie: c.skip_when_set_cookie === undefined ? true : bool(c.skip_when_set_cookie, true),
    cache_ignore_response_private: bool(c.ignore_response_private),
    cache_honor_pragma_no_cache: c.honor_pragma_no_cache === undefined ? true : bool(c.honor_pragma_no_cache, true),
    cache_key_headers: keyHeaders.join(', '),
    cache_methods: methods.length ? methods.join(', ') : '',
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
    redirect_permanent: true,
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
    cache_key_headers: '',
    cache_methods: '',
  }
}

export function emptyPathForm(): PathForm {
  return {
    path: '/',
    ...emptyBackendForm(),
  }
}

export function pathToForm(row: Record<string, unknown>): PathForm {
  return {
    path: str(row.path),
    ...backendToForm(obj(row.backend)),
  }
}

export function ruleToForm(rule: Record<string, unknown>): RuleForm {
  return {
    host: str(rule.host),
    host_type: str(rule.host_type, 'auto'),
    paths: arr<Record<string, unknown>>(rule.paths).map(pathToForm),
    ...backendToForm(obj(rule.backend)),
  }
}

export function emptyRuleForm(): RuleForm {
  return {
    host: 'app.example.com',
    host_type: 'exact',
    paths: [],
    ...emptyBackendForm(),
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

  const keyHeaders = form.cache_key_headers.split(',').map((s) => s.trim()).filter(Boolean)
  if (keyHeaders.length) cache.key_headers = keyHeaders
  else delete cache.key_headers

  const methods = form.cache_methods.split(',').map((s) => s.trim()).filter(Boolean)
  if (methods.length) cache.methods = methods
  else delete cache.methods

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

export function formToBackend(form: BackendForm, original?: Record<string, unknown>): Record<string, unknown> {
  const orig = original ? { ...original } : {}
  const core = buildBackendCore(form)
  const next: Record<string, unknown> = { ...orig }

  if (core.type) next.type = core.type

  if (form.backend_type === 'service') {
    const svc = { ...obj(orig.service), ...obj(core.service) }
    if (!form.service_strip_prefix) delete svc.strip_prefix
    if (form.service_mode !== 'internal' && form.service_mode !== 'external') delete svc.mode
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
  return next
}

export function formToRule(form: RuleForm, original?: Record<string, unknown>): Record<string, unknown> {
  const next: Record<string, unknown> = original ? { ...original } : {}
  next.host = form.host.trim()
  if (form.host_type && form.host_type !== 'auto') next.host_type = form.host_type
  else delete next.host_type
  next.backend = formToBackend(form, obj(original?.backend))
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
  if (bool(cache.enabled)) base += ` · cache ${num(cache.ttl, 300)}s`
  return base
}

export function pathSummary(row: Record<string, unknown>): string {
  return backendSummary(obj(row.backend))
}

export function ruleSummary(rule: Record<string, unknown>): string {
  return backendSummary(obj(rule.backend))
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
    redirect_permanent: bool(redirect.permanent, true),
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
