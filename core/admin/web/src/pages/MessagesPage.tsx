import { useMemo, useState, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { Bell } from 'lucide-react'
import { PageHeader } from '../components/PageHeader'
import { useNotificationContext } from '../context/NotificationContext'
import type { AppNotification } from '../lib/appNotifications'

type Filter = 'all' | 'unread' | 'read'

export function MessagesPage() {
  const { items, markRead, markAllRead } = useNotificationContext()
  const [filter, setFilter] = useState<Filter>('all')

  const filtered = useMemo(() => {
    if (filter === 'unread') return items.filter((item) => !item.read)
    if (filter === 'read') return items.filter((item) => item.read)
    return items
  }, [items, filter])

  const unreadCount = items.filter((item) => !item.read).length

  return (
    <div className="page messages-page">
      <PageHeader
        title="消息通知"
        desc="系统提示与运维告警摘要；已读状态保存在浏览器本地"
        actions={
          unreadCount > 0 ? (
            <button type="button" className="btn btn-sm btn-ghost" onClick={markAllRead}>
              全部标记已读
            </button>
          ) : null
        }
      />

      <div className="page-tabs messages-tabs" role="tablist" aria-label="消息筛选">
        <TabButton active={filter === 'all'} onClick={() => setFilter('all')}>
          全部 ({items.length})
        </TabButton>
        <TabButton active={filter === 'unread'} onClick={() => setFilter('unread')}>
          未读 ({unreadCount})
        </TabButton>
        <TabButton active={filter === 'read'} onClick={() => setFilter('read')}>
          已读 ({items.length - unreadCount})
        </TabButton>
      </div>

      <div className="panel">
        <div className="panel-body">
          {filtered.length === 0 ? (
            <p className="empty-hint">
              <Bell size={16} style={{ verticalAlign: 'middle', marginRight: 6 }} />
              {filter === 'unread' ? '暂无未读消息' : filter === 'read' ? '暂无已读消息' : '暂无消息'}
            </p>
          ) : (
            <ul className="messages-list">
              {filtered.map((item) => (
                <MessageRow key={item.id} item={item} onMarkRead={markRead} />
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  )
}

function TabButton({
  active,
  onClick,
  children,
}: {
  active: boolean
  onClick: () => void
  children: ReactNode
}) {
  return (
    <button
      type="button"
      role="tab"
      aria-selected={active}
      className={active ? 'btn btn-sm active' : 'btn btn-sm btn-ghost'}
      onClick={onClick}
    >
      {children}
    </button>
  )
}

function MessageRow({
  item,
  onMarkRead,
}: {
  item: AppNotification
  onMarkRead: (id: string) => void
}) {
  return (
    <li className={`messages-item messages-item-${item.level}${item.read ? ' messages-item-read' : ''}`}>
      <div className="messages-item-main">
        <div className="messages-item-head">
          <span className="messages-item-title">{item.title}</span>
          <span className={`messages-item-status${item.read ? ' read' : ''}`}>
            {item.read ? '已读' : '未读'}
          </span>
        </div>
        <p className="messages-item-detail">{item.detail}</p>
        {item.readAt ? (
          <time className="messages-item-time">{formatTime(item.readAt)}</time>
        ) : null}
      </div>
      <div className="messages-item-actions">
        {!item.read ? (
          <button type="button" className="btn btn-ghost btn-sm" onClick={() => onMarkRead(item.id)}>
            标记已读
          </button>
        ) : null}
        {item.href ? (
          <Link
            to={item.href}
            className="btn btn-sm"
            onClick={() => {
              if (!item.read) onMarkRead(item.id)
            }}
          >
            查看
          </Link>
        ) : null}
      </div>
    </li>
  )
}

function formatTime(iso: string) {
  try {
    return new Date(iso).toLocaleString('zh-CN', { hour12: false })
  } catch {
    return iso
  }
}
