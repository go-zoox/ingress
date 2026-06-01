import { arr, num, obj, str } from './ingressModuleForms'
import {
  backendToForm,
  emptyBackendForm,
  formToBackend,
  rateLimitFromDoc,
  patchGlobalRateLimit,
  patchGlobalSecurity,
  securityFromDoc,
  type BackendForm,
  type RateLimitFormSlice,
  type SecurityFormSlice,
} from './configEntities'
import {
  globalMaintenanceFromDoc,
  patchGlobalMaintenance,
  type GlobalMaintenanceForm,
  validateGlobalMaintenanceForm,
} from './maintenance'

export type ScenarioOverlaySections = {
  cache: boolean
  rate_limit: boolean
  waf: boolean
  maintenance: boolean
  security: boolean
  rules: boolean
}

export type ScenarioCacheOverlayForm = {
  ttl: number
  host: string
  port: number
  prefix: string
  username: string
  password: string
  db: number
}

export type ScenarioRulePatchForm = {
  host: string
  backend: BackendForm
}

export type ScenarioItemForm = {
  id: string
  label: string
  description: string
  sections: ScenarioOverlaySections
  cache: ScenarioCacheOverlayForm
  rate_limit: RateLimitFormSlice
  rule_patches: ScenarioRulePatchForm[]
  waf: Record<string, unknown>
  maintenance: Record<string, unknown>
  security: Record<string, unknown>
}

export type ScenariosFormState = {
  active: string
  items: ScenarioItemForm[]
}

export const DEFAULT_SCENARIO_ID = 'default'
export const DEFAULT_SCENARIO_LABEL = '默认'
export const DEFAULT_SCENARIO_DESCRIPTION = '根配置，不应用 overlay'

export function isDefaultScenario(id: string): boolean {
  const t = id.trim()
  return t === '' || t === DEFAULT_SCENARIO_ID
}

export function effectiveScenarioActive(active: string): string {
  return isDefaultScenario(active) ? DEFAULT_SCENARIO_ID : active.trim()
}

export function emptyScenarioCacheOverlay(): ScenarioCacheOverlayForm {
  return {
    ttl: 300,
    host: '',
    port: 6379,
    prefix: '',
    username: '',
    password: '',
    db: 0,
  }
}

export function emptyScenarioOverlaySections(): ScenarioOverlaySections {
  return {
    cache: false,
    rate_limit: false,
    waf: false,
    maintenance: false,
    security: false,
    rules: false,
  }
}

export function emptyScenarioItem(id = ''): ScenarioItemForm {
  return {
    id,
    label: '',
    description: '',
    sections: emptyScenarioOverlaySections(),
    cache: emptyScenarioCacheOverlay(),
    rate_limit: {
      rate_limit_enabled: undefined,
      rate_limit_requests: 0,
      rate_limit_period: 60,
      rate_limit_key: 'ip',
      rate_limit_header: '',
      rate_limit_trust_proxy: false,
      rate_limit_xff_index: 0,
    },
    rule_patches: [],
    waf: {},
    maintenance: {},
    security: {},
  }
}

export function emptyScenariosFormState(): ScenariosFormState {
  return { active: '', items: [] }
}

function cacheOverlayFrom(raw: Record<string, unknown>): ScenarioCacheOverlayForm {
  return {
    ttl: num(raw.ttl, 300),
    host: str(raw.host),
    port: num(raw.port, 6379),
    prefix: str(raw.prefix),
    username: str(raw.username),
    password: str(raw.password),
    db: num(raw.db, 0),
  }
}

function buildCacheOverlay(form: ScenarioCacheOverlayForm): Record<string, unknown> {
  const out: Record<string, unknown> = {}
  if (form.ttl > 0) out.ttl = form.ttl
  if (form.host.trim()) out.host = form.host.trim()
  if (form.port > 0 && form.port !== 6379) out.port = form.port
  else if (form.port > 0 && form.host.trim()) out.port = form.port
  if (form.prefix.trim()) out.prefix = form.prefix.trim()
  if (form.username.trim()) out.username = form.username.trim()
  if (form.password) out.password = form.password
  if (form.db > 0) out.db = form.db
  return out
}

