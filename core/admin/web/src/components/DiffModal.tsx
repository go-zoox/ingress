export function DiffModal({
  open,
  diffHtml,
  title = '配置变更（草稿 vs 已发布）',
  onClose,
}: {
  open: boolean
  diffHtml: string
  title?: string
  onClose: () => void
}) {
  if (!open) return null

  return (
    <div className="modal-overlay open" onClick={(e) => e.target === e.currentTarget && onClose()}>
      <div className="modal" role="dialog">
        <header>
          <h2>{title}</h2>
        </header>
        <div className="content">
          <pre className="diff" dangerouslySetInnerHTML={{ __html: diffHtml || '(无变更)' }} />
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
