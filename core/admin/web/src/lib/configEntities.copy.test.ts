import { describe, expect, it } from 'vitest'
import {
  cloneRuleRow,
  copyHostValidation,
  copyHostWarning,
  effectiveHostTypeForHost,
  findHostConflict,
  formToRule,
  insertCopiedRule,
  ruleToForm,
  suggestCopyHost,
} from './configEntities'

function hostRows(...hosts: string[]): Record<string, unknown>[] {
  return hosts.map((host, index) => ({ host, id: `r${index + 1}` }))
}

describe('effectiveHostTypeForHost', () => {
  it('mirrors the compile-time inference order (regex before wildcard)', () => {
    expect(effectiveHostTypeForHost('api.example.com')).toBe('exact')
    expect(effectiveHostTypeForHost('*.example.com')).toBe('wildcard')
    expect(effectiveHostTypeForHost('^legacy\\.example\\.com$')).toBe('regex')
    expect(effectiveHostTypeForHost('^.*\\.example\\.com$')).toBe('regex')
    expect(effectiveHostTypeForHost('')).toBe('exact')
  })
})

describe('suggestCopyHost', () => {
  it('appends -copy to the first label for exact hosts', () => {
    const suggestion = suggestCopyHost('api.example.com', [])
    expect(suggestion.host).toBe('api-copy.example.com')
    expect(suggestion.auto).toBe(true)
    expect(suggestion.firstSuggestion).toBe('api-copy.example.com')
    expect(suggestion.collisionAvoided).toBe(false)
  })

  it('handles single-label hosts', () => {
    expect(suggestCopyHost('localhost', []).host).toBe('localhost-copy')
  })

  it('keeps the *. prefix and only patches the label after it', () => {
    expect(suggestCopyHost('*.shop.example.com', []).host).toBe('*.shop-copy.example.com')
  })

  it('walks -copy-2 / -copy-3 when suggestions are taken (case-insensitive)', () => {
    const suggestion = suggestCopyHost('admin.example.com', [
      'admin-copy.example.com',
      'ADMIN-COPY-2.EXAMPLE.COM',
    ])
    expect(suggestion.host).toBe('admin-copy-3.example.com')
    expect(suggestion.firstSuggestion).toBe('admin-copy.example.com')
    expect(suggestion.collisionAvoided).toBe(true)
  })

  it('continues the numbering when copying a copy', () => {
    expect(suggestCopyHost('api-copy.example.com', []).host).toBe('api-copy-2.example.com')
    expect(suggestCopyHost('api-copy-2.example.com', []).host).toBe('api-copy-3.example.com')
  })

  it('never auto-renames regex hosts', () => {
    const suggestion = suggestCopyHost('^legacy\\.example\\.com$', [])
    expect(suggestion.host).toBe('^legacy\\.example\\.com$')
    expect(suggestion.auto).toBe(false)
  })

  it('honours an explicit host_type instead of inferring from the string', () => {
    const explicitRegex = suggestCopyHost('plain.example.com', [], 'regex')
    expect(explicitRegex.host).toBe('plain.example.com')
    expect(explicitRegex.auto).toBe(false)

    const explicitWildcard = suggestCopyHost('*.example.com', [], 'wildcard')
    expect(explicitWildcard.host).toBe('*.example-copy.com')
  })
})

describe('findHostConflict / copyHostValidation', () => {
  const rules = hostRows('www.example.com', 'API.example.com')

  it('matches hosts case-insensitively', () => {
    expect(findHostConflict('api.example.com', rules)?.index).toBe(1)
    expect(findHostConflict('  API.EXAMPLE.COM  ', rules)?.index).toBe(1)
  })

  it('can skip the current edit index', () => {
    expect(findHostConflict('www.example.com', rules, 0)).toBeNull()
  })

  it('blocks empty, whitespace and duplicate hosts', () => {
    expect(copyHostValidation('', rules)).toBe('Host 不能为空')
    expect(copyHostValidation('bad host.example.com', rules)).toContain('空白字符')
    const duplicate = copyHostValidation('www.example.com', rules)
    expect(duplicate).toContain('规则 #1')
    expect(duplicate).toContain('www.example.com')
    expect(copyHostValidation('copy.example.com', rules)).toBeNull()
  })
})

describe('copyHostWarning', () => {
  it('only warns for regex hosts', () => {
    expect(copyHostWarning('^legacy\\.example\\.com$')).not.toBeNull()
    expect(copyHostWarning('plain.example.com', 'regex')).not.toBeNull()
    expect(copyHostWarning('api.example.com')).toBeNull()
    expect(copyHostWarning('*.example.com')).toBeNull()
  })
})

