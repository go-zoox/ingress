import { useEffect, useState } from 'react'
import { Navigate, useLocation, useNavigate } from 'react-router-dom'
import { FormField, FormGrid } from '../components/Form'
import { PageLoading } from '../components/PageLoading'
import { useAuth } from '../context/AuthContext'

export function LoginPage() {
  const { loading, config, login, refresh, startOAuth } = useAuth()
  const location = useLocation()
  const navigate = useNavigate()
  const [username, setUsername] = useState('admin')
  const [password, setPassword] = useState('')
  const [err, setErr] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const from =
    typeof location.state === 'object' &&
    location.state &&
    'from' in location.state &&
    typeof (location.state as { from?: string }).from === 'string'
      ? (location.state as { from: string }).from
      : '/'

  useEffect(() => {
    if (loading || !config) return
    if (config.type === 'none' || config.authenticated) return
    if (config.type === 'oauth') {
      startOAuth(from)
    }
  }, [loading, config, from, startOAuth])

  if (loading || !config) {
    return <PageLoading label="正在加载登录配置…" />
  }

  if (config.type === 'none' || config.authenticated) {
    return <Navigate to={from} replace />
  }

  if (config.type === 'oauth') {
    return <PageLoading label="正在跳转到第三方登录…" />
  }

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    setSubmitting(true)
    setErr('')
    try {
      await login(username, password)
      await refresh()
      navigate(from, { replace: true })
    } catch (error: unknown) {
      setErr(error instanceof Error ? error.message : String(error))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="login-page">
      <form className="login-card panel" onSubmit={submit}>
        <div className="login-brand">
          <h1>Ingress Console</h1>
          <p>使用管理员账号登录运维控制台</p>
        </div>
        {err ? <p className="err">{err}</p> : null}
        <FormGrid columns={1}>
          <FormField
            label="用户名"
            autoComplete="username"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
          />
          <FormField
            label="密码"
            type="password"
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </FormGrid>
        <button type="submit" className="btn btn-primary login-submit" disabled={submitting}>
          {submitting ? '登录中…' : '登录'}
        </button>
      </form>
    </div>
  )
}
