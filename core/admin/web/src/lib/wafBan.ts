/**
 * Runtime WAF ban list helpers.
 *
 * Bans are operational state, not ingress config: they live in the admin DB and
 * are pushed to the running core as an immutable snapshot. Nothing here touches
 * ingress.yaml, so nothing here triggers a config reload.
 */

export type BanKind = 'ip' | 'ip_cidr' | 'ua_contains' | 'ua_regex'

export type BanScope = 'global' | 'hosts'

/** One ban row as returned by the admin API. */
export type WAFBanRow = {
  id: number
  kind: BanKind
  value: string
  // Backend may return `null` for an empty host scope (Go nil []string), and
  // existing rows written before the backend fix can still deserialize that
  // way. Treat it as nullable at the boundary and normalize before reading.
  hosts: string[] | null
  scope: BanScope
  reason: string
  source: string
  actor: string
  hits: number
  enabled: boolean
  expires_at: string | null
  active: boolean
  created_at: string
}

/** Normalizes the nullable hosts field to a safe array. */
export function banHosts(row: Pick<WAFBanRow, 'hosts'>): string[] {
  return Array.isArray(row.hosts) ? row.hosts : []
}

/** Impact preview returned by POST /waf/bans/preview. */
export type WAFBanPreview = {
  kind: BanKind
  value: string
  normalized: string
  hits_24h: number
  hosts: number
  distinct_ua: number
  paths: number
  trusted_proxy: boolean
  in_allow_list: boolean
  warnings: string[]
  blocked: boolean
  blocked_reason?: string
}

export type BanCreateInput = {
  kind: BanKind
  value: string
  hosts?: string[]
  reason?: string
  ttl_seconds?: number
  expires_at?: string | null
  confirm_permanent?: boolean
}

export const BAN_KIND_LABELS: Record<BanKind, string> = {
  ip: 'IP',
  ip_cidr: 'CIDR',
  ua_contains: 'UA',
  ua_regex: 'UA 正则',
}

/** Preset TTLs offered in the ban drawer. */
export const BAN_TTL_PRESETS = [
  { value: '1h', label: '1 小时', seconds: 3600 },
  { value: '24h', label: '24 小时', seconds: 86400 },
  { value: '7d', label: '7 天', seconds: 604800 },
  { value: 'forever', label: '永久', seconds: 0 },
] as const

export type BanTtlPreset = (typeof BAN_TTL_PRESETS)[number]['value']

export const DEFAULT_BAN_TTL: BanTtlPreset = '24h'

/** Reasonable threshold before we warn about NAT / carrier-grade NAT. */
export const NAT_WARNING_UA_THRESHOLD = 20

/** Warn when a ban expires within this window. */
export const EXPIRING_SOON_MS = 2 * 60 * 60 * 1000

const IPV4_RE = /^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/
const IPV6_RE = /^[0-9a-fA-F:]+$/

function isValidIPv4(value: string): boolean {
  const m = IPV4_RE.exec(value)
  if (!m) return false
  for (let i = 1; i <= 4; i++) {
    const octet = Number(m[i])
    if (!Number.isInteger(octet) || octet < 0 || octet > 255) return false
  }
  return true
}

function isValidIPv6(value: string): boolean {
  if (!IPV6_RE.test(value)) return false
  if (!value.includes(':')) return false
  // Reject runaway colon counts; full validation happens server-side.
  return value.split(':').length <= 9
}

/** Returns a normalized IP string, or null when invalid. */
export function normalizeIP(raw: string): string | null {
  const value = raw.trim()
  if (!value) return null
  if (isValidIPv4(value)) return value
  // IPv4-mapped IPv6 form: strip the prefix so it compares equal to plain IPv4.
  const mapped = /^::ffff:(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})$/i.exec(value)
  if (mapped && isValidIPv4(mapped[1])) return mapped[1]
  if (isValidIPv6(value)) return value
  return null
}

/** Validates a CIDR block (IPv4 prefix 0-32; IPv6 is lenient client-side). */
export function isValidCIDR(raw: string): boolean {
  const value = raw.trim()
  const slash = value.lastIndexOf('/')
  if (slash <= 0) return false
  const addr = value.slice(0, slash)
  const bitsRaw = value.slice(slash + 1)
  if (!/^\d{1,3}$/.test(bitsRaw)) return false
  const bits = Number(bitsRaw)
  if (isValidIPv4(addr)) return bits >= 0 && bits <= 32
  if (isValidIPv6(addr)) return bits >= 0 && bits <= 128
  return false
}

