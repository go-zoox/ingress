import { useCallback, useEffect, useRef, useState } from 'react'
import { PageHeader } from '../components/PageHeader'
import { api, type WAFEvent } from '../api/client'

export function WAFPage() {
  const [status, setStatus] = useState<Record<string, unknown> | null>(null)
  const [events, setEvents] = useState<WAFEvent[]>([])
  const [err, setErr] = useState('')

  // filter state
  const [filterAction, setFilterAction] = useState('all')
  const [filterHost, setFilterHost] = useState('')
  const [filterPath, setFilterPath] = useState('')
  const [filterClientIP, setFilterClientIP] = useState('')
  const [filterRule, setFilterRule] = useState('')
  const [filterTimeStart, setFilterTimeStart] = useState('')
  const [filterTimeEnd, setFilterTimeEnd] = useState('')

  // dropdown options
  const [wafHosts, setWafHosts] = useState<string[]>([])
  const [wafRules, setWafRules] = useState<string[]>([])

  // real-time polling
  const [realtime, setRealtime] = useState(false)
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const load = useCallback(() => {
    const params: Parameters<typeof api.wafEvents>[0] = {}
    if (filterAction !== 'all') params.action = filterAction
    if (filterHost) params.host = filterHost
    if (filterPath.trim()) params.path = filterPath.trim()
    if (filterClientIP.trim()) params.client_ip = filterClientIP.trim()
    if (filterRule) params.rule = filterRule
    if (filterTimeStart) params.time_start = filterTimeStart
    if (filterTimeEnd) params.time_end = filterTimeEnd + 'T23:59:59Z'

    api
      .wafEvents(Object.keys(params).length ? params : undefined)
      .then((data) => setEvents(Array.isArray(data) ? data : []))
      .catch((e: Error) => setErr(e.message))
  }, [filterAction, filterHost, filterPath, filterClientIP, filterRule, filterTimeStart, filterTimeEnd])

  const loadStatus = useCallback(() => {
    api.status().then(setStatus).catch(() => setStatus(null))
  }, [])

  useEffect(() => { loadStatus() }, [loadStatus])
  useEffect(() => { load() }, [load])

  // fetch distinct hosts and rules for dropdown options
  useEffect(() => {
    api.wafHosts().then(setWafHosts).catch(() => setWafHosts([]))
    api.wafRules().then(setWafRules).catch(() => setWafRules([]))
  }, [])

  // real-time polling
  useEffect(() => {
    if (realtime) {
      timerRef.current = setInterval(() => { load(); loadStatus() }, 3000)
    }
    return () => { if (timerRef.current) clearInterval(timerRef.current) }
  }, [realtime, load, loadStatus])

  const handleRefresh = () => { load(); loadStatus() }
  const handleResetFilters = () => {
    setFilterAction('all')
    setFilterHost('')
    setFilterPath('')
    setFilterClientIP('')
    setFilterRule('')
    setFilterTimeStart('')
    setFilterTimeEnd('')
  }

  const handleToggle = async (enabled: boolean) => {
    try {
      await api.wafToggle(enabled ? true : null)
      loadStatus()
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e))
    }
  }

  const runtimeWafEnabled = status?.waf_runtime_enabled !== undefined
    ? Boolean(status.waf_runtime_enabled)
    : Boolean(status?.waf_enabled)
  const configWafEnabled = Boolean(status?.waf_enabled)
  const overrideActive = runtimeWafEnabled !== configWafEnabled

  return (
    <div className="page">
      <PageHeader title="WAF" desc="全局基线配置与近期 block / audit 事件" />
      {err && <p className="err">{err}</p>}

      {/* status cards */}
      <div className="cards">
        <div className="card">
          <div className="label">状态（配置）</div>
          <div className="value">{status?.waf_enabled ? '已启用' : '关闭'}</div>
          <div className="sub">全局 waf.enabled</div>
        </div>
        <div className={`card ${overrideActive ? 'warn' : ''}`}>
          <div className="label">WAF 实时开关</div>
          <div className="value">
            <label className="live-toggle">
              <input
                type="checkbox"
                checked={runtimeWafEnabled}
                onChange={(e) => handleToggle(e.target.checked)}
              />{' '}
              {runtimeWafEnabled ? '已启用' : '已关闭'}
            </label>
          </div>
          <div className="sub">
            {overrideActive ? '运行时覆盖生效，与配置文件不同' : '无需 reload，立即生效'}
          </div>
        </div>
        <div className="card">
          <div className="label">模式</div>
          <div className="value">{status?.waf_log_only ? '仅审计' : '拦截'}</div>
          <div className="sub">log_only</div>
        </div>
      </div>

      {/* filter bar */}
      <div className="panel">
        <div className="panel-head">
          <h2>事件过滤</h2>
          <div className="filter-actions">
            <label className="realtime-toggle">
              <input
                type="checkbox"
                checked={realtime}
                onChange={(e) => setRealtime(e.target.checked)}
              />{' '}
              实时刷新{realtime ? ' (3s)' : ''}
            </label>
            <button className="btn btn-sm" onClick={handleRefresh}>刷新</button>
            <button className="btn btn-sm btn-ghost" onClick={handleResetFilters}>重置</button>
          </div>
        </div>
        <div className="panel-body filter-grid">
          <div className="filter-field">
            <label>动作</label>
            <select value={filterAction} onChange={(e) => setFilterAction(e.target.value)}>
              <option value="all">全部</option>
              <option value="block">Block</option>
              <option value="audit">Audit</option>
            </select>
          </div>
          <div className="filter-field">
            <label>Host</label>
            <select value={filterHost} onChange={(e) => setFilterHost(e.target.value)}>
              <option value="">全部</option>
              {wafHosts.map((h) => (
                <option key={h} value={h}>{h}</option>
              ))}
            </select>
          </div>
          <div className="filter-field">
            <label>Path</label>
            <input
              type="text"
              placeholder="模糊匹配 path"
              value={filterPath}
              onChange={(e) => setFilterPath(e.target.value)}
            />
          </div>
          <div className="filter-field">
            <label>客户端 IP</label>
            <input
              type="text"
              placeholder="模糊匹配 IP"
              value={filterClientIP}
              onChange={(e) => setFilterClientIP(e.target.value)}
            />
          </div>
          <div className="filter-field">
            <label>规则</label>
            <select value={filterRule} onChange={(e) => setFilterRule(e.target.value)}>
              <option value="">全部</option>
              {wafRules.map((r) => (
                <option key={r} value={r}>{r}</option>
              ))}
            </select>
          </div>
          <div className="filter-field">
            <label>开始时间</label>
            <input
              type="date"
              value={filterTimeStart}
              onChange={(e) => setFilterTimeStart(e.target.value)}
            />
          </div>
          <div className="filter-field">
            <label>结束时间</label>
            <input
              type="date"
              value={filterTimeEnd}
              onChange={(e) => setFilterTimeEnd(e.target.value)}
            />
          </div>
        </div>
      </div>

      {/* events table */}
      <div className="panel">
        <div className="panel-head">
          <h2>事件列表 ({events.length})</h2>
        </div>
        <div className="panel-body panel-table-wrap">
          {events.length === 0 ? (
            <p className="empty">暂无事件</p>
          ) : (
            <table className="data">
              <thead>
                <tr>
                  <th className="col-time">时间</th>
                  <th className="col-action">动作</th>
                  <th className="col-rule">规则</th>
                  <th className="col-host">Host</th>
                  <th className="col-path">Path</th>
                  <th className="col-ip">客户端 IP</th>
                </tr>
              </thead>
              <tbody>
                {events.map((e) => (
                  <tr key={e.id}>
                    <td className="col-time">{formatTime(e.created_at)}</td>
                    <td className="col-action">
                      <span className={`badge badge-${e.action}`}>{e.action}</span>
                    </td>
                    <td className="col-rule">{e.rule}</td>
                    <td className="col-host">{e.host}</td>
                    <td className="col-path">
                      <code>{e.path}</code>
                    </td>
                    <td className="col-ip">{e.client_ip}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>
    </div>
  )
}

function formatTime(iso: string) {
  try {
    return new Date(iso).toLocaleString('zh-CN', {
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    })
  } catch {
    return iso
  }
}
