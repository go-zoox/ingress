/** Parse route-detail scope from URL search params or pasted URL. */

export type RouteScope = {
  host: string
  path: string
  pathMatch: 'prefix' | 'exact'
}

export function parseRouteScopeFromSearchParams(sp: URLSearchParams): RouteScope {
  let host = (sp.get('host') || '').trim()
  let path = (sp.get('path') || '').trim()
  const urlParam = (sp.get('url') || '').trim()
  const pathMatchRaw = (sp.get('path_match') || 'prefix').trim().toLowerCase()
  const pathMatch: RouteScope['pathMatch'] = pathMatchRaw === 'exact' ? 'exact' : 'prefix'

  const tryParseUrl = (v: string) => {
    try {
      const u = new URL(v)
      return { host: u.host, path: u.pathname || '/' }
    } catch {
      return null
    }
  }

  if (urlParam) {
    const parsed = tryParseUrl(urlParam)
    if (parsed) {
      if (!host) host = parsed.host
      if (!path) path = parsed.path
    }
  }

  if (path && (path.startsWith('http://') || path.startsWith('https://'))) {
    const parsed = tryParseUrl(path)
    if (parsed) {
      if (!host) host = parsed.host
      path = parsed.path
    }
  }

  return { host, path, pathMatch }
}

/** Accept host/path fields or a full URL pasted into either field. */
export function normalizeScopeInput(hostInput: string, pathInput: string): { host: string; path: string } {
  const host = hostInput.trim()
  let path = pathInput.trim()

  const tryParse = (v: string) => {
    try {
      const u = new URL(v)
      return { host: u.host, path: u.pathname || '/' }
    } catch {
      return null
    }
  }

  if (path && (path.startsWith('http://') || path.startsWith('https://'))) {
    const p = tryParse(path)
    if (p) return { host: p.host || host, path: p.path }
  }
  if (!host && path.startsWith('http')) {
    const p = tryParse(path)
    if (p) return { host: p.host, path: p.path }
  }
  if (!path && host.startsWith('http')) {
    const p = tryParse(host)
    if (p) return { host: p.host, path: p.path }
  }

  return { host, path }
}

export function scopeSearchParams(scope: RouteScope): URLSearchParams {
  const q = new URLSearchParams()
  if (scope.host) q.set('host', scope.host)
  if (scope.path) q.set('path', scope.path)
  if (scope.pathMatch === 'exact') q.set('path_match', 'exact')
  return q
}

export function hostLooksLikePattern(host: string): boolean {
  const h = host.trim()
  if (!h) return false
  return /[*^$()[\]|\\?+]/.test(h)
}
