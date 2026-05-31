export type UIPreferences = {
  logLiveIntervalMs: number
  /** @deprecated use overviewRange */
  metricsWindow: string
  overviewRange?: string
}

export const DEFAULT_PREFERENCES: UIPreferences = {
  logLiveIntervalMs: 2000,
  metricsWindow: 'live',
}

/** Session-only UI preferences (not persisted across page reloads). */
let sessionPreferences: UIPreferences = { ...DEFAULT_PREFERENCES }

export function loadPreferences(): UIPreferences {
  return { ...sessionPreferences }
}

export function savePreferences(prefs: UIPreferences) {
  sessionPreferences = { ...prefs }
}

export function displayPath(path: string) {
  const full = path.trim()
  if (!full) return { display: '—', full: '' }
  const normalized = full.replace(/\\/g, '/')
  const adminMark = '/examples/admin-console/'
  const adminIdx = normalized.lastIndexOf(adminMark)
  if (adminIdx >= 0) {
    return { display: `./${normalized.slice(adminIdx + adminMark.length)}`, full }
  }
  const parts = normalized.split('/').filter(Boolean)
  if (parts.length > 3) {
    return { display: `…/${parts.slice(-3).join('/')}`, full }
  }
  return { display: full, full }
}
