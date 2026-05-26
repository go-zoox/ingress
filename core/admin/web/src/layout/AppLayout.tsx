import { useCallback, useEffect, useState } from 'react'
import { NavLink, Outlet, useLocation } from 'react-router-dom'
import { api } from '../api/client'
import { navGroups } from './navConfig'
import { useNavBadges } from '../hooks/useNavBadges'
import { SidebarGlobalStatus } from '../components/SidebarGlobalStatus'
import { useSSE } from '../hooks/useSSE'

export function AppLayout() {
  const [configPath, setConfigPath] = useState('—')
  const [reloadReady, setReloadReady] = useState(false)
  const [configHash, setConfigHash] = useState('')
  const [runtimeHash, setRuntimeHash] = useState('')
  const [latestHash, setLatestHash] = useState('')
  const [runtimeDrift, setRuntimeDrift] = useState(false)
  const [revisionDrift, setRevisionDrift] = useState(false)
  const [drawerOpen, setDrawerOpen] = useState(false)
  const location = useLocation()
  const badges = useNavBadges()

  useEffect(() => {
    setDrawerOpen(false)
  }, [location.pathname])

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setDrawerOpen(false)
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [])

  useEffect(() => {
    document.body.style.overflow = drawerOpen ? 'hidden' : ''
    return () => {
      document.body.style.overflow = ''
    }
  }, [drawerOpen])

  const loadStatus = useCallback(() => {
    api
      .status()
      .then((s) => {
        setConfigPath(String(s.config_path || '—'))
        setReloadReady(Boolean(s.reload_ready))
        setConfigHash(String(s.file_hash || s.config_hash || ''))
        setRuntimeHash(String(s.runtime_hash || ''))
        setRuntimeDrift(Boolean(s.runtime_drift))
        setRevisionDrift(Boolean(s.revision_drift))
      })
      .catch(() => {
        setConfigPath('—')
        setReloadReady(false)
        setConfigHash('')
        setRuntimeHash('')
        setRuntimeDrift(false)
        setRevisionDrift(false)
      })
    api
      .configRevisions(1)
      .then((revs) => {
        setLatestHash(revs.length > 0 ? revs[0].hash : '')
      })
      .catch(() => setLatestHash(''))
  }, [])

  useEffect(() => {
    loadStatus()
    const timer = window.setInterval(loadStatus, 30_000)
    return () => window.clearInterval(timer)
  }, [loadStatus])

  const close = useCallback(() => setDrawerOpen(false), [])
  const { connected: sseConnected } = useSSE(['metrics'])

  return (
    <>
      <div className="mobile-topbar">
        <button
          className="mobile-hamburger"
          aria-label="打开导航"
          aria-expanded={drawerOpen}
          onClick={() => setDrawerOpen((v) => !v)}
        >
          <span className="hamburger-line" />
          <span className="hamburger-line" />
          <span className="hamburger-line" />
        </button>
        <span className="mobile-topbar-title">Ingress Console</span>
      </div>

      {drawerOpen ? <div className="sidebar-backdrop" onClick={close} /> : null}

      <div className="layout">
        <aside className={`sidebar${drawerOpen ? ' open' : ''}`}>
          <div className="brand">
            Ingress Console
            <span>运维控制台</span>
          </div>
          <nav className="nav" aria-label="主导航">
            {navGroups.map((group) => (
              <div key={group.label} className="nav-group">
                <div className="nav-group-label">{group.label}</div>
                {group.items.map((item) => {
                  const Icon = item.icon
                  const badge =
                    item.badgeKey && badges[item.badgeKey] > 0
                      ? badges[item.badgeKey]
                      : 0
                  return (
                    <NavLink
                      key={item.to}
                      to={item.to}
                      end={item.end}
                      className={({ isActive }) => (isActive ? 'active' : '')}
                      onClick={close}
                    >
                      <span className="icon" aria-hidden>
                        <Icon size={18} strokeWidth={1.75} />
                      </span>
                      <span className="nav-label">{item.label}</span>
                      {badge > 0 ? (
                        <span
                          className={`nav-badge${
                            item.badgeKey === 'healths' ||
                            (item.badgeKey === 'overview' && badges.healths > 0)
                              ? ' danger'
                              : ' warn'
                          }`}
                        >
                          {badge > 99 ? '99+' : badge}
                        </span>
                      ) : null}
                    </NavLink>
                  )
                })}
              </div>
            ))}
          </nav>
          <div className="sidebar-footer">
            <SidebarGlobalStatus
              reloadReady={reloadReady}
              configHash={configHash}
              runtimeHash={runtimeHash}
              latestHash={latestHash}
              runtimeDrift={runtimeDrift}
              revisionDrift={revisionDrift}
              sseConnected={sseConnected}
            />
            <div className="sidebar-config-path">
              配置路径
              <br />
              <code title={configPath}>{configPath}</code>
            </div>
          </div>
        </aside>
        <main className="main">
          <Outlet />
        </main>
      </div>
    </>
  )
}
