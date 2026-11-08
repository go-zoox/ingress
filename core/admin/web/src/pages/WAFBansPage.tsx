import { useCallback, useEffect, useMemo, useState } from 'react'
import { PageHeader } from '../components/PageHeader'
import { BanDrawer } from '../components/ban/BanDrawer'
import { BanDetailDrawer } from '../components/ban/BanDetailDrawer'
import { api } from '../api/client'
import type { WAFBanRow } from '../lib/wafBan'
import {
  BAN_KIND_LABELS,
  describeBanError,
  describeScope,
  filterBanRows,
  formatRemaining,
  isExpiringSoon,
  summarizeBans,
} from '../lib/wafBan'

type Tab = 'active' | 'expired' | 'audit'

type AuditRow = { id: number; action: string; detail: string; actor: string; created_at: string }

export function WAFBansPage() {
  const [tab, setTab] = useState<Tab>('active')
  const [active, setActive] = useState<WAFBanRow[]>([])
  const [expired, setExpired] = useState<WAFBanRow[]>([])
  const [audit, setAudit] = useState<AuditRow[]>([])
  const [hosts, setHosts] = useState<string[]>([])
  const [keyword, setKeyword] = useState('')
  const [kindFilter, setKindFilter] = useState<'all' | 'ip' | 'ua'>('all')
  const [err, setErr] = useState('')
  const [loading, setLoading] = useState(false)

  const [createOpen, setCreateOpen] = useState(false)
  const [detailId, setDetailId] = useState<number | null>(null)

  const load = useCallback(() => {
    setLoading(true)
    Promise.all([
      api.wafBans({ status: 'active' }),
      api.wafBans({ status: 'expired' }),
      api.wafBans({ status: 'all', limit: 200 }),
    ])
      .then(([a, e]) => {
        setActive(Array.isArray(a) ? a : [])
        setExpired(Array.isArray(e) ? e : [])
        setErr('')
      })
      .catch((e: Error) => setErr(describeBanError(e.message)))
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    load()
    api.wafHosts().then(setHosts).catch(() => setHosts([]))
  }, [load])

  useEffect(() => {
    if (tab !== 'audit') return
    api
      .auditLogs(80)
      .then((rows) => {
        const list = (rows as AuditRow[]).filter((r) => r.action.startsWith('waf.ban.'))
        setAudit(list)
      })
      .catch(() => setAudit([]))
  }, [tab])

  const summary = useMemo(() => summarizeBans(active), [active])
  const rows = useMemo(
    () => filterBanRows(tab === 'expired' ? expired : active, keyword, kindFilter),
    [tab, active, expired, keyword, kindFilter],
  )

  const removeBan = async (row: WAFBanRow) => {
    try {
      await api.deleteWafBan(row.id)
      load()
    } catch (e) {
      setErr(describeBanError(e instanceof Error ? e.message : String(e)))
    }
  }

  return (
    <div className="page">
      <PageHeader
        title="封禁名单"
        desc="运行时生效，无需 reload · 独立于 WAF 总开关，WAF 关闭时仍然生效"
        actions={
          <button type="button" className="btn btn-sm btn-primary" onClick={() => setCreateOpen(true)}>
            新增封禁
          </button>
        }
      />

      {err ? <p className="err">{err}</p> : null}

      <div className="cards">
        <div className="card">
          <div className="label">生效中</div>
          <div className="value">{summary.total}</div>
          <div className="sub">
            IP {summary.ip} · UA {summary.ua}
          </div>
        </div>
        <div className="card warn">
          <div className="label">即将到期</div>
          <div className="value">{summary.expiring}</div>
          <div className="sub">2 小时内，可续期</div>
        </div>
        <div className="card">
          <div className="label">已过期（保留记录）</div>
          <div className="value muted">{expired.length}</div>
          <div className="sub">可一键重新封禁</div>
        </div>
      </div>

      <div className="route-page-tabs page-tabs">
        <button
          type="button"
          className={`btn btn-sm${tab === 'active' ? ' active' : ''}`}
          onClick={() => setTab('active')}
        >
          生效中<span className="tab-count">{active.length}</span>
        </button>
        <button
          type="button"
          className={`btn btn-sm${tab === 'expired' ? ' active' : ''}`}
          onClick={() => setTab('expired')}
        >
          已过期<span className="tab-count">{expired.length}</span>
        </button>
        <button
          type="button"
          className={`btn btn-sm${tab === 'audit' ? ' active' : ''}`}
          onClick={() => setTab('audit')}
        >
          操作审计
        </button>
      </div>

      {tab === 'audit' ? (
        <div className="panel">
          <div className="panel-head">
            <h2>
              封禁操作审计<span className="panel-sub">action: waf.ban.*</span>
            </h2>
          </div>
          <div className="panel-body panel-table-wrap">
            {audit.length === 0 ? (
              <div className="empty">暂无封禁审计记录</div>
            ) : (
              <table className="data">
                <thead>
                  <tr>
                    <th className="col-time">时间</th>
                    <th style={{ width: 150 }}>动作</th>
                    <th>详情</th>
                    <th className="col-time">操作人</th>
                  </tr>
                </thead>
                <tbody>
                  {audit.map((a) => (
                    <tr key={a.id}>
                      <td className="col-time">{new Date(a.created_at).toLocaleString()}</td>
                      <td>
                        <code>{a.action}</code>
                      </td>
                      <td>{a.detail}</td>
                      <td className="col-time">{a.actor || '—'}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>
      ) : (
        <div className="panel">
          <div className="panel-head">
            <h2>{tab === 'active' ? '生效中的封禁' : '已过期封禁'}</h2>
            <div className="filter-actions">
              <input
                type="text"
                className="ban-search-input"
                placeholder="搜索 IP / UA / 原因"
                value={keyword}
                onChange={(e) => setKeyword(e.target.value)}
              />
              <select
                className="ban-search-input"
                value={kindFilter}
                onChange={(e) => setKindFilter(e.target.value as 'all' | 'ip' | 'ua')}
              >
                <option value="all">全部类型</option>
                <option value="ip">IP</option>
                <option value="ua">UA</option>
              </select>
              <button type="button" className="btn btn-sm" onClick={load} disabled={loading}>
                刷新
              </button>
            </div>
          </div>
          <div className="panel-body panel-table-wrap">
            {rows.length === 0 ? (
              <div className="empty">
                {tab === 'active' ? '当前没有生效中的封禁' : '没有匹配的封禁条目'}
              </div>
            ) : (
              <table className="data">
                <thead>
                  <tr>
                    <th style={{ width: 92 }}>类型</th>
                    <th>封禁目标</th>
                    <th style={{ width: 130 }}>生效范围</th>
                    <th style={{ width: 140 }}>到期</th>
                    <th style={{ width: 92 }}>命中</th>
                    <th className="col-time">操作人</th>
                    <th className="col-actions">操作</th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((row) => {
                    const soon = isExpiringSoon(row.expires_at)
                    return (
                      <tr key={row.id}>
                        <td>
                          <span className="badge badge-kind">{BAN_KIND_LABELS[row.kind]}</span>
                        </td>
                        <td className="ban-value-cell">
                          {row.value}
                          <div className="sub-line">{row.reason || '—'}</div>
                        </td>
                        <td>
                          {row.hosts.length === 0 ? (
                            <span className="badge badge-muted">全局</span>
                          ) : (
                            <>
                              <span className="badge badge-warn">指定 Host</span>
                              <div className="sub-line">{describeScope(row)}</div>
                            </>
                          )}
                        </td>
                        <td className={soon ? 'text-warn' : ''}>
                          {row.expires_at ? new Date(row.expires_at).toLocaleString() : '永久'}
                          <div className="sub-line">{formatRemaining(row.expires_at)}</div>
                        </td>
                        <td className="mono">{row.hits.toLocaleString()}</td>
                        <td className="col-time">{row.actor || '—'}</td>
                        <td className="col-actions">
                          <button
                            type="button"
                            className="btn btn-ghost btn-sm"
                            onClick={() => setDetailId(row.id)}
                          >
                            详情
                          </button>
                          <button
                            type="button"
                            className="btn btn-sm btn-danger"
                            onClick={() => removeBan(row)}
                          >
                            解封
                          </button>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            )}
          </div>
        </div>
      )}

      <BanDrawer
        open={createOpen}
        mode="create"
        seed={null}
        availableHosts={hosts}
        onClose={() => setCreateOpen(false)}
        onCreated={load}
      />

      <BanDetailDrawer
        open={detailId != null}
        banId={detailId}
        onClose={() => setDetailId(null)}
        onChanged={load}
      />
    </div>
  )
}
