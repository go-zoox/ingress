import { memo, useState, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import {
  AlertTriangle,
  HeartPulse,
  ShieldBan,
  Clock,
  FileWarning,
} from 'lucide-react'
import type { OverviewMetrics, TLSCert, HealthCheckResult, WAFEvent, WAFEventDetail, AccessLogParseIssue } from '../api/client'
import { healthLink, investigateLink, logsLink, wafLink } from '../lib/deepLinks'
import { ParseIssueDetailDrawer } from './ParseIssueDetailDrawer'
import { WafEventDetailDrawer } from './WafEventDetailDrawer'
import { WafTrialDrawer } from './WafTrialDrawer'

export type AttentionAction = { label: string; href: string }
export type AttentionButton = { label: string; onClick: () => void }

export type AttentionItem = {
  level: 'danger' | 'warn' | 'info'
  title: string
  detail: string
  href?: string
  actions?: AttentionAction[]
  icon?: 'cert' | 'error' | 'slow'
}

type Props = {
  metrics: OverviewMetrics | null
  certs: TLSCert[]
  healthChecks: HealthCheckResult[]
  wafBlocks: WAFEvent[]
  parseIssues?: AccessLogParseIssue[]
  onParseIssueStatus?: (id: number, status: 'ignored' | 'resolved') => void
  onWafEventStatus?: (id: number, status: 'ignored' | 'resolved') => void
  embedded?: boolean
}

export const OverviewAttentionPanel = memo(function OverviewAttentionPanel({
  metrics,
  certs,
  healthChecks,
  wafBlocks,
  parseIssues = [],
  onParseIssueStatus,
  onWafEventStatus,
  embedded = true,
}: Props) {
  const [parseIssueId, setParseIssueId] = useState<number | null>(null)
  const [wafEventId, setWafEventId] = useState<number | null>(null)
  const [wafTrialEvent, setWafTrialEvent] = useState<WAFEvent | WAFEventDetail | null>(null)

  const openWafTrial = (event: WAFEvent | WAFEventDetail) => {
    setWafEventId(null)
    setWafTrialEvent(event)
  }
  const downs = healthChecks.filter((h) => h.status === 'down')
  const otherItems = buildOtherAttentionItems(metrics, certs)
  const totalCount = downs.length + wafBlocks.length + otherItems.length + parseIssues.length

  return (
    <div className={`panel attention-panel${embedded ? '' : ' attention-panel-page'}`}>
      {embedded ? (
        <div className="panel-head">
          <h2>
            <AlertTriangle size={18} style={{ verticalAlign: 'text-bottom', marginRight: 6 }} />
            需要关注
          </h2>
          <Link to="/attention" className="btn btn-ghost btn-sm">
            查看全部
          </Link>
        </div>
      ) : null}
      <div className="panel-body attention-sections">
        <AttentionSection
          title="健康检查"
          icon={<HeartPulse size={16} />}
          href={healthLink({ status: 'down' })}
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
                  href={healthLink({ status: 'down', host: h.host })}
                  actions={[
                    { label: '调查', href: investigateLink({ host: h.host, path: h.path || '/' }) },
                    { label: '查日志', href: logsLink({ host: h.host, log: 'access' }) },
                  ]}
                />
              ))}
            </ul>
          )}
        </AttentionSection>

        <AttentionSection
          title="WAF 拦截"
          icon={<ShieldBan size={16} />}
          href={wafLink({ action: 'block' })}
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
                  onView={() => setWafEventId(e.id)}
                />
              ))}
            </ul>
          )}
        </AttentionSection>

        <AttentionSection
          title="日志解析"
          icon={<FileWarning size={16} />}
          emptyText="access.log 解析正常"
          emptyOk
        >
          {parseIssues.length === 0 ? null : (
            <ul className="attention-list">
              {parseIssues.map((issue) => (
                <AttentionRow
                  key={issue.id}
                  level="warn"
                  title={`无法解析 · ${parseIssueReasonLabel(issue.reason)}`}
                  detail={`${issue.hit_count} 次 · ${truncateIssueLine(issue.sample_line)}`}
                  onView={() => setParseIssueId(issue.id)}
                  buttons={[
                    {
                      label: '已处理',
                      onClick: () => onParseIssueStatus?.(issue.id, 'resolved'),
                    },
                    {
                      label: '忽略',
                      onClick: () => onParseIssueStatus?.(issue.id, 'ignored'),
                    },
                  ]}
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
                  actions={item.actions}
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

      <WafEventDetailDrawer
        eventId={wafEventId}
        open={wafEventId != null}
        onClose={() => setWafEventId(null)}
        onStatusChange={onWafEventStatus}
        onTrial={(detail) => openWafTrial(detail)}
      />

      <WafTrialDrawer
        open={wafTrialEvent != null}
        eventId={wafTrialEvent?.id}
        seed={wafTrialEvent}
        onClose={() => setWafTrialEvent(null)}
      />

      <ParseIssueDetailDrawer
        issueId={parseIssueId}
        open={parseIssueId != null}
        onClose={() => setParseIssueId(null)}
        onStatusChange={onParseIssueStatus}
      />
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
  onView,
  actions,
  buttons,
  icon,
}: {
  level: 'danger' | 'warn' | 'info'
  title: string
  detail: string
  href?: string
  onView?: () => void
  actions?: AttentionAction[]
  buttons?: AttentionButton[]
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
      <div className="attention-actions">
        {onView ? (
          <button type="button" className="btn btn-ghost btn-sm" onClick={onView}>
            查看
          </button>
        ) : href ? (
          <Link to={href} className="btn btn-ghost btn-sm">
            查看
          </Link>
        ) : null}
        {actions?.map((a) => (
          <Link key={a.href} to={a.href} className="btn btn-ghost btn-sm">
            {a.label}
          </Link>
        ))}
        {buttons?.map((b) => (
          <button key={b.label} type="button" className="btn btn-ghost btn-sm" onClick={b.onClick}>
            {b.label}
          </button>
        ))}
      </div>
    </li>
  )
}

function buildOtherAttentionItems(metrics: OverviewMetrics | null, certs: TLSCert[]): AttentionItem[] {
  const out: AttentionItem[] = []

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
      href: logsLink({ log: 'access', status: '5' }),
      actions: [{ label: '全部日志', href: logsLink({ log: 'access' }) }],
      icon: 'error',
    })
  } else if (metrics && metrics.error_rate > 5) {
    out.push({
      level: 'warn',
      title: '错误率上升',
      detail: `${metrics.error_rate.toFixed(1)}%`,
      href: logsLink({ log: 'access' }),
      icon: 'error',
    })
  }

  if (metrics && metrics.p95_ms > 2000) {
    out.push({
      level: 'warn',
      title: 'P95 延迟较高',
      detail: formatMs(metrics.p95_ms),
      href: logsLink({ log: 'access' }),
      icon: 'slow',
    })
  }

  if (metrics) {
    for (const s of (metrics.slowest ?? []).slice(0, 2)) {
      if (s.duration_ms < 1000) break
      out.push({
        level: 'warn',
        title: `慢请求 ${formatMs(s.duration_ms)}`,
        detail: `${s.host} ${s.method} ${s.path}`,
        href: investigateLink({
          host: s.host,
          path: s.path,
          method: s.method,
          status: s.status,
        }),
        actions: [{ label: '查日志', href: logsLink({ host: s.host, path: s.path, log: 'access' }) }],
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

function parseIssueReasonLabel(reason: string) {
  switch (reason) {
    case 'missing_host':
      return '缺少 host'
    case 'missing_request':
      return '缺少 HTTP 请求段'
    case 'empty_after_prefix':
      return '去掉前缀后为空'
    default:
      return '格式不兼容'
  }
}

function truncateIssueLine(line: string) {
  const trimmed = String(line || '').trim()
  if (trimmed.length <= 120) return trimmed || '—'
  return `${trimmed.slice(0, 120)}…`
}
