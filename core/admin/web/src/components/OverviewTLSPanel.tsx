import { memo } from 'react'
import { Link } from 'react-router-dom'
import { Lock, ShieldAlert } from 'lucide-react'
import type { TLSCert } from '../api/client'

type Props = {
  certs: TLSCert[]
}

export const OverviewTLSPanel = memo(function OverviewTLSPanel({ certs }: Props) {
  if (certs.length === 0) {
    return <p className="empty-hint">未加载 TLS 证书</p>
  }

  const sorted = [...certs].sort((a, b) => a.days_remaining - b.days_remaining)
  const maxDays = Math.max(90, ...sorted.map((c) => c.days_remaining))

  return (
    <>
      <ul className="tls-expiry-list">
        {sorted.slice(0, 8).map((c) => {
          const pct = Math.min(100, (c.days_remaining / maxDays) * 100)
          const tone =
            c.days_remaining < 7 ? 'danger' : c.days_remaining < 30 ? 'warn' : 'ok'
          return (
            <li key={c.domain} className="tls-expiry-row">
              <span className="tls-expiry-domain" title={c.domain}>
                {c.days_remaining < 30 ? (
                  <ShieldAlert size={14} className={`icon-${tone}`} />
                ) : (
                  <Lock size={14} className="icon-muted" />
                )}
                {c.domain}
              </span>
              <div className="bar-track tls-expiry-track">
                <div className={`bar-fill seg-${tone === 'ok' ? '2xx' : tone === 'warn' ? '4xx' : '5xx'}`} style={{ width: `${pct}%` }} />
              </div>
              <span className={`tls-expiry-days tls-${tone}`}>{c.days_remaining} 天</span>
            </li>
          )
        })}
      </ul>
      <Link to="/tls" className="btn btn-ghost btn-sm" style={{ marginTop: 8 }}>
        证书管理
      </Link>
    </>
  )
})
