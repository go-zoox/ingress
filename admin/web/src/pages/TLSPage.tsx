import { useEffect, useRef, useState } from 'react'
import { PageHeader } from '../components/PageHeader'
import { TLSCheckModal } from '../components/TLSCheckModal'
import { ToastContainer, useToast } from '../components/Toast'
import { api, type TLSCert, type TLSCertCheck } from '../api/client'

function statusBadge(status: string) {
  if (status === 'ok') return <span className="badge badge-exact">正常</span>
  if (status === 'warn') return <span className="badge badge-wildcard">即将过期</span>
  if (status === 'expired' || status === 'missing')
    return <span className="badge badge-block">{status === 'missing' ? '缺失' : '已过期'}</span>
  return <span>{status}</span>
}

/** Show a short cert path in the table; keep full path in title. */
function displayCertPath(path: string) {
  const full = path.trim()
  if (!full) return { display: '—', full: '' }
  const normalized = full.replace(/\\/g, '/')
  const certsMark = '/certs/'
  const certsIdx = normalized.lastIndexOf(certsMark)
  if (certsIdx >= 0) {
    return { display: `./${normalized.slice(certsIdx + 1)}`, full }
  }
  const adminMark = '/examples/admin-console/'
  const adminIdx = normalized.lastIndexOf(adminMark)
  if (adminIdx >= 0) {
    return { display: `./${normalized.slice(adminIdx + adminMark.length)}`, full }
  }
  const parts = normalized.split('/').filter(Boolean)
  if (parts.length > 2) {
    return { display: `…/${parts.slice(-2).join('/')}`, full }
  }
  return { display: full, full }
}

async function copyText(text: string) {
  if (!text) throw new Error('路径为空')
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(text)
    return
  }
  const ta = document.createElement('textarea')
  ta.value = text
  ta.style.position = 'fixed'
  ta.style.opacity = '0'
  document.body.appendChild(ta)
  ta.select()
  document.execCommand('copy')
  document.body.removeChild(ta)
}

function CertRowActions({
  cert,
  checking,
  onCheck,
  onCopyPath,
}: {
  cert: TLSCert
  checking: boolean
  onCheck: () => void
  onCopyPath: (path: string, label: string) => void
}) {
  const [menuOpen, setMenuOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!menuOpen) return
    const close = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false)
      }
    }
    document.addEventListener('mousedown', close)
    return () => document.removeEventListener('mousedown', close)
  }, [menuOpen])

  return (
    <div className="row-actions">
      <button
        type="button"
        className="action-link"
        disabled={checking}
        onClick={onCheck}
      >
        {checking ? '检测中…' : '检测'}
      </button>
      <div className="action-menu" ref={menuRef}>
        <button
          type="button"
          className="action-link action-advanced"
          aria-expanded={menuOpen}
          onClick={() => setMenuOpen((v) => !v)}
        >
          高级
        </button>
        {menuOpen && (
          <div className="action-menu-panel" role="menu">
            <button
              type="button"
              role="menuitem"
              className="action-menu-item"
              onClick={() => {
                setMenuOpen(false)
                onCopyPath(cert.certificate, '证书路径')
              }}
            >
              复制证书路径
            </button>
            <button
              type="button"
              role="menuitem"
              className="action-menu-item"
              onClick={() => {
                setMenuOpen(false)
                onCopyPath(cert.certificate_key, '私钥路径')
              }}
            >
              复制私钥路径
            </button>
          </div>
        )}
      </div>
    </div>
  )
}

