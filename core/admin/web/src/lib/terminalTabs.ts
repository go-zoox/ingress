export type ConnState = 'idle' | 'connecting' | 'open' | 'closed' | 'reconnecting' | 'error'

export type TerminalTabRecord = {
  tabId: string
  sessionId: string | null
  title: string
}

export type TerminalTabsState = {
  activeTabId: string
  nextNumber: number
  tabs: TerminalTabRecord[]
}

const TABS_STORAGE_KEY = 'ingress_admin_terminal_tabs_v2'
const LEGACY_SESSION_KEY = 'ingress_admin_terminal_session'
export const TERMINAL_MAX_TABS = 8

export function newTabId() {
  return crypto.randomUUID()
}

export function defaultTabsState(): TerminalTabsState {
  const tabId = newTabId()
  return {
    activeTabId: tabId,
    nextNumber: 2,
    tabs: [{ tabId, sessionId: null, title: '终端 1' }],
  }
}

export function loadTabsState(): TerminalTabsState {
  try {
    const raw = sessionStorage.getItem(TABS_STORAGE_KEY)
    if (raw) {
      const parsed = JSON.parse(raw) as TerminalTabsState
      if (parsed?.tabs?.length && parsed.activeTabId) {
        return parsed
      }
    }
    const legacy = sessionStorage.getItem(LEGACY_SESSION_KEY)
    if (legacy) {
      const tabId = newTabId()
      sessionStorage.removeItem(LEGACY_SESSION_KEY)
      return {
        activeTabId: tabId,
        nextNumber: 2,
        tabs: [{ tabId, sessionId: legacy, title: '终端 1' }],
      }
    }
  } catch {
    /* ignore */
  }
  return defaultTabsState()
}

export function saveTabsState(state: TerminalTabsState) {
  try {
    sessionStorage.setItem(TABS_STORAGE_KEY, JSON.stringify(state))
  } catch {
    /* ignore */
  }
}

export function connStateLabel(state: ConnState) {
  switch (state) {
    case 'open':
      return '已连接'
    case 'connecting':
      return '连接中…'
    case 'reconnecting':
      return '重连中…'
    case 'error':
      return '连接失败'
    case 'closed':
      return '已断开'
    default:
      return '未连接'
  }
}
