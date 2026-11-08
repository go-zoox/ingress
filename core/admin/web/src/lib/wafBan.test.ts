import { describe, expect, it } from 'vitest'
import {
  BAN_TTL_PRESETS,
  NAT_WARNING_UA_THRESHOLD,
  banErrorCode,
  buildBanPayload,
  buildImpactWarnings,
  describeBanError,
  describeScope,
  filterBanRows,
  formatRemaining,
  goRegexToJS,
  isExpiringSoon,
  isValidCIDR,
  normalizeIP,
  resolveBanKind,
  summarizeBans,
  validateBanValue,
  validateRegex,
  type WAFBanPreview,
  type WAFBanRow,
} from './wafBan'

function row(overrides: Partial<WAFBanRow> = {}): WAFBanRow {
  return {
    id: 1,
    kind: 'ip',
    value: '203.0.113.42',
    hosts: [],
    scope: 'global',
    reason: '',
    source: 'manual',
    actor: 'admin',
    hits: 0,
    enabled: true,
    expires_at: null,
    active: true,
    created_at: '2026-11-07T10:00:00Z',
    ...overrides,
  }
}

function preview(overrides: Partial<WAFBanPreview> = {}): WAFBanPreview {
  return {
    kind: 'ip',
    value: '203.0.113.42',
    normalized: '203.0.113.42',
    hits_24h: 0,
    hosts: 0,
    distinct_ua: 0,
    paths: 0,
    trusted_proxy: false,
    in_allow_list: false,
    warnings: [],
    blocked: false,
    ...overrides,
  }
}

describe('normalizeIP', () => {
  it('accepts valid IPv4', () => {
    expect(normalizeIP('203.0.113.42')).toBe('203.0.113.42')
    expect(normalizeIP('  203.0.113.42  ')).toBe('203.0.113.42')
  })

  it('rejects out-of-range octets', () => {
    expect(normalizeIP('999.1.1.1')).toBeNull()
    expect(normalizeIP('203.0.113.256')).toBeNull()
  })

  it('rejects malformed addresses', () => {
    expect(normalizeIP('')).toBeNull()
    expect(normalizeIP('not-an-ip')).toBeNull()
    expect(normalizeIP('203.0.113')).toBeNull()
  })

  it('unwraps IPv4-mapped IPv6', () => {
    expect(normalizeIP('::ffff:203.0.113.42')).toBe('203.0.113.42')
  })

  it('accepts plain IPv6', () => {
    expect(normalizeIP('2001:db8::1')).toBe('2001:db8::1')
  })
})

describe('isValidCIDR', () => {
  it('accepts IPv4 CIDR', () => {
    expect(isValidCIDR('203.0.113.0/24')).toBe(true)
    expect(isValidCIDR('0.0.0.0/0')).toBe(true)
    expect(isValidCIDR('203.0.113.42/32')).toBe(true)
  })

  it('rejects bad prefix lengths', () => {
    expect(isValidCIDR('203.0.113.0/33')).toBe(false)
    expect(isValidCIDR('203.0.113.0/-1')).toBe(false)
  })

  it('rejects missing or bad address part', () => {
    expect(isValidCIDR('203.0.113.0')).toBe(false)
    expect(isValidCIDR('/24')).toBe(false)
    expect(isValidCIDR('999.0.0.0/24')).toBe(false)
  })
})

describe('goRegexToJS', () => {
  it('hoists inline case-insensitive flags', () => {
    expect(goRegexToJS('(?i)(nikto|sqlmap)')).toEqual({ source: '(nikto|sqlmap)', flags: 'i' })
  })

  it('handles combined inline flags', () => {
    const out = goRegexToJS('(?is)nikto')
    expect(out.flags).toBe('is')
    expect(out.source).toBe('nikto')
  })

  it('translates Go named groups to JS syntax', () => {
    expect(goRegexToJS('(?P<name>abc)').source).toBe('(?<name>abc)')
  })

  it('maps POSIX character classes', () => {
    expect(goRegexToJS('[[:digit:]]+').source).toBe('[0-9]+')
    expect(goRegexToJS('[[:alpha:]]+').source).toBe('[A-Za-z]+')
  })

  it('leaves plain patterns untouched', () => {
    expect(goRegexToJS('sqlmap')).toEqual({ source: 'sqlmap', flags: '' })
  })
})

describe('validateRegex', () => {
  it('accepts compilable patterns', () => {
    expect(validateRegex('(?i)(nikto|sqlmap)')).toBeNull()
    expect(validateRegex('^Mozilla/5\\.0')).toBeNull()
    expect(validateRegex('(?P<bot>nikto|sqlmap)')).toBeNull()
  })

  it('accepts Go-only syntax the backend would also accept', () => {
    // The backend compiles with Go regexp, which supports inline flags. The
    // form must not reject these as invalid JavaScript.
    expect(validateRegex('(?i)SQLMap')).toBeNull()
    expect(validateRegex('[[:digit:]]{1,3}')).toBeNull()
  })

  it('rejects broken patterns', () => {
    expect(validateRegex('(?i')).not.toBeNull()
    expect(validateRegex('(')).not.toBeNull()
    expect(validateRegex('')).not.toBeNull()
  })
})

