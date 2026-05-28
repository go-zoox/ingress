import { useCallback, useEffect, useMemo, useState } from 'react'
import { api } from '../api/client'
import { buildAppNotifications, type AppNotification } from '../lib/appNotifications'
import { loadPreferences } from '../lib/preferences'
import { markNotificationRead } from '../lib/notificationReadState'

export type { AppNotification } from '../lib/appNotifications'

type Options = {
  runtimeDrift?: boolean
  revisionDrift?: boolean
}

export function useAppNotifications(options: Options = {}) {
  const [items, setItems] = useState<AppNotification[]>([])
  const [tick, setTick] = useState(0)

  const refresh = useCallback(() => {
    const window = loadPreferences().metricsWindow
    api
      .overviewMetrics(window)
      .then((metrics) => {
        setItems(buildAppNotifications(metrics, options))
      })
      .catch(() => setItems([]))
  }, [options.runtimeDrift, options.revisionDrift])

  useEffect(() => {
    refresh()
    const timer = window.setInterval(refresh, 60_000)
    return () => window.clearInterval(timer)
  }, [refresh, tick])

  const markRead = useCallback((id: string) => {
    setItems((prev) => {
      const target = prev.find((item) => item.id === id && !item.read)
      if (!target) return prev
      markNotificationRead(target.id, target.fingerprint)
      return prev.map((item) =>
        item.id === id
          ? { ...item, read: true, readAt: new Date().toISOString() }
          : item,
      )
    })
  }, [])

  const markAllRead = useCallback(() => {
    setItems((prev) => {
      const next = prev.map((item) => {
        if (item.read) return item
        markNotificationRead(item.id, item.fingerprint)
        return { ...item, read: true, readAt: new Date().toISOString() }
      })
      return next
    })
  }, [])

  const unreadCount = useMemo(() => items.filter((item) => !item.read).length, [items])

  return {
    items,
    unreadCount,
    count: unreadCount,
    markRead,
    markAllRead,
    refresh: () => setTick((v) => v + 1),
  }
}