/**
 * Translates the Go (RE2) regex constructs that JavaScript does not accept.
 *
 * The backend compiles UA regexes with Go's `regexp`, so operators naturally
 * write things like `(?i)(nikto|massql)` or Go named groups. JavaScript throws
 * "Invalid group" on those, which would make the form reject input the server
 * would happily accept. Only syntax is translated; semantics stay equivalent
 * for the constructs supported by both engines.
 */
export function goRegexToJS(pattern: string): { source: string; flags: string } {
  let source = pattern
  let flags = ''

  // Leading inline flag groups such as (?i), (?is), (?im) at the start, plus
  // any later standalone ones (Go allows (?i) mid-pattern to toggle for the rest).
  source = source.replace(/\(\?([imsU]+)\)/g, (_match, groupFlags: string) => {
    if (groupFlags.includes('i')) flags = flags.includes('i') ? flags : flags + 'i'
    // s (dot-all) and m (multiline) are accepted by JS as flags too.
    if (groupFlags.includes('s') && !flags.includes('s')) flags += 's'
    if (groupFlags.includes('m') && !flags.includes('m')) flags += 'm'
    return ''
  })

  // Go named groups (?P<name>...) use JS syntax (?<name>...).
  source = source.replace(/\(\?P<([A-Za-z0-9_]+)>/g, '(?<$1>')

  // Go supports a few POSIX classes JS lacks; map the common ones.
  source = source.replace(/\[\[:alpha:\]\]/g, '[A-Za-z]')
  source = source.replace(/\[\[:digit:\]\]/g, '[0-9]')
  source = source.replace(/\[\[:alnum:\]\]/g, '[A-Za-z0-9]')
  source = source.replace(/\[\[:space:\]\]/g, '\\s')

  return { source, flags }
}

/** Compiles a regex to test validity, returning an error message when bad. */
export function validateRegex(pattern: string): string | null {
  if (!pattern.trim()) return '正则不能为空'
  try {
    const { source, flags } = goRegexToJS(pattern)
    // eslint-disable-next-line no-new
    new RegExp(source, flags)
    return null
  } catch (e) {
    return e instanceof Error ? e.message : '正则可编译失败'
  }
}

export type BanValueValidation = { ok: true; normalized: string } | { ok: false; error: string }

/** Validates the value for a ban kind, mirroring the backend rules. */
export function validateBanValue(kind: BanKind, raw: string): BanValueValidation {
  const value = raw.trim()
  if (!value) return { ok: false, error: '请输入封禁目标' }

  switch (kind) {
    case 'ip': {
      const ip = normalizeIP(value)
      if (!ip) return { ok: false, error: '请输入合法的 IP 地址' }
      return { ok: true, normalized: ip }
    }
    case 'ip_cidr': {
      if (!isValidCIDR(value)) return { ok: false, error: '请输入合法的 CIDR 网段，如 203.0.113.0/24' }
      return { ok: true, normalized: value }
    }
    case 'ua_contains':
      return { ok: true, normalized: value }
    case 'ua_regex': {
      const err = validateRegex(value)
      if (err) return { ok: false, error: err }
      return { ok: true, normalized: value }
    }
    default:
      return { ok: false, error: '未知的封禁类型' }
  }
}

/** Maps the kind selector to a concrete kind (IP vs CIDR is inferred). */
export function resolveBanKind(mode: 'ip' | 'ua', raw: string, uaMode: 'contains' | 'regex'): BanKind {
  if (mode === 'ua') return uaMode === 'regex' ? 'ua_regex' : 'ua_contains'
  return raw.includes('/') ? 'ip_cidr' : 'ip'
}

/** Computes request fields for a preset TTL. */
export function buildBanPayload(input: {
  mode: 'ip' | 'ua'
  value: string
  uaMode: 'contains' | 'regex'
  ttl: BanTtlPreset
  reason: string
  hosts: string[]
  confirmPermanent?: boolean
}): BanCreateInput {
  const kind = resolveBanKind(input.mode, input.value, input.uaMode)
  const preset = BAN_TTL_PRESETS.find((p) => p.value === input.ttl) ?? BAN_TTL_PRESETS[1]
  const payload: BanCreateInput = {
    kind,
    value: input.value.trim(),
    reason: input.reason.trim(),
  }
  const hosts = Array.isArray(input.hosts) ? input.hosts : []
  if (hosts.length > 0) payload.hosts = hosts
  if (preset.seconds > 0) payload.ttl_seconds = preset.seconds
  if (preset.seconds === 0) payload.confirm_permanent = Boolean(input.confirmPermanent)
  return payload
}

/** Formats the remaining time before a ban expires. */
export function formatRemaining(expiresAt: string | null, now: Date = new Date()): string {
  if (!expiresAt) return '永久'
  const end = new Date(expiresAt).getTime()
  if (Number.isNaN(end)) return '—'
  const diff = end - now.getTime()
  if (diff <= 0) return '已过期'

  const minutes = Math.floor(diff / 60000)
  if (minutes < 60) return `剩余 ${minutes}m`
  const hours = Math.floor(minutes / 60)
  if (hours < 48) return `剩余 ${hours}h`
  const days = Math.floor(hours / 24)
  return `剩余 ${days}d`
}

/** True when a ban expires within EXPIRING_SOON_MS (null = never). */
export function isExpiringSoon(expiresAt: string | null, now: Date = new Date()): boolean {
  if (!expiresAt) return false
  const end = new Date(expiresAt).getTime()
  if (Number.isNaN(end)) return false
  const diff = end - now.getTime()
  return diff > 0 && diff <= EXPIRING_SOON_MS
}

/** Human summary of a ban's scope for list rows. */
export function describeScope(row: Pick<WAFBanRow, 'hosts'>): string {
  const hosts = banHosts(row)
  return hosts.length === 0 ? '全局' : hosts.join(', ')
}

/** Builds impact warnings shown above the confirm button. */
export function buildImpactWarnings(preview: WAFBanPreview | null): string[] {
  if (!preview) return []
  const out: string[] = []
  if (preview.blocked) {
    out.push(preview.blocked_reason || '该目标不允许封禁')
    return out
  }
  if (preview.trusted_proxy) {
    out.push('该地址是反向代理/负载均衡自身，封禁会导致整站不可访问')
  }
  if (
    (preview.kind === 'ip' || preview.kind === 'ip_cidr') &&
    preview.distinct_ua >= NAT_WARNING_UA_THRESHOLD
  ) {
    out.push(
      `该 IP 近 24h 出现 ${preview.distinct_ua} 个不同 UA，可能位于 NAT / 运营商出口之后，封禁可能误伤大量真实用户`,
    )
  }
  if (preview.kind === 'ua_contains' || preview.kind === 'ua_regex') {
    out.push('UA 是客户端可控字段，更换 UA 即可绕过，按 UA 封禁仅作止血手段')
  }
  if (preview.in_allow_list) {
    out.push('该目标在 allow 白名单中；运行时封禁优先于 allow，依然会生效')
  }
  return out
}

/** Filters rows client-side by keyword across value/reason/actor. */
export function filterBanRows(rows: WAFBanRow[], keyword: string, kind: 'all' | 'ip' | 'ua'): WAFBanRow[] {
  const kw = keyword.trim().toLowerCase()
  return rows.filter((row) => {
    if (kind === 'ip' && !(row.kind === 'ip' || row.kind === 'ip_cidr')) return false
    if (kind === 'ua' && !(row.kind === 'ua_contains' || row.kind === 'ua_regex')) return false
    if (!kw) return true
    return `${row.value} ${row.reason} ${row.actor}`.toLowerCase().includes(kw)
  })
}

/** Counts active bans split by family for the summary cards. */
export function summarizeBans(rows: WAFBanRow[]): { total: number; ip: number; ua: number; expiring: number } {
  const now = new Date()
  let ip = 0
  let ua = 0
  let expiring = 0
  for (const row of rows) {
    if (row.kind === 'ua_contains' || row.kind === 'ua_regex') ua++
    else ip++
    if (isExpiringSoon(row.expires_at, now)) expiring++
  }
  return { total: rows.length, ip, ua, expiring }
}

/** Extracts the error code prefix returned by the ban API. */
export function banErrorCode(message: string): string {
  const idx = message.indexOf(':')
  return idx > 0 ? message.slice(0, idx) : ''
}

/** Friendly copy for known ban API error codes. */
export function describeBanError(message: string): string {
  const code = banErrorCode(message)
  switch (code) {
    case 'trusted_proxy':
      return '该 IP 是受信代理（waf.trusted_proxies）自身，封禁会导致整站不可访问，已拒绝'
    case 'confirm_required':
      return '永久封禁需要二次确认'
    case 'invalid_value':
      return `封禁目标不合法：${message.slice(message.indexOf(':') + 1).trim()}`
    case 'invalid_kind':
      return '封禁类型不合法'
    case 'invalid_hosts':
      return '生效域名不合法'
    case 'duplicate':
      return '已存在相同的生效封禁'
    case 'not_found':
      return '封禁条目不存在或已被删除'
    default:
      return message
  }
}
