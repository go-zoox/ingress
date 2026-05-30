import { useEffect, useState } from 'react'
import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { PageLoading } from '../components/PageLoading'
import { useAuth } from '../context/AuthContext'

export function RequireAuth() {
  const { loading, config } = useAuth()
  const location = useLocation()
  const [oauthStarted, setOauthStarted] = useState(false)

  useEffect(() => {
    if (loading || !config || config.authenticated) return
    if (config.type === 'oauth' && !oauthStarted) {
      setOauthStarted(true)
      const redirect = `${location.pathname}${location.search}`
      window.location.assign(
        `/api/v1/auth/oauth/login?redirect=${encodeURIComponent(redirect || '/')}`,
      )
    }
  }, [loading, config, location.pathname, location.search, oauthStarted])

  if (loading) {
    return <PageLoading label="正在验证登录状态…" />
  }

  if (!config || config.type === 'none' || config.authenticated) {
    return <Outlet />
  }

  if (config.type === 'oauth') {
    return <PageLoading label="正在跳转到第三方登录…" />
  }

  return <Navigate to="/login" replace state={{ from: location.pathname }} />
}
