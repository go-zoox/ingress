import { arr, bool, obj, str } from './ingressModuleForms'
import { actionFromRow, type WAFAction, wafActionLabel } from './wafAction'

export type WAFRuleType = 'regex' | 'contains'

export type WAFRuleForm = {
  id: string
  name: string
  type: WAFRuleType
  pattern: string
  target_path: boolean
  target_query: boolean
  target_uri: boolean
  target_headers: boolean
  target_extra: string
  action: WAFAction
  enabled: boolean
}

export type { WAFAction } from './wafAction'
export { WAF_ACTION_OPTIONS, wafActionLabel } from './wafAction'

const STANDARD_TARGETS = new Set(['path', 'query', 'uri', 'headers'])

export function wafRulesFromDoc(doc: Record<string, unknown>): Record<string, unknown>[] {
  return arr<Record<string, unknown>>(obj(doc.waf).rules)
}

export function wafDenyList(doc: Record<string, unknown>): string[] {
  return arr<string>(obj(doc.waf).deny)
}

export function wafAllowList(doc: Record<string, unknown>): string[] {
  return arr<string>(obj(doc.waf).allow)
}

export function patchWafDeny(doc: Record<string, unknown>, items: string[]): Record<string, unknown> {
  const filtered = items
    .map((s) => s.trim())
    .filter(Boolean)
  const waf = { ...obj(doc.waf) }
  if (filtered.length > 0) {
    waf.deny = filtered
  } else {
    delete waf.deny
  }
  return { waf }
}

export function patchWafAllow(doc: Record<string, unknown>, items: string[]): Record<string, unknown> {
  const filtered = items
    .map((s) => s.trim())
    .filter(Boolean)
  const waf = { ...obj(doc.waf) }
  if (filtered.length > 0) {
    waf.allow = filtered
  } else {
    delete waf.allow
  }
  return { waf }
}

export function wafRuleToForm(row: Record<string, unknown>): WAFRuleForm {
  const targets = arr<string>(row.targets).map((t) => str(t).toLowerCase())
  const extra = targets.filter((t) => !STANDARD_TARGETS.has(t))
  return {
    id: str(row.id),
    name: str(row.name),
    type: str(row.type, 'regex') === 'contains' ? 'contains' : 'regex',
    pattern: str(row.pattern),
    target_path: targets.includes('path'),
    target_query: targets.includes('query'),
    target_uri: targets.includes('uri'),
    target_headers: targets.includes('headers'),
    target_extra: extra.join(', '),
    action: actionFromRow(row),
    enabled: row.enabled === undefined ? true : bool(row.enabled),
  }
}

export function emptyWAFRuleForm(): WAFRuleForm {
  return {
    id: 'custom-rule',
    name: '',
    type: 'contains',
    pattern: '/internal',
    target_path: true,
    target_query: false,
    target_uri: false,
    target_headers: false,
    target_extra: '',
    action: 'block',
    enabled: true,
  }
}

export function formTargets(form: WAFRuleForm): string[] {
  const out: string[] = []
  if (form.target_path) out.push('path')
  if (form.target_query) out.push('query')
  if (form.target_uri) out.push('uri')
  if (form.target_headers) out.push('headers')
  for (const part of form.target_extra.split(',')) {
    const t = part.trim()
    if (t) out.push(t)
  }
  return out
}

export function wafRuleToRow(form: WAFRuleForm): Record<string, unknown> {
  const row: Record<string, unknown> = {
    id: form.id.trim(),
    pattern: form.pattern,
    targets: formTargets(form),
  }
  if (form.name.trim()) row.name = form.name.trim()
  if (form.type !== 'regex') row.type = form.type
  if (form.action === 'audit' || form.action === 'pass') {
    row.action = form.action
  }
  if (!form.enabled) row.enabled = false
  return row
}

export function wafRuleSummary(row: Record<string, unknown>): string {
  const id = str(row.id) || '—'
  const typ = str(row.type, 'regex')
  const pattern = str(row.pattern)
  const targets = arr<string>(row.targets).join(', ') || '—'
  const short = pattern.length > 28 ? `${pattern.slice(0, 28)}…` : pattern
  const act = actionFromRow(row)
  const actLabel = act !== 'block' ? ` · ${wafActionLabel(act)}` : ''
  const off = row.enabled === false ? ' · 已禁用' : ''
  return `${id} · ${typ} · ${short} → [${targets}]${actLabel}${off}`
}

export function wafRuleSaveDisabled(form: WAFRuleForm): boolean {
  if (!form.id.trim() || !form.pattern.trim()) return true
  return formTargets(form).length === 0
}

export function patchWafRules(
  doc: Record<string, unknown>,
  rules: Record<string, unknown>[],
): Record<string, unknown> {
  const waf = { ...obj(doc.waf), rules }
  return { ...doc, waf }
}

export function patchCustomRuleEnabled(
  doc: Record<string, unknown>,
  index: number,
  enabled: boolean,
): Record<string, unknown> {
  const rules = wafRulesFromDoc(doc).map((row, i) => {
    if (i !== index) return row
    const next = { ...row }
    if (enabled) {
      delete next.enabled
    } else {
      next.enabled = false
    }
    return next
  })
  return patchWafRules(doc, rules)
}
