import { useEffect, useState } from 'react'
import { api, type RouteMetrics } from '../../api/client'

type Props = {
  ruleIndex: number
  pathIndex: number
}

export function InvestigateRouteMetrics({ ruleIndex, pathIndex }: Props) {
  const [metrics, setMetrics] = useState<RouteMetrics | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    setLoading(true)
    api
      .routeMetrics(ruleIndex, pathIndex)
      .then(setMetrics)
      .catch(() => setMetrics(null))
      .finally(() => setLoading(false))
  }, [ruleIndex, pathIndex])

  if (loading) {
    return <p className="empty-hint">加载路由指标…</p>
  }
  if (!metrics || metrics.total === 0) {
    return <p className="empty-hint">暂无该路由的 access 日志样本（约 15m 窗口）</p>
  }

  return (
    <div className="investigate-route-metrics">
      <div className="route-metrics-cards investigate-route-metrics-cards">
        <div className="route-metric-card">
          <div className="label">RPM</div>
          <div className="value">{metrics.rpm.toFixed(1)}</div>
        </div>
        <div className="route-metric-card">
          <div className="label">P95</div>
          <div className="value">{metrics.p95_ms.toFixed(0)}ms</div>
        </div>
        <div className="route-metric-card">
          <div className="label">错误率</div>
          <div
            className="value"
            style={{ color: metrics.error_rate > 5 ? 'var(--danger)' : undefined }}
          >
            {metrics.error_rate.toFixed(1)}%
          </div>
        </div>
        <div className="route-metric-card">
          <div className="label">缓存命中</div>
          <div className="value">{metrics.cache_hit_rate.toFixed(1)}%</div>
        </div>
      </div>
      <div className="investigate-metrics-bars">
        <div className="investigate-metrics-bar-row">
          <span className="chart-hint">错误率</span>
          <div className="investigate-metrics-bar-track">
            <div
              className="investigate-metrics-bar investigate-metrics-bar--err"
              style={{ width: `${Math.min(100, Math.max(0, metrics.error_rate))}%` }}
            />
          </div>
          <span className="investigate-metrics-bar-val">{metrics.error_rate.toFixed(1)}%</span>
        </div>
        <div className="investigate-metrics-bar-row">
          <span className="chart-hint">缓存命中</span>
          <div className="investigate-metrics-bar-track">
            <div
              className="investigate-metrics-bar investigate-metrics-bar--cache"
              style={{ width: `${Math.min(100, Math.max(0, metrics.cache_hit_rate))}%` }}
            />
          </div>
          <span className="investigate-metrics-bar-val">{metrics.cache_hit_rate.toFixed(1)}%</span>
        </div>
        <div className="investigate-metrics-bar-row">
          <span className="chart-hint">P95 延迟</span>
          <div className="investigate-metrics-bar-track">
            <div
              className="investigate-metrics-bar investigate-metrics-bar--latency"
              style={{
                width: `${Math.min(100, (metrics.p95_ms / 3000) * 100)}%`,
              }}
            />
          </div>
          <span className="investigate-metrics-bar-val">{metrics.p95_ms.toFixed(0)}ms</span>
        </div>
      </div>
    </div>
  )
}
