import { useEffect, useState } from 'react'
import { api, type ConfigPreview } from '../api/client'
import { CONFIG_MODULE_LABELS, buildDiff } from '../lib/config'

export function PreviewModal({
  open,
  draft,
  published,
  onClose,
  onPublish,
}: {
  open: boolean
  draft: string
  published: string
  onClose: () => void
  onPublish: () => void
}) {
  const [preview, setPreview] = useState<ConfigPreview | null>(null)
  const [loading, setLoading] = useState(false)
  const [err, setErr] = useState('')

  useEffect(() => {
    if (!open) {
      setPreview(null)
      setErr('')
      return
    }
    setLoading(true)
    setErr('')
    api
      .previewConfig(draft)
      .then(setPreview)
      .catch((e: Error) => setErr(e.message))
      .finally(() => setLoading(false))
  }, [open, draft])

  if (!open) return null

  const diffHtml = buildDiff(published, draft)

  return (
    <div className="modal-overlay open" onClick={(e) => e.target === e.currentTarget && onClose()}>
      <div className="modal config-preview-modal" role="dialog">
        <header>
          <h2>发布预览</h2>
        </header>
        <div className="content">
          {loading && <p className="empty-hint">正在校验并分析变更…</p>}
          {err && <p className="err">{err}</p>}
          {!loading && preview && (
            <>
              <div className={`config-preview-summary ${preview.valid ? 'ok' : 'fail'}`}>
                <span>{preview.valid ? '校验通过' : '校验失败'}</span>
                <span>
                  草稿 <code>{preview.hash}</code>
                  {preview.changed ? (
                    <>
                      {' '}
                      · 相对已发布 <code>{preview.published_hash}</code> 有变更
                    </>
                  ) : (
                    ' · 与已发布内容一致'
                  )}
                </span>
              </div>
              {!preview.valid && preview.error && (
                <pre className="validate-err">{preview.error}</pre>
              )}
              {preview.modules_changed.length > 0 && (
                <div className="config-preview-modules">
                  <strong>变更模块</strong>
                  <div className="config-preview-module-tags">
                    {preview.modules_changed.map((id) => (
                      <span key={id} className="badge badge-regex">
                        {CONFIG_MODULE_LABELS[id] || id}
                      </span>
                    ))}
                  </div>
                </div>
              )}
              <div>
                <strong>Diff（已发布 → 草稿）</strong>
                <pre className="diff" dangerouslySetInnerHTML={{ __html: diffHtml }} />
              </div>
            </>
          )}
        </div>
        <footer>
          <button type="button" className="btn" onClick={onClose}>
            关闭
          </button>
          <button
            type="button"
            className="btn btn-primary"
            disabled={!preview?.valid}
            onClick={() => {
              onClose()
              onPublish()
            }}
          >
            继续发布
          </button>
        </footer>
      </div>
    </div>
  )
}
