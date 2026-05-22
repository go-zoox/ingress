import type { TLSCertCheck } from '../api/client'

function checkIcon(level: string) {
  if (level === 'ok') return '✓'
  if (level === 'warn') return '!'
  return '×'
}

export function TLSCheckModal({
  open,
  result,
  loading,
  onClose,
}: {
  open: boolean
  result: TLSCertCheck | null
  loading: boolean
  onClose: () => void
}) {
  if (!open) return null

  return (
    <div className="modal-overlay open" onClick={(e) => e.target === e.currentTarget && onClose()}>
      <div className="modal tls-check-modal" role="dialog">
        <header>
          <h2>证书检测{result ? ` · ${result.domain}` : ''}</h2>
        </header>
        <div className="content">
          {loading && <p className="empty-hint">检测中…</p>}
          {!loading && result && (
            <>
              <div className={`tls-check-summary ${result.ok ? 'ok' : 'fail'}`}>
                <span className="tls-check-badge">{result.ok ? '通过' : '未通过'}</span>
                <span>
                  {result.issuer || '—'} · 到期 {result.expires_at || '—'}（剩余{' '}
                  {result.days_remaining ?? '—'} 天）
                </span>
              </div>
              <ul className="tls-check-list">
                {result.checks.map((c) => (
                  <li key={c.id} className={`tls-check-item level-${c.level}`}>
                    <span className="tls-check-icon" aria-hidden>
                      {checkIcon(c.level)}
                    </span>
                    <div>
                      <strong>{c.label}</strong>
                      <p>{c.message}</p>
                    </div>
                  </li>
                ))}
              </ul>
              {result.dns_names?.length ? (
                <p className="tls-check-dns">
                  SAN: <code>{result.dns_names.join(', ')}</code>
                </p>
              ) : null}
            </>
          )}
        </div>
        <footer>
          <button type="button" className="btn" onClick={onClose}>
            关闭
          </button>
        </footer>
      </div>
    </div>
  )
}
