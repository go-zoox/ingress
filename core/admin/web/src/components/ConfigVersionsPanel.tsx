import { useEffect, useState } from 'react'
import { api, type ConfigRevisionDetail, type ConfigRevisionSummary } from '../api/client'

export function ConfigVersionsPanel({
  onRestore,
}: {
  onRestore: (content: string) => void
}) {
  const [rows, setRows] = useState<ConfigRevisionSummary[]>([])
  const [err, setErr] = useState('')
  const [detail, setDetail] = useState<ConfigRevisionDetail | null>(null)
  const [loadingId, setLoadingId] = useState<number | null>(null)

  useEffect(() => {
    api
      .configRevisions()
      .then((data) => setRows(Array.isArray(data) ? data : []))
      .catch((e: Error) => setErr(e.message))
  }, [])

  const openDetail = async (id: number) => {
    setLoadingId(id)
    setErr('')
    try {
      const row = await api.configRevision(id)
      setDetail(row)
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e))
    } finally {
      setLoadingId(null)
    }
  }

  return (
    <div className="config-versions">
      {err && <p className="err">{err}</p>}
      <table className="data config-revisions">
        <thead>
          <tr>
            <th>版本</th>
            <th>Hash</th>
            <th>说明</th>
            <th>时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          {rows.length === 0 ? (
            <tr>
              <td colSpan={5} className="empty-hint">
                暂无保存/发布记录
              </td>
            </tr>
          ) : (
            rows.map((r) => (
              <tr key={r.id}>
                <td>#{r.id}</td>
                <td>
                  <code>{r.hash}</code>
                </td>
                <td>{r.note || '—'}</td>
                <td>{new Date(r.created_at).toLocaleString()}</td>
                <td className="row-actions">
                  <button
                    type="button"
                    className="action-link"
                    disabled={loadingId === r.id}
                    onClick={() => openDetail(r.id)}
                  >
                    {loadingId === r.id ? '加载中…' : '查看'}
                  </button>
                </td>
              </tr>
            ))
          )}
        </tbody>
      </table>

      {detail && (
        <div className="modal-overlay open" onClick={(e) => e.target === e.currentTarget && setDetail(null)}>
          <div className="modal config-revision-modal" role="dialog">
            <header>
              <h2>
                版本 #{detail.id} · {detail.note || '无说明'}
              </h2>
            </header>
            <div className="content">
              <p className="config-revision-meta">
                <code>{detail.hash}</code> · {new Date(detail.created_at).toLocaleString()}
              </p>
              <textarea className="code config-revision-view" readOnly value={detail.content} spellCheck={false} />
            </div>
            <footer>
              <button type="button" className="btn" onClick={() => setDetail(null)}>
                关闭
              </button>
              <button
                type="button"
                className="btn btn-primary"
                onClick={() => {
                  onRestore(detail.content)
                  setDetail(null)
                }}
              >
                恢复为草稿
              </button>
            </footer>
          </div>
        </div>
      )}
    </div>
  )
}