function overlaySectionsFrom(overlay: Record<string, unknown>): ScenarioOverlaySections {
  return {
    cache: overlay.cache !== undefined,
    rate_limit: overlay.rate_limit !== undefined,
    waf: overlay.waf !== undefined,
    maintenance: overlay.maintenance !== undefined,
    security: overlay.security !== undefined,
    rules: overlay.rules !== undefined,
  }
}

function rulePatchFromOverlay(row: Record<string, unknown>): ScenarioRulePatchForm {
  return {
    host: str(row.host),
    backend: backendToForm(obj(row.backend)),
  }
}

function rulePatchToOverlay(form: ScenarioRulePatchForm): Record<string, unknown> | null {
  const host = form.host.trim()
  if (!host) return null
  const backend = formToBackend(form.backend, {})
  const patch: Record<string, unknown> = { host }
  if (Object.keys(backend).length > 0) patch.backend = backend
  return patch
}

export function scenarioItemFromYAML(item: Record<string, unknown>): ScenarioItemForm {
  const overlay = obj(item.overlay)
  const sections = overlaySectionsFrom(overlay)
  const form = emptyScenarioItem(str(item.id))
  form.id = str(item.id)
  form.label = str(item.label)
  form.description = str(item.description)
  form.sections = sections
  if (sections.cache) form.cache = cacheOverlayFrom(obj(overlay.cache))
  if (sections.rate_limit) {
    form.rate_limit = rateLimitFromDoc({ rate_limit: overlay.rate_limit })
  }
  if (sections.waf) form.waf = { ...obj(overlay.waf) }
  if (sections.maintenance) form.maintenance = { ...obj(overlay.maintenance) }
  if (sections.security) form.security = { ...obj(overlay.security) }
  if (sections.rules) {
    form.rule_patches = arr<Record<string, unknown>>(overlay.rules).map(rulePatchFromOverlay)
  }
  return form
}

export function buildScenarioItemOverlay(form: ScenarioItemForm): Record<string, unknown> {
  const overlay: Record<string, unknown> = {}
  if (form.sections.cache) {
    const cache = buildCacheOverlay(form.cache)
    if (Object.keys(cache).length > 0) overlay.cache = cache
    else overlay.cache = {}
  }
  if (form.sections.rate_limit) {
    const rl = patchRateLimitOverlay(form.rate_limit)
    if (Object.keys(rl).length > 0) overlay.rate_limit = rl
    else overlay.rate_limit = { enabled: false }
  }
  if (form.sections.waf && Object.keys(form.waf).length > 0) overlay.waf = { ...form.waf }
  if (form.sections.maintenance && Object.keys(form.maintenance).length > 0) {
    overlay.maintenance = { ...form.maintenance }
  }
  if (form.sections.security && Object.keys(form.security).length > 0) {
    overlay.security = { ...form.security }
  }
  if (form.sections.rules) {
    const rules = form.rule_patches
      .map(rulePatchToOverlay)
      .filter((r): r is Record<string, unknown> => r != null)
    overlay.rules = rules
  }
  return overlay
}

export function scenarioItemToYAML(form: ScenarioItemForm): Record<string, unknown> {
  const row: Record<string, unknown> = {
    id: form.id.trim(),
  }
  if (form.label.trim()) row.label = form.label.trim()
  if (form.description.trim()) row.description = form.description.trim()
  const overlay = buildScenarioItemOverlay(form)
  if (Object.keys(overlay).length > 0) row.overlay = overlay
  return row
}

export function scenariosFromModuleDoc(doc: Record<string, unknown>): ScenariosFormState {
  const block = obj(doc.scenarios)
  const items = arr<Record<string, unknown>>(block.items).map(scenarioItemFromYAML)
  return {
    active: effectiveScenarioActive(str(block.active)),
    items,
  }
}

export function patchModuleDocScenarios(
  doc: Record<string, unknown>,
  state: ScenariosFormState,
): Record<string, unknown> {
  const next = { ...doc }
  const scenarios: Record<string, unknown> = {
    active: effectiveScenarioActive(state.active),
    items: state.items.map(scenarioItemToYAML),
  }
  next.scenarios = scenarios
  return next
}

