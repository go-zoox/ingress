import { createContext, useContext, useMemo, useState, type ReactNode } from 'react'

export type OverviewStreamState = {
  connected: boolean
  reconnecting: boolean
  fallbackPolling: boolean
  metricsSource?: string
  windowStale?: boolean
}

type OverviewStreamContextValue = {
  stream: OverviewStreamState | null
  setStream: (stream: OverviewStreamState | null) => void
}

const OverviewStreamContext = createContext<OverviewStreamContextValue | null>(null)

export function OverviewStreamProvider({ children }: { children: ReactNode }) {
  const [stream, setStream] = useState<OverviewStreamState | null>(null)
  const value = useMemo(() => ({ stream, setStream }), [stream])
  return (
    <OverviewStreamContext.Provider value={value}>{children}</OverviewStreamContext.Provider>
  )
}

export function useOverviewStream() {
  const ctx = useContext(OverviewStreamContext)
  if (!ctx) {
    throw new Error('useOverviewStream must be used within OverviewStreamProvider')
  }
  return ctx
}

/** Optional hook for components outside overview lifecycle. */
export function useOverviewStreamOptional() {
  return useContext(OverviewStreamContext)
}
