import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { Activity, Filter } from 'lucide-react'
import { PageHeader } from '../components/PageHeader'
import { api } from '../api/client'
import { buildEventsFeed, type FeedEvent } from '../lib/buildEventsFeed'
import { runbookForEvent } from '../lib/eventRunbook'

type KindFilter = 'all' | 'waf' | 'health' | 'tls'

export function EventsPage() {
  const [events, setEvents] = useState<FeedEvent[]>([])
  const [loading, setLoading] = useState(true)
  const [err, setErr] = useState('')
  const [kind, setKind] = useState<KindFilter>('all')

  const load = () => {
    setLoading(true)
    Promise.all([
      api.wafEvents({ action: 'block', limit: 30 }).catch(() => []),
      api.healthCheck().catch(() => ({ checks: [], summary: { total: 0, up: 0, down: 0, unknown: 0 } })),
      api.tlsCerts().catch(() => []),
    ])
      .then(([waf, health, certs]) => {
        setEvents(buildEventsFeed(Array.isArray(waf) ? waf : [], health.checks || [], Array.isArray(certs) ? certs : []))
        setLoading(false)
      })
      .catch((e: Error) => {
        setErr(e.message)
        setLoading(false)
      })
  }

  useEffect(() => {
    load()
    const t = window.setInterval(load, 60_000)
    return () => window.clearInterval(t)
  }, [])

  const filtered = useMemo(() => {
    if (kind === 'all') return events
    return events.filter((e) => e.kind === kind)
  }, [events, kind])

  return (
    <div className="page">
      <PageHeader
        title="事件"
        desc="聚合 WAF 拦截、健康检查 DOWN 与证书告警；含 Runbook 处理建议与调查深链"
        actions={
          <button type="button" className="btn btn-sm" onClick={load}>
            刷新
          </button>
        }
      />
      {err ? <p className="err">{err}</p> : null}

      <div className="events-toolbar">
        <Filter size={14} aria-hidden />
        <select value={kind} onChange={(e) => setKind(e.target.value as KindFilter)}>
          <option value="all">全部类型</option>
          <option value="waf">WAF 拦截</option>
          <option value="health">健康检查</option>
          <option value="tls">TLS 证书</option>
        </select>
        <span className="chart-hint">{filtered.length} 条</span>
      </div>

      <div className="panel">
        <div className="panel-body">
          {loading ? (
            <p className="empty-hint">加载中…</p>
          ) : filtered.length === 0 ? (
            <p className="empty-hint ok-hint">
              <Activity size={16} style={{ verticalAlign: 'middle', marginRight: 6 }} />
              当前无需要处理的事件
            </p>
          ) : (
            <ul className="events-feed">
              {filtered.map((e) => (
                <li key={e.id} className={`events-feed-item events-${e.level}`}>
                  <div className="events-feed-main">
                    <span className={`events-kind events-kind-${e.kind}`}>{kindLabel(e.kind)}</span>
                    <div>
                      <div className="events-feed-title">{e.title}</div>
                      <div className="events-feed-detail">{e.detail}</div>
                      {e.time ? (
                        <time className="events-feed-time">{formatTime(e.time)}</time>
                      ) : null}
                      <ul className="events-runbook">
                        {runbookForEvent(e).map((step, i) => (
                          <li key={i}>
                            {step.href ? (
                              <Link to={step.href}>
                                {step.text}
                                {step.label ? `（${step.label}）` : ''}
                              </Link>
                            ) : (
                              <span>{step.text}</span>
                            )}
                          </li>
                        ))}
                      </ul>
                    </div>
                  </div>
                  <div className="events-feed-actions">
                    <Link to={e.href} className="btn btn-ghost btn-sm">
                      处理
                    </Link>
                    {e.actions?.map((a) => (
                      <Link key={a.href} to={a.href} className="btn btn-ghost btn-sm">
                        {a.label}
                      </Link>
                    ))}
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  )
}

function kindLabel(k: FeedEvent['kind']) {
  switch (k) {
    case 'waf':
      return 'WAF'
    case 'health':
      return '健康'
    case 'tls':
      return 'TLS'
  }
}

function formatTime(iso: string) {
  try {
    return new Date(iso).toLocaleString('zh-CN', { hour12: false })
  } catch {
    return iso
  }
}
