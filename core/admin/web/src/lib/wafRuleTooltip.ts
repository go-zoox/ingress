import type { WAFRuleDetail } from '../api/client'

/** Build a lookup map from catalog + resolve by event rule field. */
export function buildWafRuleLookup(catalog: WAFRuleDetail[]): Map<string, WAFRuleDetail> {
  const m = new Map<string, WAFRuleDetail>()
  const put = (key: string, d: WAFRuleDetail) => {
    const k = key.trim().toLowerCase()
    if (k && !m.has(k)) m.set(k, d)
  }
  for (const d of catalog) {
    put(d.id, d)
    put(d.name, d)
    if (d.id.startsWith('builtin:')) {
      put(d.id.slice('builtin:'.length), d)
    }
    put(d.phase, d)
  }
  return m
}

export function resolveWafRule(
  lookup: Map<string, WAFRuleDetail>,
  ruleField: string,
): WAFRuleDetail | undefined {
  const raw = ruleField.trim()
  if (!raw) return undefined

  const tryKey = (k: string) => lookup.get(k.trim().toLowerCase())

  const direct = tryKey(raw)
  if (direct) return direct

  if (raw.toLowerCase() === 'ip deny' || raw.toLowerCase() === 'ip allow') {
    return tryKey(raw)
  }

  let id = raw
  if (raw.toLowerCase().startsWith('sig ')) {
    id = raw.slice(4).trim()
  }
  const byID = tryKey(id)
  if (byID) return byID
  const byBuiltin = tryKey(`builtin:${id}`)
  if (byBuiltin) return byBuiltin

  return undefined
}

const SOURCE_LABEL: Record<string, string> = {
  config: '配置文件',
  builtin: '内置',
  demo: '演示数据',
  phase: '阶段',
  unknown: '未知',
}

export function formatWafRuleTooltip(
  detail: WAFRuleDetail | undefined,
  ruleField: string,
): string {
  if (!detail) {
    return ruleField
      ? `${ruleField}\n未找到规则定义（可能为历史或演示事件）`
      : '无规则信息'
  }
  const lines: string[] = []
  if (detail.name) lines.push(detail.name)
  lines.push(`标识: ${ruleField}`)
  if (detail.type) lines.push(`类型: ${detail.type}`)
  if (detail.source) lines.push(`来源: ${SOURCE_LABEL[detail.source] ?? detail.source}`)
  if (detail.phase && detail.phase !== ruleField) lines.push(`阶段: ${detail.phase}`)
  if (detail.pattern) lines.push(`模式: ${detail.pattern}`)
  if (detail.targets?.length) lines.push(`目标: ${detail.targets.join(', ')}`)
  if (detail.action && detail.action !== 'block') {
    const act =
      detail.action === 'audit' ? '仅记录' : detail.action === 'pass' ? '通过' : detail.action
    lines.push(`处置: ${act}`)
  } else if (detail.log_only) {
    lines.push('规则级: 仅审计 (log_only)')
  }
  if (detail.description) lines.push(detail.description)
  return lines.join('\n')
}
