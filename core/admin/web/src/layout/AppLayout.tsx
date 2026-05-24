import { useCallback, useEffect, useState } from 'react'
import { NavLink, Outlet, useLocation } from 'react-router-dom'
import { api } from '../api/client'

const nav = [
  { to: '/', label: '总览', icon: '◉', end: true },
  { to: '/routes', label: '路由', icon: '⇄' },
  { to: '/cache', label: '缓存', icon: '◫' },
  { to: '/waf', label: 'WAF', icon: '⛨' },
  { to: '/tls', label: 'TLS', icon: '🔒' },
  { to: '/logs', label: '日志', icon: '≡' },
  { to: '/config', label: '配置', icon: '⌘' },
  { to: '/settings', label: '设置', icon: '⚙' },
]

export function AppLayout() {
  const [configPath, setConfigPath] = useState('—')
  const [drawerOpen, setDrawerOpen] = useState(false)
  const location = useLocation()

  // Close drawer on route change
  useEffect(() => {
    setDrawerOpen(false)
  }, [location.pathname])

  // Close drawer on Escape key
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setDrawerOpen(false)
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [])

  // Lock body scroll when drawer is open
  useEffect(() => {
    document.body.style.overflow = drawerOpen ? 'hidden' : ''
    return () => { document.body.style.overflow = '' }
  }, [drawerOpen])

  useEffect(() => {
    api
      .status()
      .then((s) => setConfigPath(String(s.config_path || '—')))
      .catch(() => setConfigPath('—'))
  }, [])

  const close = useCallback(() => setDrawerOpen(false), [])

  return (
    <>
      <div className="proto-banner">
        <button
          className="mobile-hamburger"
          aria-label="Toggle navigation"
          aria-expanded={drawerOpen}
          onClick={() => setDrawerOpen((v) => !v)}
        >
          <span className="hamburger-line" />
          <span className="hamburger-line" />
          <span className="hamburger-line" />
        </button>
        <span className="proto-text">
          <strong>产品原型</strong> · 单机部署 · 配置落回 YAML · 连接真实 API
        </span>
        <span className="tag">ingress admin</span>
      </div>

      {/* Backdrop */}
      {drawerOpen && (
        <div className="sidebar-backdrop" onClick={close} />
      )}

      <div className="layout">
        <aside className={`sidebar${drawerOpen ? ' open' : ''}`}>
          <div className="brand">
            Ingress Console
            <span>单机运维管理</span>
          </div>
          <nav className="nav">
            {nav.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                end={item.end}
                className={({ isActive }) => (isActive ? 'active' : '')}
                onClick={close}
              >
                <span className="icon">{item.icon}</span>
                {item.label}
              </NavLink>
            ))}
          </nav>
          <div className="sidebar-footer">
            配置路径
            <br />
            <code>{configPath}</code>
          </div>
        </aside>
        <main className="main">
          <Outlet />
        </main>
      </div>
    </>
  )
}
