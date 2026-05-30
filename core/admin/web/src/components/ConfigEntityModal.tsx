import { useCallback, useEffect, useId, useRef, useState, type CSSProperties, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
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
      width={wide ? 920 : 560}
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
  const triggerRef = useRef<HTMLButtonElement>(null)
  const menuId = useId()
  const [menuOpen, setMenuOpen] = useState(false)
  const [menuStyle, setMenuStyle] = useState<CSSProperties>({})

  const updateMenuPosition = useCallback(() => {
    const el = triggerRef.current
    if (!el) return
    const rect = el.getBoundingClientRect()
    const panelW = 148
    const panelH = 200
    let top = rect.bottom + 4
    let left = rect.right - panelW
    if (left < 12) left = 12
    if (top + panelH > window.innerHeight - 12) {
      top = Math.max(12, rect.top - panelH - 4)
    }
    setMenuStyle({ top, left, minWidth: panelW })
  }, [])

  useEffect(() => {
    if (!menuOpen) return
    updateMenuPosition()
    const onScrollOrResize = () => updateMenuPosition()
    window.addEventListener('scroll', onScrollOrResize, true)
    window.addEventListener('resize', onScrollOrResize)
    return () => {
      window.removeEventListener('scroll', onScrollOrResize, true)
      window.removeEventListener('resize', onScrollOrResize)
    }
  }, [menuOpen, updateMenuPosition])

  useEffect(() => {
    if (!menuOpen) return
    const close = (e: MouseEvent) => {
      const target = e.target as Node
      if (triggerRef.current?.contains(target)) return
      const panel = document.getElementById(menuId)
      if (panel?.contains(target)) return
      setMenuOpen(false)
    }
    document.addEventListener('mousedown', close)
    return () => document.removeEventListener('mousedown', close)
  }, [menuOpen, menuId])

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

  const menuPanel = menuOpen ? (
    <div
      id={menuId}
      className="action-menu-panel action-menu-panel--fixed"
      role="menu"
      style={menuStyle}
    >
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
  ) : null

  return (
    <div className="row-actions">
      <button type="button" className="action-link" onClick={onEdit}>
        编辑
      </button>
      <div className="action-menu">
        <button
          ref={triggerRef}
          type="button"
          className="action-link action-advanced"
          aria-expanded={menuOpen}
          onClick={() => {
            if (!menuOpen) updateMenuPosition()
            setMenuOpen((v) => !v)
          }}
        >
          高级
        </button>
        {menuPanel && createPortal(menuPanel, document.body)}
      </div>
    </div>
  )
}
