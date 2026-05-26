import { useEffect, useMemo, useState } from 'react'
import { api, type AuditLogRow, type ConfigRevisionSummary } from '../api/client'

type TimelineItem = {
  id: string
  kind: 'revision' | 'audit'
  time: string
  title: string
  detail: string
  hash?: string
  revisionId?: number
}

export function ConfigChangeTimeline({
  onRestore,
  onRollback,
}: {
  onRestore?: (content: string) => void
  onRollback?: (revision: ConfigRevisionSummary) => void
}) {
  const [revisions, setRevisions] = useState<ConfigRevisionSummary[]>([])
  const [audits, setAudits] = useState<AuditLogRow[]>([])
  const [err, setErr] = useState('')
  const [loading, setLoading] = useState(true)

  const load = () => {
    setLoading(true)
    Promise.all([api.configRevisions(30), api.auditLogs(40)])
      .then(([revs, logs]) => {
        setRevisions(Array.isArray(revs) ? revs : [])
        setAudits(Array.isArray(logs) ? logs : [])
        setErr('')
      })
      .catch((e: Error) => setErr(e.message))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    load()
  }, [])

  const items = useMemo(() => {
    const out: TimelineItem[] = []
    for (const r of revisions) {
      out.push({
        id: `rev-${r.id}`,
        kind: 'revision',
        time: r.created_at,
        title: `发布 #${r.id}`,
        detail: r.note || 'publish',
        hash: r.hash,
        revisionId: r.id,
      })
    }
    for (const a of audits) {
      out.push({
        id: `audit-${a.id}`,
        kind: 'audit',
        time: a.created_at,
        title: auditActionLabel(a.action),
        detail: a.detail || a.actor || '—',
      })
    }
    out.sort((a, b) => new Date(b.time).getTime() - new Date(a.time).getTime())
    return out.slice(0, 40)
  }, [revisions, audits])

  return (
    <div className="config-change-timeline">
      <div className="config-change-timeline-head">
        <h3>变更时间线</h3>
        <button type="button" className="btn btn-sm btn-ghost" onClick={load}>
          刷新
        </button>
      </div>
      {err ? <p className="err">{err}</p> : null}
      {loading ? (
        <p className="empty-hint">加载中…</p>
      ) : items.length === 0 ? (
        <p className="empty-hint">暂无发布记录或审计日志</p>
      ) : (
        <ul className="config-timeline-list">
          {items.map((it) => (
            <li key={it.id} className={`config-timeline-item config-timeline-${it.kind}`}>
              <time>{formatTime(it.time)}</time>
              <div>
                <div className="config-timeline-title">{it.title}</div>
                <div className="config-timeline-detail">
                  {it.hash ? <code>{it.hash}</code> : null}
                  {it.hash ? ' · ' : null}
                  {it.detail}
                </div>
              </div>
              {it.kind === 'revision' && it.revisionId != null ? (
                <div className="config-timeline-actions">
                  <button
                    type="button"
                    className="action-link"
                    onClick={async () => {
                      const row = await api.configRevision(it.revisionId!)
                      onRestore?.(row.content)
                    }}
                  >
                    载入草稿
                  </button>
                  {onRollback ? (
                    <button
                      type="button"
                      className="action-link action-danger"
                      onClick={() => {
                        const rev = revisions.find((r) => r.id === it.revisionId)
                        if (rev) onRollback(rev)
                      }}
                    >
                      回滚
                    </button>
                  ) : null}
                </div>
              ) : null}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

function auditActionLabel(action: string) {
  switch (action) {
    case 'config.save':
      return '保存配置'
    case 'ingress.reload':
      return '热加载'
    default:
      return action
  }
}

function formatTime(iso: string) {
  try {
    return new Date(iso).toLocaleString('zh-CN', { hour12: false })
  } catch {
    return iso
  }
}
