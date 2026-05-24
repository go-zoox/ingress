import { useState } from 'react'
import type { ConfigRevisionSummary } from '../api/client'
import { api } from '../api/client'

interface RollbackConfirmModalProps {
  revision: ConfigRevisionSummary
  diffSummary?: string
  onConfirm: () => void
  onCancel: () => void
}

export function RollbackConfirmModal({ revision, diffSummary, onConfirm, onCancel }: RollbackConfirmModalProps) {
  const [loading, setLoading] = useState(false)
  const [err, setErr] = useState('')

  const handleRollback = async () => {
    setLoading(true)
    setErr('')
    try {
      // Fetch the full revision content
      const detail = await api.configRevision(revision.id)
      // Validate first
      await api.validateConfig(detail.content)
      // Publish the revision content
      await api.publishConfig(detail.content, `回滚到版本 #${revision.id}`)
      onConfirm()
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="modal-overlay open" onClick={(e) => e.target === e.currentTarget && onCancel()}>
      <div className="modal rollback-confirm-modal" role="dialog">
        <header>
          <h2>确认回滚到版本 #{revision.id}？</h2>
        </header>
        <div className="content">
          <p className="rollback-info">
            版本 <code>#{revision.id}</code> · Hash <code>{revision.hash}</code>
          </p>
          <p className="rollback-meta">
            说明：{revision.note || '无'} · {new Date(revision.created_at).toLocaleString()}
          </p>
          {diffSummary && (
            <p className="rollback-diff-summary">{diffSummary}</p>
          )}
          <p className="rollback-warning">
            ⚠️ 回滚将加载该版本内容，校验后直接发布并 reload。当前未保存的变更将丢失。
          </p>
          {err && <p className="err">{err}</p>}
        </div>
        <footer>
          <button type="button" className="btn" onClick={onCancel} disabled={loading}>
            取消
          </button>
          <button type="button" className="btn btn-danger" onClick={handleRollback} disabled={loading}>
            {loading ? '回滚中…' : '回滚并发布'}
          </button>
        </footer>
      </div>
    </div>
  )
}