export function overlaySummaryKeys(form: ScenarioItemForm): string[] {
  const keys: string[] = []
  if (form.sections.cache) keys.push('cache')
  if (form.sections.rate_limit) keys.push('rate_limit')
  if (form.sections.waf) keys.push('waf')
  if (form.sections.maintenance) keys.push('maintenance')
  if (form.sections.security) keys.push('security')
  if (form.sections.rules) keys.push('rules')
  return keys
}

export function newScenarioId(existing: string[]): string {
  const base = 'scenario'
  const reserved = new Set([...existing, DEFAULT_SCENARIO_ID])
  let n = existing.length + 1
  let id = `${base}-${n}`
  while (reserved.has(id)) {
    n += 1
    id = `${base}-${n}`
  }
  return id
}

/** Derive a stable scenario id from label; falls back to newScenarioId when label is not slug-able. */
export function scenarioIdFromLabel(label: string, existing: string[]): string {
  const slug = label
    .trim()
    .toLowerCase()
    .replace(/[\s_]+/g, '-')
    .replace(/[^a-z0-9-]/g, '')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '')
  if (slug.length >= 2 && slug !== DEFAULT_SCENARIO_ID && !existing.includes(slug)) return slug
  return newScenarioId(existing)
}

export function maintenanceFormFromOverlay(raw: Record<string, unknown>): GlobalMaintenanceForm {
  return globalMaintenanceFromDoc({ maintenance: raw })
}

export function securityFormFromOverlay(raw: Record<string, unknown>): SecurityFormSlice {
  return securityFromDoc({ security: raw })
}

export function patchRateLimitOverlay(form: RateLimitFormSlice): Record<string, unknown> {
  const patched = patchGlobalRateLimit({}, form)
  return obj(patched.rate_limit)
}

export function patchMaintenanceOverlay(form: GlobalMaintenanceForm): Record<string, unknown> {
  const patched = patchGlobalMaintenance({}, form)
  return obj(patched.maintenance)
}

export function patchSecurityOverlay(form: SecurityFormSlice): Record<string, unknown> {
  const patched = patchGlobalSecurity({}, form)
  return obj(patched.security)
}

export function emptyScenarioRulePatch(host = ''): ScenarioRulePatchForm {
  const backend = emptyBackendForm()
  backend.cache_enabled = false
  return { host, backend }
}

export function scenarioListLabels(state: ScenariosFormState): Array<{
  id: string
  label: string
  description: string
  active: boolean
}> {
  return state.items.map((item) => ({
    id: item.id,
    label: item.label.trim() || item.id,
    description: item.description,
    active: item.id === state.active,
  }))
}

/** True when overlay has any enabled section with meaningful content. */
export function scenarioItemConfigured(form: ScenarioItemForm): boolean {
  return overlaySummaryKeys(form).length > 0 || form.label.trim() !== '' || form.description.trim() !== ''
}

export function validateScenarioMaintenanceOverlay(item: ScenarioItemForm): string | null {
  if (!item.sections.maintenance) return null
  return validateGlobalMaintenanceForm(maintenanceFormFromOverlay(item.maintenance))
}

export function validateScenariosFormState(state: ScenariosFormState): string | null {
  const ids = new Set<string>()
  for (const item of state.items) {
    const maintErr = validateScenarioMaintenanceOverlay(item)
    if (maintErr) {
      const label = item.label.trim() || item.id.trim() || '场景'
      return `${label}：${maintErr}`
    }
    const id = item.id.trim()
    if (!id) return '每个场景必须填写 ID'
    if (id === DEFAULT_SCENARIO_ID) {
      return `场景 ID「${DEFAULT_SCENARIO_ID}」为系统保留，表示根配置`
    }
    if (ids.has(id)) return `场景 ID 重复：${id}`
    ids.add(id)
  }
  if (state.items.length > 0) {
    const active = effectiveScenarioActive(state.active)
    if (!isDefaultScenario(active) && !ids.has(active)) {
      return `当前场景 ${active} 不在列表中`
    }
  }
  return null
}
