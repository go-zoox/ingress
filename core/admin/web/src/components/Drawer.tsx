import { useEffect, type ReactNode } from 'react'

type DrawerProps = {
  open: boolean
  title: string
  onClose: () => void
  children: ReactNode
  footer?: ReactNode
  width?: number
}

export function Drawer({ open, title, onClose, children, footer, width = 420 }: DrawerProps) {
  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', onKey)
    const prev = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', onKey)
      document.body.style.overflow = prev
    }
  }, [open, onClose])

  if (!open) return null

  return (
    <div
      className="drawer-overlay open"
      role="presentation"
      onClick={(e) => e.target === e.currentTarget && onClose()}
    >
      <aside
        className="drawer"
        role="dialog"
        aria-modal="true"
        aria-labelledby="drawer-title"
        style={{ width: `min(${width}px, 100vw)` }}
        onClick={(e) => e.stopPropagation()}
      >
        <header className="drawer-head">
          <h2 id="drawer-title">{title}</h2>
          <button type="button" className="drawer-close" onClick={onClose} aria-label="关闭">
            ×
          </button>
        </header>
        <div className="drawer-body">{children}</div>
        {footer ? <footer className="drawer-foot">{footer}</footer> : null}
      </aside>
    </div>
  )
}
