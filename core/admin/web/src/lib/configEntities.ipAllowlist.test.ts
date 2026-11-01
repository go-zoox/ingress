import { describe, expect, it } from 'vitest'
import {
  buildIPAllowlistWaf,
  emptyRuleForm,
  formToRule,
  ipAllowlistFromDoc,
  ruleIPAllowlistLabel,
  ruleSaveDisabled,
  ruleToForm,
  type IPAllowlistFormSlice,
  type RuleForm,
} from './configEntities'

function ipForm(overrides: Partial<IPAllowlistFormSlice> = {}): IPAllowlistFormSlice {
  return {
    waf_ip_enabled: true,
    waf_ip_allow: ['10.0.0.0/8'],
    waf_ip_deny: [],
    waf_ip_signatures: false,
    ...overrides,
  }
}

function baseRule(): Record<string, unknown> {
  return {
    host: 'admin.example.com',
    backend: { type: 'service', service: { name: 'admin-svc', port: 8080 } },
    waf: {
      enabled: true,
      disable_builtin: true,
      allow: ['10.0.0.0/8', '192.168.0.0/16'],
      block_status_code: 403,
    },
  }
}

describe('ipAllowlistFromDoc', () => {
  it('detects an enabled allowlist and the IP-only signature default', () => {
    const form = ipAllowlistFromDoc({ allow: ['10.0.0.0/8'], disable_builtin: true })
    expect(form.waf_ip_enabled).toBe(true)
    expect(form.waf_ip_allow).toEqual(['10.0.0.0/8'])
    expect(form.waf_ip_signatures).toBe(false)
  })

  it('treats explicit disable_builtin: false as "signatures on"', () => {
    expect(ipAllowlistFromDoc({ allow: ['10.0.0.0/8'], disable_builtin: false }).waf_ip_signatures).toBe(
      true,
    )
  })

  it('is disabled when allow/deny are empty', () => {
    expect(ipAllowlistFromDoc({ enabled: true }).waf_ip_enabled).toBe(false)
  })
})

describe('ruleToForm', () => {
  it('parses rules[].waf into the form slice', () => {
    const form = ruleToForm(baseRule())
    expect(form.waf_ip_enabled).toBe(true)
    expect(form.waf_ip_allow).toEqual(['10.0.0.0/8', '192.168.0.0/16'])
    expect(form.waf_ip_signatures).toBe(false)
  })
})

describe('formToRule', () => {
  it('writes enabled + disable_builtin + allow/deny + block_status_code', () => {
    const form: RuleForm = { ...emptyRuleForm(), host: 'admin.example.com', ...ipForm({ waf_ip_deny: ['203.0.113.66/32'] }) }
    const rule = formToRule(form)
    expect(rule.waf).toEqual({
      enabled: true,
      disable_builtin: true,
      allow: ['10.0.0.0/8'],
      deny: ['203.0.113.66/32'],
      block_status_code: 403,
    })
  })

  it('omits disable_builtin when attack signatures are enabled', () => {
    const form: RuleForm = { ...emptyRuleForm(), host: 'admin.example.com', ...ipForm({ waf_ip_signatures: true }) }
    const rule = formToRule(form)
    expect((rule.waf as Record<string, unknown>).disable_builtin).toBeUndefined()
  })

  it('preserves other rules[].waf keys (custom signatures, allow_hosts)', () => {
    const original = baseRule()
    original.waf = {
      ...(original.waf as Record<string, unknown>),
      allow_hosts: ['*.internal.example.com'],
      rules: [{ id: 'custom', pattern: 'BAD', targets: ['query'] }],
    }
    const form = ruleToForm(original)
    const rule = formToRule(form, original)
    const waf = rule.waf as Record<string, unknown>
    expect(waf.allow_hosts).toEqual(['*.internal.example.com'])
    expect(waf.rules).toEqual([{ id: 'custom', pattern: 'BAD', targets: ['query'] }])
    expect(waf.allow).toEqual(['10.0.0.0/8', '192.168.0.0/16'])
  })

  it('removes the whole waf block when an IP-only allowlist is turned off', () => {
    const original = baseRule()
    const form = ruleToForm(original)
    form.waf_ip_enabled = false
    form.waf_ip_allow = []
    const rule = formToRule(form, original)
    expect(rule.waf).toBeUndefined()
  })

  it('keeps custom signatures when the allowlist is turned off', () => {
    const original = baseRule()
    original.waf = {
      ...(original.waf as Record<string, unknown>),
      rules: [{ id: 'custom', pattern: 'BAD', targets: ['query'] }],
    }
    const form = ruleToForm(original)
    form.waf_ip_enabled = false
    form.waf_ip_allow = []
    const rule = formToRule(form, original)
    const waf = rule.waf as Record<string, unknown>
    expect(waf.allow).toBeUndefined()
    expect(waf.deny).toBeUndefined()
    expect(waf.rules).toEqual([{ id: 'custom', pattern: 'BAD', targets: ['query'] }])
  })

  it('does not touch unrelated WAF config when the feature was never used', () => {
    const original = {
      host: 'app.example.com',
      backend: { type: 'service', service: { name: 'app', port: 8080 } },
      waf: { enabled: true, rules: [{ id: 'custom', pattern: 'BAD', targets: ['query'] }] },
    }
    const form = ruleToForm(original)
    const rule = formToRule(form, original)
    expect(rule.waf).toEqual(original.waf)
  })
})

describe('buildIPAllowlistWaf', () => {
  it('returns an empty map when disabled and nothing was configured', () => {
    expect(buildIPAllowlistWaf(ipForm({ waf_ip_enabled: false, waf_ip_allow: [] }), {})).toEqual({})
  })
})

describe('ruleSaveDisabled', () => {
  it('blocks saving an enabled but empty allowlist', () => {
    const form: RuleForm = { ...emptyRuleForm(), ...ipForm({ waf_ip_allow: [], waf_ip_deny: [] }) }
    expect(ruleSaveDisabled(form)).toBe(true)
  })

  it('blocks saving invalid IP / CIDR entries', () => {
    const form: RuleForm = { ...emptyRuleForm(), ...ipForm({ waf_ip_allow: ['10.0.0.0/33'] }) }
    expect(ruleSaveDisabled(form)).toBe(true)
  })

  it('allows saving a valid allowlist', () => {
    const form: RuleForm = { ...emptyRuleForm(), ...ipForm() }
    expect(ruleSaveDisabled(form)).toBe(false)
  })
})

describe('ruleIPAllowlistLabel', () => {
  it('renders allow / deny counts', () => {
    expect(ruleIPAllowlistLabel(baseRule())).toBe('allow 2')
    expect(
      ruleIPAllowlistLabel({ waf: { allow: ['10.0.0.0/8'], deny: ['203.0.113.66/32'] } }),
    ).toBe('allow 1 · deny 1')
  })

  it('returns null when unrestricted', () => {
    expect(ruleIPAllowlistLabel({ host: 'app.example.com' })).toBeNull()
  })
})
