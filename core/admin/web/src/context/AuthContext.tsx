import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import { api, type AuthConfigView, type AuthUser } from '../api/client'

type AuthState = {
  loading: boolean
  config: AuthConfigView | null
  user: AuthUser | null
  refresh: () => Promise<void>
  login: (username: string, password: string) => Promise<void>
  logout: () => Promise<void>
  startOAuth: (redirect?: string) => void
}

const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [loading, setLoading] = useState(true)
  const [config, setConfig] = useState<AuthConfigView | null>(null)
  const [user, setUser] = useState<AuthUser | null>(null)

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      const next = await api.authConfig()
      setConfig(next)
      setUser(next.authenticated ? next.user ?? null : null)
    } catch {
      setConfig({ type: 'none', authenticated: false })
      setUser(null)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void refresh()
  }, [refresh])

  const login = useCallback(async (username: string, password: string) => {
    const result = await api.authLogin(username, password)
    setUser(result.user)
    setConfig((prev) =>
      prev
        ? { ...prev, authenticated: true, user: result.user }
        : { type: 'none', authenticated: true, user: result.user },
    )
  }, [])

  const logout = useCallback(async () => {
    await api.authLogout()
    setUser(null)
    setConfig((prev) =>
      prev ? { ...prev, authenticated: false, user: undefined } : { type: 'none', authenticated: false },
    )
  }, [])

  const startOAuth = useCallback((redirect = '/') => {
    const url = `/api/v1/auth/oauth/login?redirect=${encodeURIComponent(redirect)}`
    window.location.assign(url)
  }, [])

  const value = useMemo(
    () => ({ loading, config, user, refresh, login, logout, startOAuth }),
    [loading, config, user, refresh, login, logout, startOAuth],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) {
    throw new Error('useAuth must be used within AuthProvider')
  }
  return ctx
}
