import { bool, str } from './ingressModuleForms'

export type WAFAction = 'block' | 'audit' | 'pass'

export const WAF_ACTION_OPTIONS: { value: WAFAction; label: string; hint: string }[] = [
  { value: 'block', label: '拦截', hint: '命中即阻断请求（默认）' },
  { value: 'audit', label: '仅记录', hint: '写入审计日志，继续检查后续规则' },
  { value: 'pass', label: '通过', hint: '命中即放行，不再执行后续签名规则' },
]

export function wafActionLabel(action: WAFAction): string {
  return WAF_ACTION_OPTIONS.find((o) => o.value === action)?.label ?? action
}

export function wafActionFromString(raw: string): WAFAction | null {
  const a = raw.trim().toLowerCase()
  if (a === 'block' || a === 'audit' || a === 'pass') return a
  return null
}

/** Read action from a waf.rules[] row or builtin_rule_actions value. */
export function actionFromRow(row: Record<string, unknown>): WAFAction {
  const parsed = wafActionFromString(str(row.action))
  if (parsed) return parsed
  if (bool(row.log_only)) return 'audit'
  return 'block'
}

export function actionFromBuiltinMap(
  actions: Record<string, unknown> | undefined,
  ruleId: string,
): WAFAction {
  if (!actions) return 'block'
  const parsed = wafActionFromString(str(actions[ruleId]))
  return parsed ?? 'block'
}
