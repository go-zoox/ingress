import { useEffect, useState } from 'react'
import { NavLink, Outlet } from 'react-router-dom'
import { api } from '../api/client'

const nav = [
  { to: '/', label: '总览', icon: '◉', end: true },
  { to: '/routes', label: '路由', icon: '⇄' },
  { to: '/waf', label: 'WAF', icon: '⛨' },
  { to: '/tls', label: 'TLS', icon: '🔒' },
  { to: '/config', label: '配置', icon: '⌘' },
  { to: '/logs', label: '日志', icon: '≡' },
]

export function AppLayout() {
  const [configPath, setConfigPath] = useState('—')

  useEffect(() => {
    api
      .status()
      .then((s) => setConfigPath(String(s.config_path || '—')))
      .catch(() => setConfigPath('—'))
  }, [])

  return (
    <>
      <div className="proto-banner">
        <span>
          <strong>产品原型</strong> · 单机部署 · 配置落回 YAML · 连接真实 API
        </span>
        <span className="tag">ingress admin</span>
      </div>
      <div className="layout">
        <aside className="sidebar">
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
