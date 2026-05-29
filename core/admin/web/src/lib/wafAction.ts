import { bool, str } from './ingressModuleForms'

/** Persisted in YAML (builtin_rule_actions / waf.rules[].action). */
export type WAFStoredAction = 'block' | 'audit' | 'pass'

/** UI selection including inherit (omit from YAML). */
export type WAFAction = 'inherit' | WAFStoredAction

/** Global default disposition (maps to waf.log_only). */
export type WAFGlobalMode = 'block' | 'audit'

export const WAF_GLOBAL_MODE_OPTIONS: { value: WAFGlobalMode; label: string; hint: string }[] = [
  { value: 'block', label: '拦截', hint: '命中后阻断请求（默认）' },
  { value: 'audit', label: '记录', hint: '仅写入审计日志，不阻断请求' },
]

export const WAF_ACTION_OPTIONS: { value: WAFAction; label: string; hint: string }[] = [
  {
    value: 'inherit',
    label: '继承',
    hint: '跟随全局处置：全局为记录时仅记录，为拦截时拦截',
  },
  { value: 'block', label: '拦截', hint: '命中即阻断请求（显式覆盖全局）' },
  { value: 'audit', label: '仅记录', hint: '写入审计日志，继续检查后续规则' },
  { value: 'pass', label: '通过', hint: '命中即放行，不再执行后续签名规则' },
]

export function wafActionLabel(action: WAFAction): string {
  return WAF_ACTION_OPTIONS.find((o) => o.value === action)?.label ?? action
}

export function wafStoredActionFromString(raw: string): WAFStoredAction | null {
  const a = raw.trim().toLowerCase()
  if (a === 'block' || a === 'audit' || a === 'pass') return a
  return null
}

export function wafActionFromString(raw: string): WAFAction | null {
  if (raw.trim().toLowerCase() === 'inherit') return 'inherit'
  const stored = wafStoredActionFromString(raw)
  return stored ?? null
}

export function wafActionIsExplicit(action: WAFAction): boolean {
  return action !== 'inherit'
}

/** Effective disposition label when action is inherit. */
export function wafInheritEffectiveLabel(globalLogOnly: boolean): string {
  return globalLogOnly ? '记录' : '拦截'
}

export function wafGlobalModeLabel(globalLogOnly: boolean): string {
  return globalLogOnly ? '记录' : '拦截'
}

export function wafGlobalModeFromLogOnly(globalLogOnly: boolean): WAFGlobalMode {
  return globalLogOnly ? 'audit' : 'block'
}

/** Read action from a waf.rules[] row. */
export function actionFromRow(row: Record<string, unknown>): WAFAction {
  const parsed = wafStoredActionFromString(str(row.action))
  if (parsed) return parsed
  if (bool(row.log_only)) return 'audit'
  return 'inherit'
}

/** Read action from waf.builtin_rule_actions. */
export function actionFromBuiltinMap(
  actions: Record<string, unknown> | undefined,
  ruleId: string,
): WAFAction {
  if (!actions) return 'inherit'
  const parsed = wafStoredActionFromString(str(actions[ruleId]))
  return parsed ?? 'inherit'
}
