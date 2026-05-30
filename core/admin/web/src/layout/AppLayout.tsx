import { useCallback, useEffect, useState } from 'react'
import { NavLink, Outlet, useLocation } from 'react-router-dom'
import { api } from '../api/client'
import { isNavBadgeKey, navIcon } from './navConfig'
import { useNavigation } from '../hooks/useNavigation'
import { useNavBadges } from '../hooks/useNavBadges'
import { useAuth } from '../context/AuthContext'
import { SettingsMenu } from '../components/SettingsMenu'
import { NotificationMenu } from '../components/NotificationMenu'
import { NotificationProvider } from '../context/NotificationContext'
import { useSSE } from '../hooks/useSSE'

export function AppLayout() {
  const [configPath, setConfigPath] = useState('—')
  const [version, setVersion] = useState('')
  const [reloadReady, setReloadReady] = useState(false)
  const [configHash, setConfigHash] = useState('')
  const [runtimeHash, setRuntimeHash] = useState('')
  const [latestHash, setLatestHash] = useState('')
  const [runtimeDrift, setRuntimeDrift] = useState(false)
  const [revisionDrift, setRevisionDrift] = useState(false)
  const [drawerOpen, setDrawerOpen] = useState(false)
  const location = useLocation()
  const badges = useNavBadges()
  const { config } = useAuth()
  const { groups: navGroups, loading: navLoading } = useNavigation()

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
        setVersion(String(s.version || ''))
        setReloadReady(Boolean(s.reload_ready))
        setConfigHash(String(s.file_hash || s.config_hash || ''))
        setRuntimeHash(String(s.runtime_hash || ''))
        setRuntimeDrift(Boolean(s.runtime_drift))
        setRevisionDrift(Boolean(s.revision_drift))
      })
      .catch(() => {
        setConfigPath('—')
        setVersion('')
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
  const { connected: sseConnected } = useSSE(['health'])

  return (
    <NotificationProvider runtimeDrift={runtimeDrift} revisionDrift={revisionDrift}>
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
            <div className="brand-top">
              <span className="brand-title">Ingress Console</span>
              {version ? (
                <span className="brand-version" title={`Ingress ${version}`}>
                  v{version.replace(/^v/i, '')}
                </span>
              ) : null}
            </div>
            <span>运维控制台</span>
          </div>
          <nav className="nav" aria-label="主导航">
            {navLoading ? (
              <p className="empty-hint nav-loading">加载菜单…</p>
            ) : navGroups.length === 0 ? (
              <p className="empty-hint nav-loading">
                当前账号没有可见菜单，请联系管理员分配「菜单」权限或角色。
                {config?.user?.username ? `（${config.user.username}）` : null}
              </p>
            ) : (
              navGroups.map((group) => (
                <div key={group.label} className="nav-group">
                  <div className="nav-group-label">{group.label}</div>
                  {group.items.map((item) => {
                    const Icon = navIcon(item.icon)
                    const badgeKey = isNavBadgeKey(item.badge_key) ? item.badge_key : undefined
                    const badge = badgeKey && badges[badgeKey] > 0 ? badges[badgeKey] : 0
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
                              badgeKey === 'events'
                                ? badges.healths > 0
                                  ? ' danger'
                                  : ' warn'
                                : badgeKey === 'healths'
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
              ))
            )}
          </nav>
        </aside>
        <main className="main">
          <div className="app-chrome">
            <NotificationMenu />
            <SettingsMenu
              configPath={configPath}
              version={version}
              reloadReady={reloadReady}
              configHash={configHash}
              runtimeHash={runtimeHash}
              latestHash={latestHash}
              runtimeDrift={runtimeDrift}
              revisionDrift={revisionDrift}
              sseConnected={sseConnected}
            />
          </div>
          <div className="main-body">
            <Outlet key={location.pathname} />
          </div>
        </main>
      </div>
    </NotificationProvider>
  )
}
