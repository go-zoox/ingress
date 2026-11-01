import { arr, obj } from './ingressModuleForms'

/** Parsed IPv4/IPv6 address as a big integer (family + value). */
type ParsedIP = { family: 4 | 6; value: bigint }
type ParsedCIDR = ParsedIP & { prefix: number }

function ipv4ToBigInt(ip: string): bigint | null {
  const parts = ip.trim().split('.')
  if (parts.length !== 4) return null
  let value = 0n
  for (const part of parts) {
    if (!/^\d{1,3}$/.test(part)) return null
    const n = Number(part)
    if (n > 255) return null
    value = value * 256n + BigInt(n)
  }
  return value
}

function ipv6ToBigInt(ip: string): bigint | null {
  let addr = ip.trim().toLowerCase()
  if (!addr || !addr.includes(':')) return null
  if (addr.includes(':::')) return null
  // Embedded IPv4 tail (e.g. ::ffff:192.0.2.1) -> convert to two hex groups.
  const v4Tail = addr.match(/(\d+\.\d+\.\d+\.\d+)$/)
  if (v4Tail) {
    const v4 = ipv4ToBigInt(v4Tail[1])
    if (v4 === null) return null
    const hi = (v4 >> 16n) & 0xffffn
    const lo = v4 & 0xffffn
    addr = addr.slice(0, v4Tail.index) + hi.toString(16) + ':' + lo.toString(16)
  }

  const halves = addr.split('::')
  if (halves.length > 2) return null
  const toGroups = (s: string): string[] | null => {
    if (!s) return []
    const groups = s.split(':')
    if (groups.some((g) => !/^[0-9a-f]{1,4}$/.test(g))) return null
    return groups
  }
  const left = toGroups(halves[0])
  const right = halves.length === 2 ? toGroups(halves[1]) : []
  if (left === null || right === null) return null

  let groups: string[]
  if (halves.length === 2) {
    const missing = 8 - left.length - right.length
    if (missing < 1) return null
    groups = [...left, ...Array.from({ length: missing }, () => '0'), ...right]
  } else {
    groups = left
  }
  if (groups.length !== 8) return null

  let value = 0n
  for (const g of groups) value = (value << 16n) + BigInt(parseInt(g, 16))
  return value
}

/** Parses an IPv4/IPv6 address. Returns null for invalid input. */
export function parseIP(raw: string): ParsedIP | null {
  let addr = String(raw ?? '').trim()
  if (!addr) return null
  if (addr.startsWith('[') && addr.endsWith(']')) addr = addr.slice(1, -1)
  addr = addr.split('%')[0] // drop IPv6 zone
  if (addr.includes(':')) {
    const mapped = addr.match(/^::ffff:(\d+\.\d+\.\d+\.\d+)$/i)
    if (mapped) return parseIP(mapped[1])
    const value = ipv6ToBigInt(addr)
    return value === null ? null : { family: 6, value }
  }
  const value = ipv4ToBigInt(addr)
  return value === null ? null : { family: 4, value }
}

/** Parses an IP or CIDR (single IPs get a full-length prefix). */
export function parseCIDR(raw: string): ParsedCIDR | null {
  const s = String(raw ?? '').trim()
  if (!s) return null
  if (!s.includes('/')) {
    const ip = parseIP(s)
    return ip ? { ...ip, prefix: ip.family === 4 ? 32 : 128 } : null
  }
  const [base, prefixRaw] = s.split('/')
  const ip = parseIP(base)
  if (!ip) return null
  const bits = ip.family === 4 ? 32 : 128
  if (!/^\d{1,3}$/.test(prefixRaw)) return null
  const prefix = Number(prefixRaw)
  if (prefix < 0 || prefix > bits) return null
  return { ...ip, prefix }
}

/** Whether value is a valid IP or CIDR. */
export function isValidIPOrCIDR(value: string): boolean {
  return parseCIDR(value) !== null
}

/** Whether every entry is a valid IP / CIDR. */
export function ipAllowlistEntriesValid(values: string[]): boolean {
  return values.every((v) => isValidIPOrCIDR(v))
}

/** Whether ip is contained in cidr (same address family only). */
export function cidrContains(cidr: string, ip: string): boolean {
  const net = parseCIDR(cidr)
  const addr = parseIP(ip)
  if (!net || !addr || net.family !== addr.family) return false
  const bits = net.family === 4 ? 32 : 128
  if (net.prefix === 0) return true
  const shift = BigInt(bits - net.prefix)
  return addr.value >> shift === net.value >> shift
}

