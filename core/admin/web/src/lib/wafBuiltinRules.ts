// Mirrors built-in WAF rules from core/waf/builtin.go
// Keep in sync when backend builtins change.
export interface BuiltinWAFRule {
  id: string
  name: string
  type: string
  pattern: string
  targets: string[]
}

export const WAF_BUILTIN_RULES: BuiltinWAFRule[] = [
  {
    id: 'builtin:sqli-common',
    name: 'SQL injection probes (query/url)',
    type: 'regex',
    pattern: `(?is)(union\\s+select\\b|sleep\\s*\\(|benchmark\\s*\\(|;\\s*(drop|truncate|alter)\\s+table\\b)`,
    targets: ['uri'],
  },
  {
    id: 'builtin:path-traversal',
    name: 'Path traversal in request line',
    type: 'regex',
    pattern: `(?:\\.\\./|\\.\\.\\\\|%2e%2e%2f|%2e%2e\\\\\\\\|etc/passwd\\b)`,
    targets: ['path'],
  },
  {
    id: 'builtin:xss-lite',
    name: 'Reflected scripting probes (lite)',
    type: 'regex',
    pattern: `(?is)(<\\s*script\\b|javascript:\\s*|on\\w+\\s*=)`,
    targets: ['uri'],
  },
]
