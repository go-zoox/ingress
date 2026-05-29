import { obj } from './ingressModuleForms'
import { actionFromBuiltinMap, type WAFAction } from './wafAction'

// Mirrors built-in WAF rules from core/waf/builtin.go
// Keep in sync when backend builtins change.
export interface BuiltinWAFRule {
  id: string
  name: string
  type: string
  pattern: string
  targets: string[]
  description?: string
}

export const WAF_BUILTIN_RULES: BuiltinWAFRule[] = [
  {
    id: 'builtin:sqli-common',
    name: 'SQL injection probes (query/url)',
    type: 'regex',
    pattern: `(?is)(union\\s+select\\b|sleep\\s*\\(|benchmark\\s*\\(|;\\s*(drop|truncate|alter)\\s+table\\b)`,
    targets: ['uri'],
    description: 'URI 中常见 SQL 注入探针（union select、sleep、benchmark 等）',
  },
  {
    id: 'builtin:path-traversal',
    name: 'Path traversal in request line',
    type: 'regex',
    pattern: `(?:\\.\\./|\\.\\.\\\\|%2e%2e%2f|%2e%2e\\\\\\\\|etc/passwd\\b)`,
    targets: ['path'],
    description: '路径遍历（../、编码 ..、/etc/passwd）',
  },
  {
    id: 'builtin:xss-lite',
    name: 'Reflected scripting probes (lite)',
    type: 'regex',
    pattern: `(?is)(<\\s*script\\b|javascript:\\s*[a-z]|\\bon(?:click|load|error|focus|blur|change|submit|mouse\\w*|key\\w*|touch\\w*|pointer\\w*|scroll|dblclick|drag\\w*|drop|input|reset|select|wheel|copy|cut|paste|abort|contextmenu|message|unload|beforeunload)\\s*=)`,
    targets: ['uri'],
    description: '反射型 XSS 探针（script 标签、javascript: 协议、常见 on* 事件属性）',
  },
  {
    id: 'builtin:rce-probes',
    name: 'Command injection probes',
    type: 'regex',
    pattern: `(?i)(\\||;|&&|\\$\\(|\\\`)\\s*(cat|curl|wget|bash|/bin/sh|cmd\\.exe|powershell)\\b`,
    targets: ['uri'],
    description: '命令注入/RCE 探针（管道、shell 命令关键字）',
  },
  {
    id: 'builtin:jndi-lookup',
    name: 'JNDI lookup injection',
    type: 'regex',
    pattern: `(?i)\\$\\{[^}]*jndi:`,
    targets: ['uri', 'headers'],
    description: 'Log4j 等 JNDI 注入（${...jndi:）',
  },
  {
    id: 'builtin:sensitive-files',
    name: 'Sensitive file and admin path probes',
    type: 'regex',
    pattern: `(?i)(/\\.env\\b|/\\.git/|/wp-admin\\b|/phpmyadmin\\b|/web\\.config\\b|/id_rsa\\b)`,
    targets: ['path'],
    description: '敏感文件与管理后台路径探测',
  },
  {
    id: 'builtin:ssrf-probes',
    name: 'SSRF and metadata endpoint probes',
    type: 'regex',
    pattern: `(?i)(169\\.254\\.169\\.254|metadata\\.google|file://|gopher://)`,
    targets: ['uri'],
    description: '云元数据 / file:// / gopher:// SSRF 探针（不含 localhost）',
  },
  {
    id: 'builtin:scanner-ua',
    name: 'Known scanner User-Agent',
    type: 'regex',
    pattern: `(?i)(sqlmap|nikto|nmap|masscan|acunetix|nessus|dirbuster|gobuster)`,
    targets: ['header:User-Agent'],
    description: '常见扫描器 User-Agent',
  },
  {
    id: 'builtin:crlf-injection',
    name: 'CRLF / response-splitting probes',
    type: 'regex',
    pattern: `(?i)(%0d%0a|%0a%0d|\\r\\n)(content-length|set-cookie|location):`,
    targets: ['uri', 'headers'],
    description: 'CRLF 注入 / HTTP 响应拆分',
  },
  {
    id: 'builtin:php-ssti',
    name: 'PHP / template injection probes',
    type: 'regex',
    pattern: `(?i)(eval\\s*\\(|base64_decode\\s*\\(|php://|expect://)`,
    targets: ['uri'],
    description: 'PHP eval、伪协议等注入探针',
  },
]

export function defaultBuiltinEnabled(doc: Record<string, unknown>): boolean {
  const waf = doc.waf
  if (!waf || typeof waf !== 'object') return true
  return !(waf as Record<string, unknown>).disable_builtin
}

export function isBuiltinRuleEnabled(doc: Record<string, unknown>, ruleId: string): boolean {
  const waf = (doc.waf ?? {}) as Record<string, unknown>
  const overrides = (waf.builtin_rules ?? {}) as Record<string, unknown>
  if (Object.prototype.hasOwnProperty.call(overrides, ruleId)) {
    return overrides[ruleId] === true
  }
  return defaultBuiltinEnabled(doc)
}

export function patchBuiltinRuleEnabled(
  doc: Record<string, unknown>,
  ruleId: string,
  enabled: boolean,
): Record<string, unknown> {
  const waf = { ...((doc.waf ?? {}) as Record<string, unknown>) }
  const overrides = { ...((waf.builtin_rules ?? {}) as Record<string, unknown>) }
  const def = defaultBuiltinEnabled(doc)
  if (enabled === def) {
    delete overrides[ruleId]
  } else {
    overrides[ruleId] = enabled
  }
  if (Object.keys(overrides).length > 0) {
    waf.builtin_rules = overrides
  } else {
    delete waf.builtin_rules
  }
  return { ...doc, waf }
}

export function builtinRuleActions(doc: Record<string, unknown>): Record<string, unknown> {
  return obj(obj(doc.waf).builtin_rule_actions)
}

export function builtinRuleAction(doc: Record<string, unknown>, ruleId: string): WAFAction {
  return actionFromBuiltinMap(builtinRuleActions(doc) as Record<string, unknown>, ruleId)
}

export function patchBuiltinRuleAction(
  doc: Record<string, unknown>,
  ruleId: string,
  action: WAFAction,
): Record<string, unknown> {
  const waf = { ...((doc.waf ?? {}) as Record<string, unknown>) }
  const actions = { ...builtinRuleActions(doc) }
  if (action === 'block') {
    delete actions[ruleId]
  } else {
    actions[ruleId] = action
  }
  if (Object.keys(actions).length > 0) {
    waf.builtin_rule_actions = actions
  } else {
    delete waf.builtin_rule_actions
  }
  return { ...doc, waf }
}

export function builtinActionOverridden(doc: Record<string, unknown>, ruleId: string): boolean {
  return Object.prototype.hasOwnProperty.call(builtinRuleActions(doc), ruleId)
}
