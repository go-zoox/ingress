import { createContext, useContext, type ReactNode } from 'react'
import { useAppNotifications } from '../hooks/useAppNotifications'

type NotificationContextValue = ReturnType<typeof useAppNotifications>

const NotificationContext = createContext<NotificationContextValue | null>(null)

type ProviderProps = {
  runtimeDrift?: boolean
  revisionDrift?: boolean
  children: ReactNode
}

export function NotificationProvider({ runtimeDrift, revisionDrift, children }: ProviderProps) {
  const value = useAppNotifications({ runtimeDrift, revisionDrift })
  return <NotificationContext.Provider value={value}>{children}</NotificationContext.Provider>
}

export function useNotificationContext() {
  const ctx = useContext(NotificationContext)
  if (!ctx) {
    throw new Error('useNotificationContext must be used within NotificationProvider')
  }
  return ctx
}
