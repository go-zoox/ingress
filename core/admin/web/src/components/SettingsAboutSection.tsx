import { useLocation } from 'react-router-dom'
import { pageAboutForPath } from '../lib/pageAbout'

type Props = {
  version?: string
}

export function SettingsAboutSection({ version }: Props) {
  const { pathname } = useLocation()
  const page = pageAboutForPath(pathname)

  return (
    <div className="settings-menu-about">
      <div className="settings-menu-about-head">关于</div>
      <div className="settings-menu-about-product">
        <div className="settings-menu-about-title">Ingress Console</div>
        <div className="settings-menu-about-sub">运维控制台</div>
      </div>
      {page ? (
        <div className="settings-menu-about-page">
          <div className="settings-menu-about-page-label">当前页面 · {page.title}</div>
          <p className="settings-menu-about-page-desc">{page.desc}</p>
        </div>
      ) : null}
      <div className="settings-menu-about-meta">
        <span className="settings-menu-about-meta-label">Ingress 版本</span>
        <span>{version?.trim() ? version : '—'}</span>
      </div>
    </div>
  )
}
