import { memo } from 'react'
import { Link } from 'react-router-dom'
import type { OverviewMetrics, TLSCert, HealthCheckResult, WAFEvent } from '../api/client'

export type AttentionItem = {
  level: 'danger' | 'warn' | 'info'
  title: string
  detail: string
  href?: string
}

type Props = {
  metrics: OverviewMetrics | null
  certs: TLSCert[]
  healthChecks: HealthCheckResult[]
  wafBlocks: WAFEvent[]
}

export const OverviewAttentionPanel = memo(function OverviewAttentionPanel({
  metrics,
  certs,
  healthChecks,
  wafBlocks,
}: Props) {
  const items = buildAttentionItems(metrics, certs, healthChecks, wafBlocks)

  return (
    <div className="panel attention-panel">
      <div className="panel-head">
        <h2>需要关注</h2>
        <span className="chart-hint">{items.length === 0 ? '当前无告警项' : `${items.length} 项`}</span>
      </div>
      <div className="panel-body">
        {items.length === 0 ? (
          <p className="empty-hint ok-hint">各项指标正常，暂无需要立即处理的事项。</p>
        ) : (
          <ul className="attention-list">
            {items.map((item, i) => (
              <li key={`${item.title}-${i}`} className={`attention-item attention-${item.level}`}>
                <div className="attention-main">
                  <span className={`attention-dot attention-dot-${item.level}`} />
                  <div>
                    <div className="attention-title">{item.title}</div>
                    <div className="attention-detail">{item.detail}</div>
                  </div>
                </div>
                {item.href ? (
                  <Link to={item.href} className="btn btn-ghost btn-sm">
                    查看
                  </Link>
                ) : null}
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
})

function buildAttentionItems(
  metrics: OverviewMetrics | null,
  certs: TLSCert[],
  healthChecks: HealthCheckResult[],
  wafBlocks: WAFEvent[],
): AttentionItem[] {
  const out: AttentionItem[] = []

  const criticalCerts = certs.filter((c) => c.days_remaining < 7)
  const warnCerts = certs.filter((c) => c.days_remaining >= 7 && c.days_remaining < 30)
  for (const c of criticalCerts.slice(0, 3)) {
    out.push({
      level: 'danger',
      title: `证书即将过期：${c.domain}`,
      detail: `剩余 ${c.days_remaining} 天`,
      href: '/tls',
    })
  }
  for (const c of warnCerts.slice(0, Math.max(0, 3 - criticalCerts.length))) {
    out.push({
      level: 'warn',
      title: `证书需关注：${c.domain}`,
      detail: `剩余 ${c.days_remaining} 天`,
      href: '/tls',
    })
  }

  const downs = healthChecks.filter((h) => h.status === 'down')
  for (const h of downs.slice(0, 3)) {
    out.push({
      level: 'danger',
      title: `健康检查 DOWN：${h.host}`,
      detail: h.error || h.url || h.backend,
      href: '/healths',
    })
  }

  if (metrics && metrics.error_rate > 10) {
    out.push({
      level: 'danger',
      title: '错误率偏高',
      detail: `${metrics.error_rate.toFixed(1)}%（4xx+5xx，窗口 ${metrics.window}）`,
      href: '/logs',
    })
  } else if (metrics && metrics.error_rate > 5) {
    out.push({
      level: 'warn',
      title: '错误率上升',
      detail: `${metrics.error_rate.toFixed(1)}%（窗口 ${metrics.window}）`,
      href: '/logs',
    })
  }

  if (metrics && metrics.p95_ms > 2000) {
    out.push({
      level: 'warn',
      title: 'P95 延迟较高',
      detail: formatMs(metrics.p95_ms),
      href: '/logs',
    })
  }

  for (const e of wafBlocks.slice(0, 3)) {
    out.push({
      level: 'info',
      title: `WAF ${e.action}：${e.rule}`,
      detail: `${e.host}${e.path}`,
      href: '/waf',
    })
  }

  if (metrics) {
    for (const s of metrics.slowest.slice(0, 3)) {
      if (s.duration_ms < 1000) break
      out.push({
        level: 'warn',
        title: `慢请求 ${formatMs(s.duration_ms)}`,
        detail: `${s.host} ${s.method} ${s.path} → ${s.status}`,
        href: '/logs',
      })
    }
  }

  return out.slice(0, 12)
}

function formatMs(ms: number) {
  if (ms >= 1000) return `${(ms / 1000).toFixed(2)}s`
  return `${Math.round(ms)}ms`
}
