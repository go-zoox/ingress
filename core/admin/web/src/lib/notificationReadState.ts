const KEY = 'ingress-admin.notificationRead'

type ReadEntry = {
  fingerprint: string
  readAt: string
}

type ReadState = Record<string, ReadEntry>

function loadState(): ReadState {
  try {
    const raw = localStorage.getItem(KEY)
    if (!raw) return {}
    const parsed = JSON.parse(raw) as ReadState
    return parsed && typeof parsed === 'object' ? parsed : {}
  } catch {
    return {}
  }
}

function saveState(state: ReadState) {
  localStorage.setItem(KEY, JSON.stringify(state))
}

export function notificationFingerprint(id: string, detail: string) {
  return `${id}:${detail}`
}

export function isNotificationRead(id: string, fingerprint: string) {
  const entry = loadState()[id]
  return entry?.fingerprint === fingerprint
}

export function markNotificationRead(id: string, fingerprint: string) {
  const state = loadState()
  state[id] = { fingerprint, readAt: new Date().toISOString() }
  saveState(state)
}

export function getNotificationReadAt(id: string, fingerprint: string) {
  const entry = loadState()[id]
  return entry?.fingerprint === fingerprint ? entry.readAt : undefined
}
