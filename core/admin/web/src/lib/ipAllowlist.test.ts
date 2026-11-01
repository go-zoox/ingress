import { describe, expect, it } from 'vitest'
import {
  buildIPAllowlistYAML,
  cidrContains,
  evaluateAllowlist,
  isValidIPOrCIDR,
  parseIP,
  resolveClientIP,
  stripPort,
  trustedProxiesFromDoc,
} from './ipAllowlist'

describe('parseIP / cidrContains', () => {
  it('parses IPv4 and IPv6', () => {
    expect(parseIP('10.1.2.3')?.family).toBe(4)
    expect(parseIP('2001:db8::1')?.family).toBe(6)
    expect(parseIP('::ffff:10.0.0.1')?.family).toBe(4)
    expect(parseIP('not-an-ip')).toBeNull()
  })

  it('matches IPv4 CIDRs', () => {
    expect(cidrContains('10.0.0.0/8', '10.1.2.3')).toBe(true)
    expect(cidrContains('10.0.0.0/8', '11.0.0.1')).toBe(false)
    expect(cidrContains('203.0.113.7/32', '203.0.113.7')).toBe(true)
    expect(cidrContains('203.0.113.7', '203.0.113.7')).toBe(true)
    expect(cidrContains('0.0.0.0/0', '198.51.100.1')).toBe(true)
  })

  it('matches IPv6 CIDRs', () => {
    expect(cidrContains('fc00::/7', 'fd12:3456::1')).toBe(true)
    expect(cidrContains('2001:db8::/32', '2001:db8:1::1')).toBe(true)
    expect(cidrContains('2001:db8::/32', '2001:db9::1')).toBe(false)
  })

  it('rejects cross-family matches', () => {
    expect(cidrContains('10.0.0.0/8', '2001:db8::1')).toBe(false)
  })

  it('validates IP / CIDR entries', () => {
    expect(isValidIPOrCIDR('10.0.0.0/8')).toBe(true)
    expect(isValidIPOrCIDR('fc00::/7')).toBe(true)
    expect(isValidIPOrCIDR('10.0.0.0/33')).toBe(false)
    expect(isValidIPOrCIDR('300.1.1.1')).toBe(false)
  })
})

describe('stripPort', () => {
  it('strips IPv4 ports and IPv6 brackets', () => {
    expect(stripPort('10.0.0.5:1234')).toBe('10.0.0.5')
    expect(stripPort('[2001:db8::1]:443')).toBe('2001:db8::1')
    expect(stripPort('2001:db8::1')).toBe('2001:db8::1')
    expect(stripPort('10.0.0.5')).toBe('10.0.0.5')
  })
})

describe('resolveClientIP', () => {
  it('uses RemoteAddr when no trusted proxies are configured', () => {
    expect(resolveClientIP('198.51.100.23:1234', '203.0.113.7', []).ip).toBe('198.51.100.23')
  })

  it('ignores XFF when the peer is not a trusted proxy', () => {
    const out = resolveClientIP('198.51.100.23:1234', '203.0.113.7', ['10.0.0.0/8'])
    expect(out.ip).toBe('198.51.100.23')
    expect(out.source).toMatch(/已忽略/)
  })

  it('takes the rightmost untrusted XFF hop', () => {
    const out = resolveClientIP('10.0.0.5:1234', '203.0.113.7, 10.0.0.2', ['10.0.0.0/8'])
    expect(out.ip).toBe('203.0.113.7')
    expect(out.source).toMatch(/右起/)
  })

  it('falls back to the peer when every XFF hop is trusted', () => {
    expect(resolveClientIP('10.0.0.5:1234', '10.0.0.9, 10.0.0.2', ['10.0.0.0/8']).ip).toBe(
      '10.0.0.5',
    )
  })
})

describe('evaluateAllowlist', () => {
  it('deny wins over allow', () => {
    expect(evaluateAllowlist(['10.0.0.0/8'], ['10.0.0.9/32'], '10.0.0.9')).toEqual({
      allowed: false,
      reason: '命中 deny',
    })
  })

  it('allows when allow matches', () => {
    expect(evaluateAllowlist(['10.0.0.0/8'], [], '10.1.1.1').allowed).toBe(true)
  })

  it('blocks when allow does not match', () => {
    expect(evaluateAllowlist(['10.0.0.0/8'], [], '198.51.100.1').allowed).toBe(false)
  })

  it('empty allow means unrestricted', () => {
    expect(evaluateAllowlist([], [], '198.51.100.1').allowed).toBe(true)
  })
})

describe('trustedProxiesFromDoc', () => {
  it('reads waf.trusted_proxies', () => {
    expect(trustedProxiesFromDoc({ waf: { trusted_proxies: ['10.0.0.0/8', ' 172.16.0.0/12 '] } })).toEqual([
      '10.0.0.0/8',
      '172.16.0.0/12',
    ])
    expect(trustedProxiesFromDoc({})).toEqual([])
  })
})

describe('buildIPAllowlistYAML', () => {
  it('shows the unrestricted comment when disabled', () => {
    const yaml = buildIPAllowlistYAML({
      host: 'app.example.com',
      enabled: false,
      allow: [],
      deny: [],
      signatures: false,
    })
    expect(yaml).toContain('未配置 IP 白名单')
    expect(yaml).not.toContain('waf:')
  })

  it('writes enabled/disable_builtin/allow for IP-only mode', () => {
    const yaml = buildIPAllowlistYAML({
      host: 'admin.example.com',
      enabled: true,
      allow: ['10.0.0.0/8'],
      deny: ['203.0.113.66/32'],
      signatures: false,
    })
    expect(yaml).toContain('enabled: true')
    expect(yaml).toContain('disable_builtin: true')
    expect(yaml).toContain("- '10.0.0.0/8'")
    expect(yaml).toContain("- '203.0.113.66/32'")
    expect(yaml).toContain('block_status_code: 403')
  })
})
