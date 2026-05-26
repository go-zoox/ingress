import { memo, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import {
  AlertTriangle,
  HeartPulse,
  ShieldBan,
  Clock,
  FileWarning,
} from 'lucide-react'
import type { OverviewMetrics, TLSCert, HealthCheckResult, WAFEvent } from '../api/client'

export type AttentionItem = {
  level: 'danger' | 'warn' | 'info'
  title: string
  detail: string
  href?: string
  icon?: 'cert' | 'error' | 'slow'
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
  const downs = healthChecks.filter((h) => h.status === 'down')
  const otherItems = buildOtherAttentionItems(metrics, certs, wafBlocks)
  const totalCount = downs.length + wafBlocks.length + otherItems.length

  return (
    <div className="panel attention-panel">
      <div className="panel-head">
        <h2>
          <AlertTriangle size={18} style={{ verticalAlign: 'text-bottom', marginRight: 6 }} />
          需要关注
        </h2>
        <span className="chart-hint">{totalCount === 0 ? '当前无告警项' : `${totalCount} 项`}</span>
      </div>
      <div className="panel-body attention-sections">
        <AttentionSection
          title="健康检查"
          icon={<HeartPulse size={16} />}
          href="/healths"
          emptyText="全部后端探测正常"
          emptyOk
        >
          {downs.length === 0 ? null : (
            <ul className="attention-list">
              {downs.map((h) => (
                <AttentionRow
                  key={h.key}
                  level="danger"
                  title={`DOWN · ${h.host}`}
                  detail={h.error || h.url || h.backend}
                  href="/healths"
                />
              ))}
            </ul>
          )}
        </AttentionSection>

        <AttentionSection
          title="WAF 拦截"
          icon={<ShieldBan size={16} />}
          href="/waf"
          emptyText="近期无 block 事件"
          emptyOk
        >
          {wafBlocks.length === 0 ? null : (
            <ul className="attention-list">
              {wafBlocks.map((e) => (
                <AttentionRow
                  key={e.id}
                  level="warn"
                  title={`block · ${e.rule}`}
                  detail={`${e.host}${e.path}`}
                  href="/waf"
                />
              ))}
            </ul>
          )}
        </AttentionSection>

        {otherItems.length > 0 ? (
          <AttentionSection title="其他" icon={<FileWarning size={16} />}>
            <ul className="attention-list">
              {otherItems.map((item, i) => (
                <AttentionRow
                  key={`${item.title}-${i}`}
                  level={item.level}
                  title={item.title}
                  detail={item.detail}
                  href={item.href}
                  icon={
                    item.icon === 'slow' ? (
                      <Clock size={14} />
                    ) : item.icon === 'cert' ? (
                      <AlertTriangle size={14} />
                    ) : undefined
                  }
                />
              ))}
            </ul>
          </AttentionSection>
        ) : null}

        {totalCount === 0 ? (
          <p className="empty-hint ok-hint">各项指标正常，暂无需要立即处理的事项。</p>
        ) : null}
      </div>
    </div>
  )
})

function AttentionSection({
  title,
  icon,
  href,
  emptyText,
  emptyOk,
  children,
}: {
  title: string
  icon: ReactNode
  href?: string
  emptyText?: string
  emptyOk?: boolean
  children: ReactNode
}) {
  const hasContent = children != null
  return (
    <section className="attention-section">
      <div className="attention-section-head">
        {icon}
        <h3>{title}</h3>
        {href ? (
          <Link to={href} className="btn btn-ghost btn-sm">
            查看全部
          </Link>
        ) : null}
      </div>
      {hasContent ? (
        children
      ) : emptyText ? (
        <p className={emptyOk ? 'empty-hint ok-hint' : 'empty-hint'}>{emptyText}</p>
      ) : null}
    </section>
  )
}

function AttentionRow({
  level,
  title,
  detail,
  href,
  icon,
}: {
  level: 'danger' | 'warn' | 'info'
  title: string
  detail: string
  href?: string
  icon?: ReactNode
}) {
  return (
    <li className={`attention-item attention-${level}`}>
      <div className="attention-main">
        <span className={`attention-dot attention-dot-${level}`} />
        {icon ? <span className="attention-icon">{icon}</span> : null}
        <div>
          <div className="attention-title">{title}</div>
          <div className="attention-detail">{detail}</div>
        </div>
      </div>
      {href ? (
        <Link to={href} className="btn btn-ghost btn-sm">
          查看
        </Link>
      ) : null}
    </li>
  )
}

function buildOtherAttentionItems(
  metrics: OverviewMetrics | null,
  certs: TLSCert[],
  wafBlocks: WAFEvent[],
): AttentionItem[] {
  const out: AttentionItem[] = []
  void wafBlocks

  const criticalCerts = certs.filter((c) => c.days_remaining < 7)
  const warnCerts = certs.filter((c) => c.days_remaining >= 7 && c.days_remaining < 30)
  for (const c of criticalCerts.slice(0, 2)) {
    out.push({
      level: 'danger',
      title: `证书即将过期：${c.domain}`,
      detail: `剩余 ${c.days_remaining} 天`,
      href: '/tls',
      icon: 'cert',
    })
  }
  for (const c of warnCerts.slice(0, Math.max(0, 2 - criticalCerts.length))) {
    out.push({
      level: 'warn',
      title: `证书需关注：${c.domain}`,
      detail: `剩余 ${c.days_remaining} 天`,
      href: '/tls',
      icon: 'cert',
    })
  }

  if (metrics && metrics.error_rate > 10) {
    out.push({
      level: 'danger',
      title: '错误率偏高',
      detail: `${metrics.error_rate.toFixed(1)}%（窗口 ${metrics.window}）`,
      href: '/logs',
      icon: 'error',
    })
  } else if (metrics && metrics.error_rate > 5) {
    out.push({
      level: 'warn',
      title: '错误率上升',
      detail: `${metrics.error_rate.toFixed(1)}%`,
      href: '/logs',
      icon: 'error',
    })
  }

  if (metrics && metrics.p95_ms > 2000) {
    out.push({
      level: 'warn',
      title: 'P95 延迟较高',
      detail: formatMs(metrics.p95_ms),
      href: '/logs',
      icon: 'slow',
    })
  }

  if (metrics) {
    for (const s of metrics.slowest.slice(0, 2)) {
      if (s.duration_ms < 1000) break
      out.push({
        level: 'warn',
        title: `慢请求 ${formatMs(s.duration_ms)}`,
        detail: `${s.host} ${s.method} ${s.path}`,
        href: '/logs',
        icon: 'slow',
      })
    }
  }

  return out.slice(0, 8)
}

function formatMs(ms: number) {
  if (ms >= 1000) return `${(ms / 1000).toFixed(2)}s`
  return `${Math.round(ms)}ms`
}
