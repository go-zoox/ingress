import type { ReactNode } from 'react'

export function ConfigEntityModal({
  open,
  title,
  children,
  saveLabel = '保存',
  wide,
  onClose,
  onSave,
  disableSave,
}: {
  open: boolean
  title: string
  children: ReactNode
  saveLabel?: string
  wide?: boolean
  disableSave?: boolean
  onClose: () => void
  onSave: () => void
}) {
  if (!open) return null

  return (
    <div className="modal-overlay open" onClick={(e) => e.target === e.currentTarget && onClose()}>
      <div className={`modal config-entity-modal${wide ? ' config-entity-modal--wide' : ''}`} role="dialog">
        <header>
          <h2>{title}</h2>
        </header>
        <div className="content">{children}</div>
        <footer>
          <button type="button" className="btn" onClick={onClose}>
            取消
          </button>
          <button type="button" className="btn btn-primary" disabled={disableSave} onClick={onSave}>
            {saveLabel}
          </button>
        </footer>
      </div>
    </div>
  )
}

export function EntityTableToolbar({
  label,
  onAdd,
}: {
  label: string
  onAdd: () => void
}) {
  return (
    <div className="entity-table-toolbar">
      <span className="entity-table-label">{label}</span>
      <button type="button" className="btn btn-ghost" onClick={onAdd}>
        + 添加
      </button>
    </div>
  )
}

export function EntityRowActions({
  onEdit,
  onDelete,
}: {
  onEdit: () => void
  onDelete: () => void
}) {
  return (
    <div className="row-actions">
      <button type="button" className="action-link" onClick={onEdit}>
        编辑
      </button>
      <button type="button" className="action-link action-danger" onClick={onDelete}>
        删除
      </button>
    </div>
  )
}
