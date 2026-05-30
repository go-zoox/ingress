export type GeoIPStatusLike = {
  configured?: boolean
  enabled?: boolean
  loaded?: boolean
  source?: string
  database?: string
  error?: string
  reason?: string
}

function isConfigured(g: GeoIPStatusLike): boolean {
  if (g.configured) return true
  return Boolean(g.database?.trim())
}

/** User-facing GeoIP mode line for WAF map / settings. */
export function geoipStatusLabel(g: GeoIPStatusLike | null | undefined): string {
  if (!g) return 'GeoIP：未启用（近似定位）'
  if (g.loaded && g.source === 'maxmind') return 'GeoIP：MaxMind GeoLite2 已启用'
  if (!isConfigured(g)) return 'GeoIP：未启用（未配置数据库，近似定位）'
  switch (g.reason) {
    case 'not_found':
      return 'GeoIP：未启用（数据库文件不存在，近似定位）'
    case 'permission_denied':
      return 'GeoIP：未启用（无读取权限，近似定位）'
    case 'invalid':
      return 'GeoIP：未启用（路径无效，近似定位）'
    case 'open_failed':
      return 'GeoIP：未启用（无法打开数据库，近似定位）'
    default:
      return 'GeoIP：未启用（数据库不可用，近似定位）'
  }
}

/** Short runtime status for settings panel. */
export function geoipRuntimeLabel(g: GeoIPStatusLike | null | undefined): {
  tone: 'ok' | 'warn' | 'muted'
  text: string
} {
  if (!g) return { tone: 'muted', text: '未启用（近似定位）' }
  if (g.loaded && g.source === 'maxmind') {
    return { tone: 'ok', text: 'MaxMind 已启用' }
  }
  if (!isConfigured(g)) {
    return { tone: 'muted', text: '未配置（近似定位）' }
  }
  switch (g.reason) {
    case 'not_found':
      return { tone: 'warn', text: '未启用 · 文件不存在' }
    case 'permission_denied':
      return { tone: 'warn', text: '未启用 · 无读取权限' }
    case 'invalid':
      return { tone: 'warn', text: '未启用 · 路径无效' }
    case 'open_failed':
      return { tone: 'warn', text: '未启用 · 无法打开数据库' }
    default:
      return { tone: 'warn', text: '未启用 · 数据库不可用' }
  }
}

export function geoipPathHint(exists: boolean, readable: boolean): string | null {
  if (!exists) return '文件不存在'
  if (!readable) return '无读取权限'
  return null
}
