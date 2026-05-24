const PREFIX = 'ingress-admin.'

export type UIPreferences = {
  logLiveIntervalMs: number
  metricsWindow: string
  metricsRefreshMs: number
}

export const DEFAULT_PREFERENCES: UIPreferences = {
  logLiveIntervalMs: 2000,
  metricsWindow: '15m',
  metricsRefreshMs: 5000,
}

const LOG_INTERVAL_KEY = `${PREFIX}logLiveIntervalMs`
const METRICS_WINDOW_KEY = `${PREFIX}metricsWindow`
const METRICS_REFRESH_KEY = `${PREFIX}metricsRefreshMs`

export function loadPreferences(): UIPreferences {
  const logLiveIntervalMs = readInt(LOG_INTERVAL_KEY, DEFAULT_PREFERENCES.logLiveIntervalMs)
  const metricsWindow =
    localStorage.getItem(METRICS_WINDOW_KEY)?.trim() || DEFAULT_PREFERENCES.metricsWindow
  const metricsRefreshMs = readInt(METRICS_REFRESH_KEY, DEFAULT_PREFERENCES.metricsRefreshMs)
  return { logLiveIntervalMs, metricsWindow, metricsRefreshMs }
}

export function savePreferences(prefs: UIPreferences) {
  localStorage.setItem(LOG_INTERVAL_KEY, String(prefs.logLiveIntervalMs))
  localStorage.setItem(METRICS_WINDOW_KEY, prefs.metricsWindow)
  localStorage.setItem(METRICS_REFRESH_KEY, String(prefs.metricsRefreshMs))
}

function readInt(key: string, fallback: number) {
  const raw = localStorage.getItem(key)
  if (!raw) return fallback
  const n = Number.parseInt(raw, 10)
  return Number.isFinite(n) && n >= 0 ? n : fallback
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
