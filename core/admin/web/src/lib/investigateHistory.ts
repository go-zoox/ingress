const KEY = 'ingress-admin.investigateHistory'
const MAX = 12

export type InvestigateHistoryEntry = {
  host: string
  path: string
  method?: string
  ts: number
}

export function loadInvestigateHistory(): InvestigateHistoryEntry[] {
  try {
    const raw = localStorage.getItem(KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw) as InvestigateHistoryEntry[]
    if (!Array.isArray(parsed)) return []
    return parsed.filter((e) => e?.host && e?.path)
  } catch {
    return []
  }
}

export function pushInvestigateHistory(entry: Omit<InvestigateHistoryEntry, 'ts'>) {
  const host = entry.host.trim()
  const path = entry.path.trim() || '/'
  if (!host) return

  const row: InvestigateHistoryEntry = {
    host,
    path,
    method: entry.method?.trim() || undefined,
    ts: Date.now(),
  }

  const prev = loadInvestigateHistory().filter(
    (e) => !(e.host === row.host && e.path === row.path && e.method === row.method),
  )
  const next = [row, ...prev].slice(0, MAX)
  localStorage.setItem(KEY, JSON.stringify(next))
}

export function clearInvestigateHistory() {
  localStorage.removeItem(KEY)
}