export function TLSPage() {
  const [certs, setCerts] = useState<TLSCert[]>([])
  const [status, setStatus] = useState<Record<string, unknown> | null>(null)
  const [err, setErr] = useState('')
  const [checkingDomain, setCheckingDomain] = useState('')
  const [checkOpen, setCheckOpen] = useState(false)
  const [checkLoading, setCheckLoading] = useState(false)
  const [checkResult, setCheckResult] = useState<TLSCertCheck | null>(null)
  const { toast, show, clear } = useToast()

  useEffect(() => {
    api.status().then(setStatus).catch(() => setStatus(null))
    api
      .tlsCerts()
      .then((data) => setCerts(Array.isArray(data) ? data : []))
      .catch((e: Error) => setErr(e.message))
  }, [])

  const warnCount = certs.filter((c) => c.status === 'warn' || c.status === 'expired').length

  const runCheck = async (cert: TLSCert) => {
    setCheckingDomain(cert.domain)
    setCheckOpen(true)
    setCheckLoading(true)
    setCheckResult(null)
    try {
      const result = await api.tlsCheck(cert.domain)
      setCheckResult(result)
      setCerts((prev) =>
        prev.map((c) =>
          c.domain === cert.domain
            ? {
                ...c,
                issuer: result.issuer,
                expires_at: result.expires_at,
                days_remaining: result.days_remaining,
                status: result.status,
              }
            : c,
        ),
      )
    } catch (e) {
      show(e instanceof Error ? e.message : '检测失败', 'error')
      setCheckOpen(false)
    } finally {
      setCheckLoading(false)
      setCheckingDomain('')
    }
  }

  const copyPath = async (path: string, label: string) => {
    try {
      await copyText(path)
      show(`${label}已复制`)
    } catch (e) {
      show(e instanceof Error ? e.message : '复制失败', 'error')
    }
  }

  return (
    <div className="page">
      <PageHeader title="TLS / 证书" desc="HTTPS 监听与证书有效期（读取 ingress.yaml + 证书文件）" />
      {err && <p className="err">{err}</p>}
      <div className="cards">
        <div className="card">
          <div className="label">HTTPS 端口</div>
          <div className="value">{String(status?.listen_https || '—')}</div>
          <div className="sub">https.port</div>
        </div>
        <div className="card">
          <div className="label">证书数量</div>
          <div className="value">{certs.length}</div>
          <div className="sub">https.ssl</div>
        </div>
        <div className={`card ${warnCount ? 'warn' : 'ok'}`}>
          <div className="label">需关注</div>
          <div className="value">{warnCount || '0'}</div>
          <div className="sub">即将过期 / 已过期</div>
        </div>
      </div>
      <div className="panel">
        <div className="panel-head">
          <h2>证书</h2>
        </div>
        <div className="panel-body panel-table-wrap">
          <table className="data tls-certs">
            <colgroup>
              <col className="col-domain" />
              <col className="col-issuer" />
              <col className="col-expires" />
              <col className="col-days" />
              <col className="col-status" />
              <col className="col-path" />
              <col className="col-actions" />
            </colgroup>
            <thead>
              <tr>
                <th>域名</th>
                <th>签发者</th>
                <th>到期日</th>
                <th>剩余天数</th>
                <th>状态</th>
                <th>证书路径</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              {certs.length === 0 ? (
                <tr>
                  <td colSpan={7} className="empty-hint">
                    未配置 HTTPS 或 https.ssl 为空
                  </td>
                </tr>
              ) : (
                certs.map((c) => {
                  const certPath = displayCertPath(c.certificate)
                  return (
                    <tr key={c.domain}>
                      <td className="tls-domain">{c.domain}</td>
                      <td className="tls-issuer" title={c.issuer || undefined}>
                        {c.issuer || '—'}
                      </td>
                      <td className="tls-expires">{c.expires_at || '—'}</td>
                      <td className="tls-days">{c.days_remaining ?? '—'}</td>
                      <td className="tls-status">{statusBadge(c.status)}</td>
                      <td className="tls-path">
                        <code className="path-cell" title={certPath.full || undefined}>
                          {certPath.display}
                        </code>
                      </td>
                      <td className="tls-actions">
                        <CertRowActions
                          cert={c}
                          checking={checkingDomain === c.domain}
                          onCheck={() => runCheck(c)}
                          onCopyPath={copyPath}
                        />
                      </td>
                    </tr>
                  )
                })
              )}
            </tbody>
          </table>
        </div>
      </div>
      <TLSCheckModal
        open={checkOpen}
        result={checkResult}
        loading={checkLoading}
        onClose={() => setCheckOpen(false)}
      />
      {toast && <ToastContainer message={toast.message} type={toast.type} onDone={clear} />}
    </div>
  )
}
