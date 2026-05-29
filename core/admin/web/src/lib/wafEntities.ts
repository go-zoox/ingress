import { arr, bool, obj, str } from './ingressModuleForms'
import {
  actionFromRow,
  type WAFAction,
  wafActionIsExplicit,
  wafActionLabel,
} from './wafAction'

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
  allow_hosts: string[]
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

export function wafAllowHostsList(doc: Record<string, unknown>): string[] {
  return arr<string>(obj(doc.waf).allow_hosts)
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

export function patchWafAllowHosts(doc: Record<string, unknown>, items: string[]): Record<string, unknown> {
  const filtered = items
    .map((s) => s.trim())
    .filter(Boolean)
  const waf = { ...obj(doc.waf) }
  if (filtered.length > 0) {
    waf.allow_hosts = filtered
  } else {
    delete waf.allow_hosts
  }
  return { waf }
}

export function wafRuleAllowHosts(doc: Record<string, unknown>, ruleID: string): string[] {
  const row = wafRulesFromDoc(doc).find((r) => str(r.id) === ruleID)
  return row ? arr<string>(row.allow_hosts) : []
}

export function patchBuiltinRuleAllowHosts(
  doc: Record<string, unknown>,
  ruleID: string,
  hosts: string[],
): Record<string, unknown> {
  const filtered = hosts.map((s) => s.trim()).filter(Boolean)
  const rules = wafRulesFromDoc(doc)
  const idx = rules.findIndex((r) => str(r.id) === ruleID)
  if (idx < 0 && filtered.length === 0) return doc

  if (filtered.length === 0) {
    if (idx < 0) return doc
    const row = { ...rules[idx] }
    delete row.allow_hosts
    const keys = Object.keys(row).filter((k) => k !== 'id')
    if (keys.length === 0) {
      return patchWafRules(doc, rules.filter((_, i) => i !== idx))
    }
    const next = [...rules]
    next[idx] = row
    return patchWafRules(doc, next)
  }

  const row = idx >= 0 ? { ...rules[idx] } : { id: ruleID }
  row.allow_hosts = filtered
  const next = [...rules]
  if (idx >= 0) next[idx] = row
  else next.push(row)
  return patchWafRules(doc, next)
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
    allow_hosts: arr<string>(row.allow_hosts),
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
    action: 'inherit',
    enabled: true,
    allow_hosts: [],
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
  if (wafActionIsExplicit(form.action)) {
    row.action = form.action
  }
  if (!form.enabled) row.enabled = false
  const hosts = form.allow_hosts.map((s) => s.trim()).filter(Boolean)
  if (hosts.length > 0) {
    row.allow_hosts = hosts
  } else {
    delete row.allow_hosts
  }
  return row
}

export function wafRuleSummary(row: Record<string, unknown>): string {
  const id = str(row.id) || '—'
  const typ = str(row.type, 'regex')
  const pattern = str(row.pattern)
  const targets = arr<string>(row.targets).join(', ') || '—'
  const short = pattern.length > 28 ? `${pattern.slice(0, 28)}…` : pattern
  const act = actionFromRow(row)
  const actLabel = ` · ${wafActionLabel(act)}`
  const off = row.enabled === false ? ' · 已禁用' : ''
  const hosts = arr<string>(row.allow_hosts)
  const hostHint = hosts.length > 0 ? ` · 域名白名单 ${hosts.length}` : ''
  return `${id} · ${typ} · ${short} → [${targets}]${actLabel}${hostHint}${off}`
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

export function wafCustomRuleName(row: Record<string, unknown>): string {
  const name = str(row.name).trim()
  if (name) return name
  return str(row.id) || '—'
}

export function wafCustomRuleDescription(row: Record<string, unknown>): string {
  const typ = str(row.type, 'regex') === 'contains' ? '子串' : 'Go 正则'
  const pattern = str(row.pattern)
  const short = pattern.length > 48 ? `${pattern.slice(0, 48)}…` : pattern
  const targets = arr<string>(row.targets).join(', ') || '—'
  const hosts = arr<string>(row.allow_hosts)
  const hostHint = hosts.length > 0 ? `；域名白名单 ${hosts.length} 条` : ''
  return `${typ} · ${short || '—'} → [${targets}]${hostHint}`
}

export function customActionOverridden(row: Record<string, unknown>): boolean {
  return wafActionIsExplicit(actionFromRow(row))
}

export function patchCustomRuleAction(
  doc: Record<string, unknown>,
  index: number,
  action: WAFAction,
): Record<string, unknown> {
  const rules = wafRulesFromDoc(doc).map((row, i) => {
    if (i !== index) return row
    const next = { ...row }
    if (action === 'inherit') {
      delete next.action
      delete next.log_only
    } else {
      next.action = action
      delete next.log_only
    }
    return next
  })
  return patchWafRules(doc, rules)
}
