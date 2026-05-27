import { useEffect, useRef, useState, type ReactNode } from 'react'
import { Drawer } from './Drawer'

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
  return (
    <Drawer
      open={open}
      title={title}
      onClose={onClose}
      width={wide ? 720 : 560}
      footer={(
        <>
          <button type="button" className="btn" onClick={onClose}>
            取消
          </button>
          <button type="button" className="btn btn-primary" disabled={disableSave} onClick={onSave}>
            {saveLabel}
          </button>
        </>
      )}
    >
      {children}
    </Drawer>
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

export type EntityRowMenuItem = {
  label: string
  onClick: () => void
  disabled?: boolean
  danger?: boolean
}

export function EntityRowActions({
  onEdit,
  onDelete,
  onMoveUp,
  onMoveDown,
  disableMoveUp,
  disableMoveDown,
  menuItems,
}: {
  onEdit: () => void
  onDelete: () => void
  onMoveUp?: () => void
  onMoveDown?: () => void
  disableMoveUp?: boolean
  disableMoveDown?: boolean
  menuItems?: EntityRowMenuItem[]
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

  const runMenuItem = (item: EntityRowMenuItem) => {
    if (item.disabled) return
    setMenuOpen(false)
    item.onClick()
  }

  const runDelete = () => {
    setMenuOpen(false)
    onDelete()
  }

  const canMove = onMoveUp != null || onMoveDown != null
  const extraItems = menuItems ?? []

  return (
    <div className="row-actions">
      <button type="button" className="action-link" onClick={onEdit}>
        编辑
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
            {canMove && (
              <>
                <button
                  type="button"
                  role="menuitem"
                  className="action-menu-item"
                  disabled={disableMoveUp}
                  onClick={() => {
                    if (disableMoveUp) return
                    setMenuOpen(false)
                    onMoveUp?.()
                  }}
                >
                  上移
                </button>
                <button
                  type="button"
                  role="menuitem"
                  className="action-menu-item"
                  disabled={disableMoveDown}
                  onClick={() => {
                    if (disableMoveDown) return
                    setMenuOpen(false)
                    onMoveDown?.()
                  }}
                >
                  下移
                </button>
              </>
            )}
            {extraItems.map((item) => (
              <button
                key={item.label}
                type="button"
                role="menuitem"
                className={`action-menu-item${item.danger ? ' action-danger' : ''}`}
                disabled={item.disabled}
                onClick={() => runMenuItem(item)}
              >
                {item.label}
              </button>
            ))}
            {(canMove || extraItems.length > 0) && <div className="action-menu-sep" aria-hidden />}
            <button
              type="button"
              role="menuitem"
              className="action-menu-item action-danger"
              onClick={runDelete}
            >
              删除
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
