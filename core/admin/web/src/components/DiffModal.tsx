import { Drawer } from './Drawer'

export function DiffModal({
  open,
  diffHtml,
  title = '配置变更（草稿 vs 已发布）',
  onClose,
  confirmLabel,
  onConfirm,
}: {
  open: boolean
  diffHtml: string
  title?: string
  onClose: () => void
  confirmLabel?: string
  onConfirm?: () => void
}) {
  return (
    <Drawer
      open={open}
      title={title}
      onClose={onClose}
      width={920}
      footer={
        <>
          <button type="button" className="btn" onClick={onClose}>
            取消
          </button>
          {onConfirm && confirmLabel ? (
            <button type="button" className="btn btn-primary" onClick={onConfirm}>
              {confirmLabel}
            </button>
          ) : (
            <button type="button" className="btn" onClick={onClose}>
              关闭
            </button>
          )}
        </>
      }
    >
      <pre className="diff diff-drawer" dangerouslySetInnerHTML={{ __html: diffHtml || '(无变更)' }} />
    </Drawer>
  )
}
