import { describe, expect, it } from 'vitest'
import { buildWafRuleLookup, formatWafRuleTooltip, resolveBanRule, resolveWafRule } from './wafRuleTooltip'

describe('resolveBanRule', () => {
  it('synthesizes a rule for each supported ban kind', () => {
    for (const kind of ['ip', 'ip_cidr', 'ua_contains', 'ua_regex']) {
      const detail = resolveBanRule(`ban:${kind}`)
      expect(detail).toBeDefined()
      expect(detail?.id).toBe(`ban:${kind}`)
      expect(detail?.source).toBe('runtime')
      expect(detail?.name).toContain('运行时封禁')
    }
  })

  it('is case-insensitive', () => {
    expect(resolveBanRule('BAN:IP')?.id).toBe('ban:ip')
  })

  it('returns undefined for non-ban rules and unknown kinds', () => {
    expect(resolveBanRule('sql-injection-uri')).toBeUndefined()
    expect(resolveBanRule('ip deny')).toBeUndefined()
    expect(resolveBanRule('ban:nope')).toBeUndefined()
    expect(resolveBanRule('')).toBeUndefined()
  })
})

describe('resolveWafRule with ban events', () => {
  it('resolves a ban rule even when the catalog is empty', () => {
    // Ban events come from the runtime ban list, never from the config catalog;
    // previously they fell through to the "no rule definition" tooltip.
    const lookup = buildWafRuleLookup([])
    const detail = resolveWafRule(lookup, 'ban:ip')
    expect(detail).toBeDefined()
    expect(detail?.source).toBe('runtime')
  })

  it('prefers a real catalog entry when one exists', () => {
    const lookup = buildWafRuleLookup([
      {
        id: 'ban:ip',
        name: 'Custom override',
        phase: 'ban',
        type: 'regex',
        source: 'config',
        description: 'from yaml',
        enabled: true,
      },
    ])
    expect(resolveWafRule(lookup, 'ban:ip')?.source).toBe('config')
  })

  it('still resolves ordinary signature rules', () => {
    const lookup = buildWafRuleLookup([
      {
        id: 'builtin:scanner-ua',
        name: 'Scanner User-Agent',
        phase: 'scanner-ua',
        type: 'regex',
        source: 'builtin',
        description: '',
        enabled: true,
      },
    ])
    expect(resolveWafRule(lookup, 'sig builtin:scanner-ua')?.id).toBe('builtin:scanner-ua')
    expect(resolveWafRule(lookup, 'scanner-ua')?.id).toBe('builtin:scanner-ua')
  })
})

describe('formatWafRuleTooltip for ban events', () => {
  it('does not claim the rule is missing', () => {
    const detail = resolveBanRule('ban:ua_contains')
    const text = formatWafRuleTooltip(detail, 'ban:ua_contains')
    expect(text).not.toContain('未找到规则定义')
    expect(text).toContain('运行时封禁')
    expect(text).toContain('运行时封禁')
  })

  it('labels the source as runtime', () => {
    const detail = resolveBanRule('ban:ip')
    const text = formatWafRuleTooltip(detail, 'ban:ip')
    expect(text).toContain('来源: 运行时封禁')
  })
})
