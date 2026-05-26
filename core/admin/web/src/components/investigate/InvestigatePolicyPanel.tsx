import type { HealthCheckResult, RouteDetail, WAFEvent } from '../../api/client'

type Props = {
  route: RouteDetail | null
  wafRecent: WAFEvent[]
  healthChecks: HealthCheckResult[]
}

export function InvestigatePolicyPanel({ route, wafRecent, healthChecks }: Props) {
  const downs = healthChecks.filter((h) => h.status === 'down')

  return (
    <div className="investigate-policy-grid">
      <div className="investigate-policy-card">
        <h3>路由策略</h3>
        {!route ? (
          <p className="empty-hint">无路由详情（未命中或未加载）</p>
        ) : (
          <ul className="investigate-policy-list">
            <li>
              Backend: <code>{route.backend.target}</code> ({route.backend.type})
            </li>
            {route.cache ? (
              <li>
                缓存: 启用 · TTL {route.cache.ttl}s
              </li>
            ) : (
              <li>缓存: 未启用</li>
            )}
            {route.auth ? (
              <li>
                认证: {route.auth.summary || route.auth.type}
                {!route.auth.enabled ? ' (disabled)' : ''}
              </li>
            ) : (
              <li>认证: 无</li>
            )}
            {route.waf?.patched ? <li>WAF: 路由级 patch</li> : <li>WAF: 继承全局</li>}
            {route.health_check ? (
              <li>
                健康探测: {route.health_check.ok ? 'UP' : 'DOWN'} · {route.health_check.method}{' '}
                {route.health_check.path}
              </li>
            ) : (
              <li>健康探测: 未配置</li>
            )}
          </ul>
        )}
      </div>

      <div className="investigate-policy-card">
        <h3>健康检查</h3>
        {healthChecks.length === 0 ? (
          <p className="empty-hint ok-hint">无相关探测目标</p>
        ) : (
          <ul className="investigate-policy-list">
            {healthChecks.map((h) => (
              <li key={h.key} className={h.status === 'down' ? 'text-danger' : ''}>
                {h.status.toUpperCase()} · {h.backend}
                {h.error ? ` — ${h.error}` : ''}
              </li>
            ))}
          </ul>
        )}
        {downs.length > 0 ? (
          <p className="chart-hint" style={{ marginTop: 8 }}>
            {downs.length} 个 DOWN
          </p>
        ) : null}
      </div>

      <div className="investigate-policy-card">
        <h3>近期 WAF block</h3>
        {wafRecent.length === 0 ? (
          <p className="empty-hint ok-hint">无近期 block</p>
        ) : (
          <ul className="investigate-policy-list">
            {wafRecent.map((e) => (
              <li key={e.id}>
                <code>{e.rule}</code> · {formatTime(e.created_at)}
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}

function formatTime(iso: string) {
  try {
    return new Date(iso).toLocaleString('zh-CN', { hour12: false })
  } catch {
    return iso
  }
}
