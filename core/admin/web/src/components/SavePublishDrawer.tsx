import { Drawer } from './Drawer'

export function SavePublishDrawer({
  open,
  diffHtml,
  title = '保存与发布',
  busy = false,
  onClose,
  onSaveOnly,
  onSaveAndPublish,
}: {
  open: boolean
  diffHtml: string
  title?: string
  busy?: boolean
  onClose: () => void
  onSaveOnly: () => void
  onSaveAndPublish: () => void
}) {
  return (
    <Drawer
      open={open}
      title={title}
      onClose={onClose}
      width={920}
      footer={
        <>
          <button type="button" className="btn" onClick={onClose} disabled={busy}>
            取消
          </button>
          <button type="button" className="btn" onClick={onSaveOnly} disabled={busy}>
            仅保存
          </button>
          <button type="button" className="btn btn-primary" onClick={onSaveAndPublish} disabled={busy}>
            保存并发布
          </button>
        </>
      }
    >
      <p style={{ marginTop: 0, color: 'var(--text-muted)', fontSize: 13 }}>
        以下为已发布配置与即将写入内容的差异。确认后可仅保存到 YAML，或保存并热加载发布。
      </p>
      <pre className="diff diff-drawer" dangerouslySetInnerHTML={{ __html: diffHtml || '(无变更)' }} />
    </Drawer>
  )
}
