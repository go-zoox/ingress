import { useCallback, useEffect, useMemo, useState } from 'react'
import { Activity, CheckSquare, Square } from 'lucide-react'
import { PageHeader } from '../components/PageHeader'
import { ParseIssueDetailDrawer } from '../components/ParseIssueDetailDrawer'
import { WafEventDetailDrawer } from '../components/WafEventDetailDrawer'
import { api, type EventsTabSummary } from '../api/client'
import {
  buildAdminEvents,
  OPEN_EVENTS_LIST_LIMIT,
  kindLabel,
  type AdminEvent,
  type EventStatusTab,
} from '../lib/adminEvents'

const STATUS_TABS: { id: EventStatusTab; label: string }[] = [
  { id: 'open', label: '待处理' },
  { id: 'resolved', label: '已处理' },
  { id: 'ignored', label: '忽略' },
]

export function EventsPage() {
  const [statusTab, setStatusTab] = useState<EventStatusTab>('open')
  const [events, setEvents] = useState<AdminEvent[]>([])
  const [summary, setSummary] = useState<EventsTabSummary | null>(null)
  const [loading, setLoading] = useState(true)
  const [err, setErr] = useState('')
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [batchNote, setBatchNote] = useState('')
  const [batchBusy, setBatchBusy] = useState(false)

  const [wafDrawerId, setWafDrawerId] = useState<number | null>(null)
  const [parseDrawerId, setParseDrawerId] = useState<number | null>(null)

  const load = useCallback(() => {
    setLoading(true)
    setErr('')
    const wafStatus = statusTab === 'open' ? 'open' : statusTab
    const parseStatus = statusTab
    return Promise.all([
      api.wafEvents({ action: 'block', status: wafStatus, limit: OPEN_EVENTS_LIST_LIMIT }).catch(() => []),
      api.parseIssues(parseStatus, OPEN_EVENTS_LIST_LIMIT).catch(() => []),
      api.eventsSummary(statusTab).catch(() => null),
      statusTab === 'open'
        ? api.healthCheck().catch(() => ({ checks: [], summary: { total: 0, up: 0, down: 0, unknown: 0 } }))
        : Promise.resolve({ checks: [] as Awaited<ReturnType<typeof api.healthCheck>>['checks'] }),
      statusTab === 'open' ? api.tlsCerts().catch(() => []) : Promise.resolve([]),
    ])
      .then(([waf, parseIssues, tabSummary, health, certs]) => {
        const rows = buildAdminEvents({
          statusTab,
          wafEvents: Array.isArray(waf) ? waf : [],
          parseIssues: Array.isArray(parseIssues) ? parseIssues : [],
          healthChecks: health.checks || [],
          certs: Array.isArray(certs) ? certs : [],
        })
        setEvents(rows)
        setSummary(tabSummary)
        setSelected(new Set())
        setLoading(false)
      })
      .catch((e: Error) => {
        setErr(e.message)
        setLoading(false)
      })
  }, [statusTab])

  useEffect(() => {
    load()
    const t = window.setInterval(load, 60_000)
    return () => window.clearInterval(t)
  }, [load])

  const actionable = useMemo(() => events.filter((e) => e.actionable), [events])
  const selectedActionable = useMemo(
    () => actionable.filter((e) => selected.has(e.key)),
    [actionable, selected],
  )

  const toggleSelect = (key: string) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(key)) next.delete(key)
      else next.add(key)
      return next
    })
  }

  const toggleSelectAll = () => {
    if (selectedActionable.length === actionable.length && actionable.length > 0) {
      setSelected(new Set())
      return
    }
    setSelected(new Set(actionable.map((e) => e.key)))
  }

  const runBatch = async (status: 'resolved' | 'ignored', keys: string[]) => {
    const targets = events.filter((e) => keys.includes(e.key) && e.actionable && e.entityId != null)
    if (targets.length === 0) return
    const wafIds = targets.filter((e) => e.kind === 'waf').map((e) => e.entityId!)
    const parseIds = targets.filter((e) => e.kind === 'parse').map((e) => e.entityId!)
    setBatchBusy(true)
    try {
      await Promise.all([
        wafIds.length ? api.batchUpdateWafEventStatus(wafIds, status, batchNote) : Promise.resolve(),
        parseIds.length ? api.batchUpdateParseIssueStatus(parseIds, status, batchNote) : Promise.resolve(),
      ])
      setBatchNote('')
      await load()
    } catch (e) {
      setErr(e instanceof Error ? e.message : '批量操作失败')
    } finally {
      setBatchBusy(false)
    }
  }

  const runBatchAllOpen = async (status: 'resolved' | 'ignored') => {
    setBatchBusy(true)
    try {
      await Promise.all([
        api.batchUpdateWafEventStatus([], status, batchNote, { allOpen: true }),
        api.batchUpdateParseIssueStatus([], status, batchNote, { allOpen: true }),
      ])
      setBatchNote('')
      await load()
    } catch (e) {
      setErr(e instanceof Error ? e.message : '全部操作失败')
    } finally {
      setBatchBusy(false)
    }
  }

  const openDetail = (event: AdminEvent) => {
    if (event.kind === 'waf' && event.entityId != null) {
      setParseDrawerId(null)
      setWafDrawerId(event.entityId)
      return
    }
    if (event.kind === 'parse' && event.entityId != null) {
      setWafDrawerId(null)
      setParseDrawerId(event.entityId)
    }
  }

  const showBatchBar = statusTab === 'open' && (summary?.total ?? actionable.length) > 0

  const countLabel = useMemo(() => {
    if (summary == null) return `${events.length} 条`
    if (events.length < summary.total) {
      return `共 ${summary.total} 条 · 本页 ${events.length}`
    }
    return `共 ${summary.total} 条`
  }, [summary, events.length])

  const actionableTotal = useMemo(() => {
    if (summary == null) return actionable.length
    return summary.waf_block + summary.parse_issues
  }, [summary, actionable.length])

  return (
    <div className="page">
      <PageHeader
        title="事件"
        desc="待处理、已处理与已忽略；点击处理打开详情弹窗，可批量或全部（数据库内全部待处理）标记"
        actions={
          <button type="button" className="btn btn-sm" onClick={() => load()} disabled={loading}>
            刷新
          </button>
        }
      />
      {err ? <p className="err">{err}</p> : null}

      <div className="events-status-tabs" role="tablist" aria-label="事件状态">
        {STATUS_TABS.map((tab) => (
          <button
            key={tab.id}
            type="button"
            role="tab"
            aria-selected={statusTab === tab.id}
            className={`events-status-tab${statusTab === tab.id ? ' active' : ''}`}
            onClick={() => setStatusTab(tab.id)}
          >
            {tab.label}
          </button>
        ))}
        <span className="chart-hint">{countLabel}</span>
      </div>

      {showBatchBar ? (
        <div className="events-batch-bar panel">
          <div className="events-batch-select">
            <button type="button" className="btn btn-ghost btn-sm" onClick={toggleSelectAll} aria-label="全选">
              {selectedActionable.length === actionable.length && actionable.length > 0 ? (
                <CheckSquare size={16} />
              ) : (
                <Square size={16} />
              )}
            </button>
            <span className="chart-hint">
              已选 {selectedActionable.length} / 本页 {actionable.length} 条可处置
              {actionableTotal > actionable.length ? ` · 共 ${actionableTotal} 条待处理` : ''}
            </span>
          </div>
          <label className="events-batch-note">
            <span className="sr-only">批量备注</span>
            <input
              type="text"
              placeholder="备注（可选，批量时共用）"
              value={batchNote}
              onChange={(e) => setBatchNote(e.target.value)}
              disabled={batchBusy}
            />
          </label>
          <div className="events-batch-actions">
            <button
              type="button"
              className="btn btn-sm"
              disabled={batchBusy || selectedActionable.length === 0}
              onClick={() => runBatch('resolved', [...selectedActionable.map((e) => e.key)])}
            >
              批量已处理
            </button>
            <button
              type="button"
              className="btn btn-ghost btn-sm"
              disabled={batchBusy || selectedActionable.length === 0}
              onClick={() => runBatch('ignored', [...selectedActionable.map((e) => e.key)])}
            >
              批量忽略
            </button>
            <button
              type="button"
              className="btn btn-sm"
              disabled={batchBusy}
              onClick={() => runBatchAllOpen('resolved')}
            >
              全部已处理
            </button>
            <button
              type="button"
              className="btn btn-ghost btn-sm"
              disabled={batchBusy}
              onClick={() => runBatchAllOpen('ignored')}
            >
              全部忽略
            </button>
          </div>
        </div>
      ) : null}

      <div className="panel">
        <div className="panel-body">
          {loading ? (
            <p className="empty-hint">加载中…</p>
          ) : events.length === 0 ? (
            <p className="empty-hint ok-hint">
              <Activity size={16} style={{ verticalAlign: 'middle', marginRight: 6 }} />
              {statusTab === 'open' ? '当前无待处理事件' : '暂无记录'}
            </p>
          ) : (
            <ul className={`events-feed${showBatchBar ? ' events-feed--selectable' : ''}`}>
              {events.map((e) => {
                const canOpen = (e.kind === 'waf' || e.kind === 'parse') && e.entityId != null
                const showCheck = showBatchBar && e.actionable
                return (
                  <li
                    key={e.key}
                    className={`events-feed-item events-${e.level}${showBatchBar ? ' events-feed-item--selectable' : ''}${e.actionable ? ' events-feed-item--actionable' : ''}`}
                  >
                    {showBatchBar ? (
                      <span className="events-feed-check-slot" aria-hidden={!showCheck}>
                        {showCheck ? (
                          <label className="events-feed-check">
                            <input
                              type="checkbox"
                              checked={selected.has(e.key)}
                              onChange={() => toggleSelect(e.key)}
                              onClick={(ev) => ev.stopPropagation()}
                            />
                          </label>
                        ) : null}
                      </span>
                    ) : null}
                    <div
                      className={`events-feed-main${canOpen ? ' events-feed-main--clickable' : ''}`}
                      role={canOpen ? 'button' : undefined}
                      tabIndex={canOpen ? 0 : undefined}
                      onClick={canOpen ? () => openDetail(e) : undefined}
                      onKeyDown={
                        canOpen
                          ? (ev) => {
                              if (ev.key === 'Enter' || ev.key === ' ') {
                                ev.preventDefault()
                                openDetail(e)
                              }
                            }
                          : undefined
                      }
                    >
                      <span className={`events-kind events-kind-${e.kind}`}>{kindLabel(e.kind)}</span>
                      <div className="events-feed-body">
                        <div className="events-feed-title">{e.title}</div>
                        {e.kind === 'waf' ? (
                          <code className="events-feed-path">{e.detail}</code>
                        ) : (
                          <div className="events-feed-detail">{e.detail}</div>
                        )}
                        {e.note ? <div className="events-feed-note">备注：{e.note}</div> : null}
                        {e.time ? (
                          <time className="events-feed-time">{formatTime(e.time)}</time>
                        ) : null}
                      </div>
                    </div>
                    <div className="events-feed-actions">
                      {canOpen ? (
                        <button type="button" className="btn btn-ghost btn-sm" onClick={() => openDetail(e)}>
                          {statusTab === 'open' ? '处理' : '详情'}
                        </button>
                      ) : null}
                    </div>
                  </li>
                )
              })}
            </ul>
          )}
        </div>
      </div>

      <WafEventDetailDrawer
        eventId={wafDrawerId}
        open={wafDrawerId != null}
        onClose={() => setWafDrawerId(null)}
        onStatusChange={() => load()}
        variant="triage"
      />

      <ParseIssueDetailDrawer
        issueId={parseDrawerId}
        open={parseDrawerId != null}
        onClose={() => setParseDrawerId(null)}
        onStatusChange={() => load()}
      />
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