describe('validateBanValue', () => {
  it('validates by kind', () => {
    expect(validateBanValue('ip', '203.0.113.42')).toEqual({ ok: true, normalized: '203.0.113.42' })
    expect(validateBanValue('ip', 'nope')).toMatchObject({ ok: false })
    expect(validateBanValue('ip_cidr', '203.0.113.0/24')).toEqual({ ok: true, normalized: '203.0.113.0/24' })
    expect(validateBanValue('ip_cidr', '203.0.113.0')).toMatchObject({ ok: false })
    expect(validateBanValue('ua_contains', 'sqlmap')).toEqual({ ok: true, normalized: 'sqlmap' })
    expect(validateBanValue('ua_regex', '(?i')).toMatchObject({ ok: false })
  })

  it('rejects empty input for every kind', () => {
    for (const kind of ['ip', 'ip_cidr', 'ua_contains', 'ua_regex'] as const) {
      expect(validateBanValue(kind, '   ')).toMatchObject({ ok: false })
    }
  })

  it('normalizes IPv4-mapped input', () => {
    expect(validateBanValue('ip', '::ffff:203.0.113.42')).toEqual({ ok: true, normalized: '203.0.113.42' })
  })
})

describe('resolveBanKind', () => {
  it('infers cidr from a slash', () => {
    expect(resolveBanKind('ip', '203.0.113.0/24', 'contains')).toBe('ip_cidr')
    expect(resolveBanKind('ip', '203.0.113.42', 'contains')).toBe('ip')
  })
  it('maps ua modes', () => {
    expect(resolveBanKind('ua', 'sqlmap', 'contains')).toBe('ua_contains')
    expect(resolveBanKind('ua', '(?i)sqlmap', 'regex')).toBe('ua_regex')
  })
})

describe('buildBanPayload', () => {
  it('converts preset ttl to seconds', () => {
    const payload = buildBanPayload({
      mode: 'ip',
      value: ' 203.0.113.42 ',
      uaMode: 'contains',
      ttl: '1h',
      reason: ' scan ',
      hosts: [],
    })
    expect(payload).toMatchObject({
      kind: 'ip',
      value: '203.0.113.42',
      ttl_seconds: 3600,
      reason: 'scan',
    })
    expect(payload.hosts).toBeUndefined()
  })

  it('omits ttl and sets confirm for permanent', () => {
    const payload = buildBanPayload({
      mode: 'ua',
      value: 'sqlmap',
      uaMode: 'contains',
      ttl: 'forever',
      reason: '',
      hosts: ['api.example.com'],
      confirmPermanent: true,
    })
    expect(payload.ttl_seconds).toBeUndefined()
    expect(payload.confirm_permanent).toBe(true)
    expect(payload.hosts).toEqual(['api.example.com'])
    expect(payload.kind).toBe('ua_contains')
  })

  it('defaults confirm_permanent to false when not passed', () => {
    const payload = buildBanPayload({
      mode: 'ip',
      value: '203.0.113.42',
      uaMode: 'contains',
      ttl: 'forever',
      reason: '',
      hosts: [],
    })
    expect(payload.confirm_permanent).toBe(false)
  })
})

describe('formatRemaining', () => {
  const now = new Date('2026-11-07T12:00:00Z')

  it('returns permanent for null', () => {
    expect(formatRemaining(null, now)).toBe('永久')
  })

  it('returns expired for past timestamps', () => {
    expect(formatRemaining('2026-11-07T11:00:00Z', now)).toBe('已过期')
  })

  it('formats minutes, hours, days', () => {
    expect(formatRemaining('2026-11-07T12:30:00Z', now)).toBe('剩余 30m')
    expect(formatRemaining('2026-11-07T18:00:00Z', now)).toBe('剩余 6h')
    expect(formatRemaining('2026-11-14T12:00:00Z', now)).toBe('剩余 7d')
  })
})

describe('isExpiringSoon', () => {
  const now = new Date('2026-11-07T12:00:00Z')

  it('is true within two hours', () => {
    expect(isExpiringSoon('2026-11-07T13:00:00Z', now)).toBe(true)
  })
  it('is false beyond the window', () => {
    expect(isExpiringSoon('2026-11-07T15:00:00Z', now)).toBe(false)
  })
  it('is false for permanent, expired, and invalid', () => {
    expect(isExpiringSoon(null, now)).toBe(false)
    expect(isExpiringSoon('2026-11-07T11:00:00Z', now)).toBe(false)
    expect(isExpiringSoon('not-a-date', now)).toBe(false)
  })
})

