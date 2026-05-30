import { useCallback, useEffect, useRef, useState } from 'react'
import { Plus, X } from 'lucide-react'
import { TerminalTabPane } from './TerminalTabPane'
import {
  TERMINAL_MAX_TABS,
  connStateLabel,
  loadTabsState,
  newTabId,
  saveTabsState,
  type ConnState,
  type TerminalTabRecord,
  type TerminalTabsState,
} from '../../lib/terminalTabs'

function syncTabsState(state: TerminalTabsState) {
  saveTabsState(state)
  return state
}

export function WebTerminal() {
  const [tabsState, setTabsState] = useState<TerminalTabsState>(() => loadTabsState())
  const [connByTab, setConnByTab] = useState<Record<string, ConnState>>({})
  const [newTabIds, setNewTabIds] = useState<Record<string, boolean>>({})

  const persistRef = useRef(tabsState)
  persistRef.current = tabsState

  useEffect(() => {
    saveTabsState(tabsState)
  }, [tabsState])

  const activeTab = tabsState.tabs.find((t) => t.tabId === tabsState.activeTabId) ?? tabsState.tabs[0]
  const activeConn = activeTab ? (connByTab[activeTab.tabId] ?? 'idle') : 'idle'

  const updateTab = useCallback((tabId: string, patch: Partial<TerminalTabRecord>) => {
    setTabsState((prev) =>
      syncTabsState({
        ...prev,
        tabs: prev.tabs.map((t) => (t.tabId === tabId ? { ...t, ...patch } : t)),
      }),
    )
  }, [])

  const onSessionChange = useCallback(
    (tabId: string, sessionId: string) => {
      updateTab(tabId, { sessionId })
    },
    [updateTab],
  )

  const onConnStateChange = useCallback((tabId: string, state: ConnState) => {
    setConnByTab((prev) => (prev[tabId] === state ? prev : { ...prev, [tabId]: state }))
  }, [])

  const addTab = useCallback(() => {
    setTabsState((prev) => {
      if (prev.tabs.length >= TERMINAL_MAX_TABS) return prev
      const tabId = newTabId()
      setNewTabIds((ids) => ({ ...ids, [tabId]: true }))
      return syncTabsState({
        activeTabId: tabId,
        nextNumber: prev.nextNumber + 1,
        tabs: [...prev.tabs, { tabId, sessionId: null, title: `终端 ${prev.nextNumber}` }],
      })
    })
  }, [])

  const closeTab = useCallback((tabId: string) => {
    setTabsState((prev) => {
      if (prev.tabs.length <= 1) return prev
      const idx = prev.tabs.findIndex((t) => t.tabId === tabId)
      if (idx < 0) return prev
      const nextTabs = prev.tabs.filter((t) => t.tabId !== tabId)
      let nextActive = prev.activeTabId
      if (prev.activeTabId === tabId) {
        const neighbor = nextTabs[Math.min(idx, nextTabs.length - 1)]
        nextActive = neighbor?.tabId ?? nextTabs[0]?.tabId ?? ''
      }
      return syncTabsState({ ...prev, activeTabId: nextActive, tabs: nextTabs })
    })
    setConnByTab((prev) => {
      if (!(tabId in prev)) return prev
      const next = { ...prev }
      delete next[tabId]
      return next
    })
    setNewTabIds((prev) => {
      if (!(tabId in prev)) return prev
      const next = { ...prev }
      delete next[tabId]
      return next
    })
  }, [])

  return (
    <div className="web-terminal">
      <div className="web-terminal-tabs" role="tablist" aria-label="终端标签">
        {tabsState.tabs.map((tab) => {
          const conn = connByTab[tab.tabId] ?? 'idle'
          const isActive = tab.tabId === tabsState.activeTabId
          return (
            <div
              key={tab.tabId}
              className={`web-terminal-tab${isActive ? ' active' : ''}`}
              role="tab"
              aria-selected={isActive}
            >
              <button
                type="button"
                className="web-terminal-tab-main"
                onClick={() => setTabsState((prev) => syncTabsState({ ...prev, activeTabId: tab.tabId }))}
              >
                <span className={`web-terminal-tab-dot web-terminal-tab-dot--${conn}`} aria-hidden />
                <span className="web-terminal-tab-title">{tab.title}</span>
              </button>
              {tabsState.tabs.length > 1 ? (
                <button
                  type="button"
                  className="web-terminal-tab-close"
                  aria-label={`关闭 ${tab.title}`}
                  onClick={() => closeTab(tab.tabId)}
                >
                  <X size={12} />
                </button>
              ) : null}
            </div>
          )
        })}
        <button
          type="button"
          className="web-terminal-tab-add"
          aria-label="新建终端标签"
          disabled={tabsState.tabs.length >= TERMINAL_MAX_TABS}
          onClick={addTab}
        >
          <Plus size={14} />
        </button>
      </div>

      <div className="web-terminal-toolbar">
        <span className={`web-terminal-status web-terminal-status--${activeConn}`}>
          {activeTab ? `${activeTab.title} · ${connStateLabel(activeConn)}` : connStateLabel(activeConn)}
        </span>
        <span className="web-terminal-tab-count">
          {tabsState.tabs.length}/{TERMINAL_MAX_TABS} 个会话
        </span>
      </div>

      <div className="web-terminal-panels">
        {tabsState.tabs.map((tab) => (
          <TerminalTabPane
            key={tab.tabId}
            tabId={tab.tabId}
            sessionId={tab.sessionId}
            active={tab.tabId === tabsState.activeTabId}
            startNew={Boolean(newTabIds[tab.tabId])}
            onSessionChange={onSessionChange}
            onConnStateChange={onConnStateChange}
          />
        ))}
      </div>
    </div>
  )
}