describe('insertCopiedRule', () => {
  const rules = hostRows('a.example.com', 'b.example.com', 'c.example.com')

  it('inserts right after the source rule by default', () => {
    const result = insertCopiedRule(rules, 1, 'after', { host: 'b-copy.example.com' })
    expect(result.index).toBe(2)
    expect(result.rules.map((row) => row.host)).toEqual([
      'a.example.com',
      'b.example.com',
      'b-copy.example.com',
      'c.example.com',
    ])
  })

  it('appends to the tail when requested', () => {
    const result = insertCopiedRule(rules, 1, 'end', { host: 'b-copy.example.com' })
    expect(result.index).toBe(3)
    expect(result.rules[3]?.host).toBe('b-copy.example.com')
  })

  it('clamps out-of-range source indexes and never mutates the input', () => {
    const result = insertCopiedRule(rules, -1, 'after', { host: 'x.example.com' })
    expect(result.index).toBe(0)
    expect(rules).toHaveLength(3)
  })
})

describe('copy fidelity (deep copy via formToRule)', () => {
  function sourceRule(): Record<string, unknown> {
    return {
      host: 'orders.example.com',
      host_type: 'auto',
      enabled: true,
      backend: {
        type: 'service',
        service: {
          name: 'orders',
          port: 8080,
          request: { headers: { 'X-From': 'ingress' }, cache_ttl: 0 },
        },
        cache: { enabled: true, ttl: 300, key: { headers: ['Accept'] } },
        unknown_backend_key: 'keep-me',
      },
      paths: [
        {
          path: '/v2/orders',
          backend: { service: { name: 'orders', port: 8080, strip_prefix: true } },
          unknown_path_key: 'keep-path',
        },
      ],
      waf: {
        enabled: true,
        disable_builtin: true,
        allow: ['10.0.0.0/8'],
        rules: [{ id: 'custom-sqli', pattern: 'x' }],
      },
      rate_limit: { requests: 100, period: 60 },
      security: { profile: 'api' },
      connect_proxy: { enabled: false },
    }
  }

  it('keeps every field and the unknown keys when the source row is passed as origin', () => {
    const source = sourceRule()
    const form = ruleToForm(source)
    form.host = 'orders-copy.example.com'
    const copy = formToRule(form, cloneRuleRow(source))

    expect(copy.host).toBe('orders-copy.example.com')
    expect((copy.backend as Record<string, unknown>).unknown_backend_key).toBe('keep-me')
    expect(
      (((copy.backend as Record<string, unknown>).service as Record<string, unknown>)
        .request as Record<string, unknown>).headers,
    ).toEqual({ 'X-From': 'ingress' })
    const copyPaths = copy.paths as Record<string, unknown>[]
    expect(copyPaths.map((row) => row.path)).toEqual(['/v2/orders'])
    expect(copyPaths[0]?.unknown_path_key).toBe('keep-path')
    expect(((copyPaths[0]?.backend as Record<string, unknown>).service as Record<string, unknown>)).toMatchObject({
      name: 'orders',
      port: 8080,
      strip_prefix: true,
    })
    expect((copy.waf as Record<string, unknown>).rules).toEqual(source.waf ? (source.waf as Record<string, unknown>).rules : undefined)
    expect(copy.rate_limit).toEqual({ requests: 100, period: 60 })
    expect(copy.security).toMatchObject({ profile: 'api' })
  })

  it('keeps the duplicate independent from its source', () => {
    const source = sourceRule()
    const before = JSON.parse(JSON.stringify(source))
    const form = ruleToForm(source)
    form.host = 'orders-copy.example.com'
    const copy = formToRule(form, cloneRuleRow(source))

    const copyWaf = copy.waf as Record<string, unknown>
    ;(copyWaf.rules as Record<string, unknown>[]).push({ id: 'only-on-copy' })
    const copyBackend = copy.backend as Record<string, unknown>
    const copyService = copyBackend.service as Record<string, unknown>
    ;((copyService.request as Record<string, unknown>).headers as Record<string, unknown>)['X-From'] = 'copy'
    ;(copy.paths as Record<string, unknown>[])[0]!.path = '/copy-only'

    expect(source).toEqual(before)
  })

  it('falls back to the source host_type when the form keeps auto', () => {
    const source = sourceRule()
    const form = ruleToForm(source)
    form.host = 'orders-copy.example.com'
    const copy = formToRule(form, cloneRuleRow(source))
    expect(copy.host_type).toBeUndefined()
  })
})
