import { memo, useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { Bell } from 'lucide-react'
import { useNotificationContext } from '../context/NotificationContext'

export const NotificationMenu = memo(function NotificationMenu() {
  const { items, unreadCount, markRead } = useNotificationContext()
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const onDoc = (e: MouseEvent) => {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onDoc)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDoc)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  const unreadItems = items.filter((item) => !item.read)
  const previewItems = unreadItems.length > 0 ? unreadItems : items

  return (
    <div className="notification-menu" ref={rootRef}>
      <button
        type="button"
        className={`notification-menu-trigger app-chrome-trigger${open ? ' active' : ''}`}
        aria-expanded={open}
        aria-haspopup="dialog"
        aria-label={unreadCount > 0 ? `消息通知 ${unreadCount} 条未读` : '消息通知'}
        title={unreadCount > 0 ? `消息通知（${unreadCount} 条未读）` : '消息通知'}
        onClick={() => setOpen((v) => !v)}
      >
        <span className="app-chrome-trigger-icon">
          <Bell size={18} aria-hidden />
          {unreadCount > 0 ? (
            <span className="notification-menu-badge">{unreadCount > 99 ? '99+' : unreadCount}</span>
          ) : null}
        </span>
      </button>
      {open ? (
        <div className="notification-menu-panel" role="dialog" aria-label="消息通知">
          <div className="notification-menu-head">
            <Bell size={16} aria-hidden />
            <span>消息通知</span>
          </div>
          {previewItems.length === 0 ? (
            <p className="notification-menu-empty">暂无消息</p>
          ) : (
            <ul className="notification-menu-list">
              {previewItems.slice(0, 5).map((item) => (
                <li
                  key={item.id}
                  className={`notification-item notification-${item.level}${item.read ? ' notification-read' : ''}`}
                >
                  <div className="notification-item-title">{item.title}</div>
                  <div className="notification-item-detail">{item.detail}</div>
                  <div className="notification-item-actions">
                    {!item.read ? (
                      <button
                        type="button"
                        className="notification-item-mark-read"
                        onClick={() => markRead(item.id)}
                      >
                        标记已读
                      </button>
                    ) : null}
                    {item.href ? (
                      <Link
                        to={item.href}
                        className="notification-item-link"
                        onClick={() => {
                          if (!item.read) markRead(item.id)
                          setOpen(false)
                        }}
                      >
                        查看
                      </Link>
                    ) : null}
                  </div>
                </li>
              ))}
            </ul>
          )}
          <div className="notification-menu-foot">
            <Link to="/messages" className="btn btn-ghost btn-sm" onClick={() => setOpen(false)}>
              全部消息
            </Link>
          </div>
        </div>
      ) : null}
    </div>
  )
})