/** Removes an optional host:port / [v6]:port wrapper and returns the IP part. */
export function stripPort(raw: string): string {
  const s = String(raw ?? '').trim()
  if (!s) return ''
  if (s.startsWith('[')) {
    const end = s.indexOf(']')
    if (end > 0) return s.slice(1, end)
  }
  const colon = s.lastIndexOf(':')
  if (colon > -1 && s.indexOf(':') === colon) return s.slice(0, colon)
  return s
}

export type ClientIPResolution = { ip: string; source: string }

/**
 * Resolves the client IP the same way the backend WAF does:
 * - no trusted proxies -> TCP peer only;
 * - peer not in trusted proxies -> ignore X-Forwarded-For;
 * - peer trusted -> rightmost XFF hop that is not itself a trusted proxy.
 */
export function resolveClientIP(
  remote: string,
  xff: string,
  trustedProxies: string[],
): ClientIPResolution {
  const peer = stripPort(remote)
  const trusted = trustedProxies.map((s) => s.trim()).filter(Boolean)
  if (trusted.length === 0) {
    return { ip: peer, source: 'RemoteAddr（未配置受信代理）' }
  }
  if (!parseIP(peer) || !trusted.some((c) => cidrContains(c, peer))) {
    return { ip: peer, source: 'RemoteAddr（来源不在受信代理，X-Forwarded-For 已忽略）' }
  }
  const parts = String(xff ?? '')
    .split(',')
    .map((s) => stripPort(s.trim().replace(/^"|"$/g, '')))
    .filter(Boolean)
  for (let i = parts.length - 1; i >= 0; i--) {
    const ip = parts[i]
    if (!parseIP(ip)) continue
    if (trusted.some((c) => cidrContains(c, ip))) continue
    return { ip, source: 'X-Forwarded-For 右起第一个非受信地址' }
  }
  return { ip: peer, source: 'RemoteAddr（X-Forwarded-For 全为受信代理或为空）' }
}

export type AllowlistDecision = { allowed: boolean; reason: string }

/** deny first, then allow gate; empty allow means unrestricted. */
export function evaluateAllowlist(allow: string[], deny: string[], clientIP: string): AllowlistDecision {
  if (!clientIP) return { allowed: false, reason: '无法解析客户端 IP' }
  if (deny.some((c) => cidrContains(c, clientIP))) return { allowed: false, reason: '命中 deny' }
  if (allow.length === 0) return { allowed: true, reason: '未配置 allow（不限制）' }
  if (allow.some((c) => cidrContains(c, clientIP))) return { allowed: true, reason: '命中 allow' }
  return { allowed: false, reason: '未命中 allow' }
}

/** Trusted proxies configured in the global waf module (doc.waf.trusted_proxies). */
export function trustedProxiesFromDoc(doc: Record<string, unknown>): string[] {
  return arr<string>(obj(doc.waf).trusted_proxies)
    .map((s) => String(s).trim())
    .filter(Boolean)
}

export type IPAllowlistYAMLInput = {
  host: string
  enabled: boolean
  allow: string[]
  deny: string[]
  signatures: boolean
}

/** YAML fragment preview for the route editor (rules[].waf). */
export function buildIPAllowlistYAML(input: IPAllowlistYAMLInput): string {
  const allow = input.allow.map((s) => s.trim()).filter(Boolean)
  const deny = input.deny.map((s) => s.trim()).filter(Boolean)
  if (!input.enabled || (allow.length === 0 && deny.length === 0)) {
    return `# ${input.host}\n# 未配置 IP 白名单 → 不限制来源（默认行为）`
  }
  const lines = ['rules:', `  - host: '${input.host}'`, '    waf:', '      enabled: true']
  if (!input.signatures) {
    lines.push('      disable_builtin: true   # 仅 IP 准入，不运行内置攻击签名')
  }
  if (allow.length > 0) {
    lines.push('      allow:')
    allow.forEach((v) => lines.push(`        - '${v}'`))
  }
  if (deny.length > 0) {
    lines.push('      deny:')
    deny.forEach((v) => lines.push(`        - '${v}'`))
  }
  lines.push('      block_status_code: 403')
  return lines.join('\n')
}