describe('describeScope', () => {
  it('labels global and host scopes', () => {
    expect(describeScope({ hosts: [] })).toBe('全局')
    expect(describeScope({ hosts: ['api.example.com'] })).toBe('api.example.com')
  })
})

describe('buildImpactWarnings', () => {
  it('returns nothing for a clean preview', () => {
    expect(buildImpactWarnings(null)).toEqual([])
    expect(buildImpactWarnings(preview())).toEqual([])
  })

  it('short-circuits on blocked targets', () => {
    const out = buildImpactWarnings(preview({ blocked: true, blocked_reason: '受信代理' }))
    expect(out).toEqual(['受信代理'])
  })

  it('warns about NAT when UA diversity is high', () => {
    const out = buildImpactWarnings(preview({ distinct_ua: NAT_WARNING_UA_THRESHOLD }))
    expect(out.join()).toContain('NAT')
  })

  it('does not warn about NAT below the threshold', () => {
    const out = buildImpactWarnings(preview({ distinct_ua: NAT_WARNING_UA_THRESHOLD - 1 }))
    expect(out.join()).not.toContain('NAT')
  })

  it('always warns that UA bans are bypassable', () => {
    const out = buildImpactWarnings(preview({ kind: 'ua_contains' }))
    expect(out.join()).toContain('客户端可控')
  })

  it('warns when the target is in the allow list', () => {
    const out = buildImpactWarnings(preview({ in_allow_list: true }))
    expect(out.join()).toContain('allow')
  })
})

describe('filterBanRows', () => {
  const rows = [
    row({ id: 1, kind: 'ip', value: '203.0.113.42', reason: 'sqlmap' }),
    row({ id: 2, kind: 'ua_contains', value: 'nikto', reason: 'scanner' }),
    row({ id: 3, kind: 'ip_cidr', value: '198.51.100.0/24', actor: 'alice' }),
  ]

  it('filters by kind family', () => {
    expect(filterBanRows(rows, '', 'ip').map((r) => r.id)).toEqual([1, 3])
    expect(filterBanRows(rows, '', 'ua').map((r) => r.id)).toEqual([2])
    expect(filterBanRows(rows, '', 'all')).toHaveLength(3)
  })

  it('filters by keyword across value/reason/actor', () => {
    expect(filterBanRows(rows, 'sqlmap', 'all').map((r) => r.id)).toEqual([1])
    expect(filterBanRows(rows, 'alice', 'all').map((r) => r.id)).toEqual([3])
    expect(filterBanRows(rows, 'SCANNER', 'all').map((r) => r.id)).toEqual([2])
  })

  it('returns everything for an empty keyword', () => {
    expect(filterBanRows(rows, '   ', 'all')).toHaveLength(3)
  })
})

describe('summarizeBans', () => {
  const now = new Date()
  const soon = new Date(now.getTime() + 30 * 60 * 1000).toISOString()

  it('splits by family and counts expiring', () => {
    const out = summarizeBans([
      row({ id: 1, kind: 'ip' }),
      row({ id: 2, kind: 'ip_cidr' }),
      row({ id: 3, kind: 'ua_contains', expires_at: soon }),
      row({ id: 4, kind: 'ua_regex', expires_at: null }),
    ])
    expect(out.total).toBe(4)
    expect(out.ip).toBe(2)
    expect(out.ua).toBe(2)
    expect(out.expiring).toBe(1)
  })

  it('handles an empty list', () => {
    expect(summarizeBans([])).toEqual({ total: 0, ip: 0, ua: 0, expiring: 0 })
  })
})

describe('ban error helpers', () => {
  it('extracts the code prefix', () => {
    expect(banErrorCode('trusted_proxy: bad')).toBe('trusted_proxy')
    expect(banErrorCode('no prefix here')).toBe('')
  })

  it('maps known codes to friendly copy', () => {
    expect(describeBanError('trusted_proxy: x')).toContain('受信代理')
    expect(describeBanError('confirm_required: x')).toContain('二次确认')
    expect(describeBanError('duplicate: x')).toContain('已存在')
    expect(describeBanError('not_found: x')).toContain('不存在')
  })

  it('passes through unknown messages', () => {
    expect(describeBanError('boom')).toBe('boom')
  })
})

describe('BAN_TTL_PRESETS', () => {
  it('exposes the four documented presets with consistent seconds', () => {
    expect(BAN_TTL_PRESETS.map((p) => p.value)).toEqual(['1h', '24h', '7d', 'forever'])
    const byValue = Object.fromEntries(BAN_TTL_PRESETS.map((p) => [p.value, p.seconds]))
    expect(byValue['1h']).toBe(3600)
    expect(byValue['24h']).toBe(86400)
    expect(byValue['7d']).toBe(604800)
    expect(byValue['forever']).toBe(0)
  })
})
