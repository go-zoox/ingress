import { useEffect, useState } from 'react'
import { PageHeader } from '../components/PageHeader'
import { api, type TLSCert } from '../api/client'

function statusBadge(status: string) {
  if (status === 'ok') return <span className="badge badge-exact">正常</span>
  if (status === 'warn') return <span className="badge badge-wildcard">即将过期</span>
  if (status === 'expired' || status === 'missing')
    return <span className="badge badge-block">已过期</span>
  return <span>{status}</span>
}

export function TLSPage() {
  const [certs, setCerts] = useState<TLSCert[]>([])
  const [err, setErr] = useState('')

  useEffect(() => {
    api
      .tlsCerts()
      .then((data) => setCerts(Array.isArray(data) ? data : []))
      .catch((e: Error) => setErr(e.message))
  }, [])

  return (
    <div className="page">
      <PageHeader title="TLS / 证书" desc="HTTPS 监听与证书有效期" />
      {err && <p className="err">{err}</p>}
      <div className="panel">
        <div className="panel-head">
          <h2>证书</h2>
        </div>
        <div className="panel-body panel-table-wrap">
          <table className="data">
            <thead>
              <tr>
                <th>域名</th>
                <th>签发者</th>
                <th>到期日</th>
                <th>剩余天数</th>
                <th>状态</th>
              </tr>
            </thead>
            <tbody>
              {certs.length === 0 ? (
                <tr>
                  <td colSpan={5} className="empty-hint">
                    未配置 HTTPS 或 https.ssl 为空
                  </td>
                </tr>
              ) : (
                certs.map((c) => (
                  <tr key={c.domain}>
                    <td>{c.domain}</td>
                    <td>—</td>
                    <td>—</td>
                    <td>—</td>
                    <td>{statusBadge(c.status)}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
