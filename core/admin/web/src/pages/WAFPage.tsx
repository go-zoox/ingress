import { useCallback, useEffect, useRef, useState } from 'react'
import { FlaskConical } from 'lucide-react'
import { PageHeader } from '../components/PageHeader'
import { EmptyStateGuide } from '../components/EmptyStateGuide'
import { WafEventDetailDrawer } from '../components/WafEventDetailDrawer'
import { WafTrialDrawer } from '../components/WafTrialDrawer'
import { WafRuleTooltip } from '../components/WafRuleTooltip'
import { api, type WAFEvent, type WAFEventDetail } from '../api/client'
import { useSSE } from '../hooks/useSSE'
import { useWafRuleLookup } from '../hooks/useWafRuleLookup'
import { formatWafRuleTooltip, resolveWafRule } from '../lib/wafRuleTooltip'

type DrawerMode = 'detail' | 'trial' | null

function wafFiltersFromLocation() {
  const sp = new URLSearchParams(window.location.search)
  return {
    action: sp.get('action') || 'all',
    host: sp.get('host') || '',
    path: sp.get('path') || '',
    rule: sp.get('rule') || '',
    trial: sp.get('trial') === '1',
    eventId: sp.get('event_id'),
  }
}

export function WAFPage() {
  const urlInit = wafFiltersFromLocation()
  const urlInitRef = useRef(false)
  const [status, setStatus] = useState<Record<string, unknown> | null>(null)
  const [events, setEvents] = useState<WAFEvent[]>([])
  const [err, setErr] = useState('')
  const [drawer, setDrawer] = useState<DrawerMode>(null)
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [trialSeed, setTrialSeed] = useState<WAFEvent | WAFEventDetail | null>(null)

  const [filterAction, setFilterAction] = useState(urlInit.action)
  const [filterHost, setFilterHost] = useState(urlInit.host)
  const [filterPath, setFilterPath] = useState(urlInit.path)
  const [filterClientIP, setFilterClientIP] = useState('')
  const [filterRule, setFilterRule] = useState(urlInit.rule)
  const [filterTimeStart, setFilterTimeStart] = useState('')
  const [filterTimeEnd, setFilterTimeEnd] = useState('')

  const [wafHosts, setWafHosts] = useState<string[]>([])
  const [wafRules, setWafRules] = useState<string[]>([])

  const [realtime, setRealtime] = useState(false)
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const { data: sseData, connected: sseConnected } = useSSE(['waf'])
  const { lookup: ruleLookup } = useWafRuleLookup()

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

  const openDetail = (id: number) => {
    setSelectedId(id)
    setDrawer('detail')
  }

  const openTrial = (seed?: WAFEvent | WAFEventDetail) => {
    setTrialSeed(seed ?? null)
    setDrawer('trial')
  }

  const closeDrawer = () => {
    setDrawer(null)
    setSelectedId(null)
    setTrialSeed(null)
  }

  useEffect(() => {
    if (urlInitRef.current) return
    urlInitRef.current = true
    const { host, path, rule, trial, eventId } = urlInit
    if (!trial && !eventId) return
    const seed: Partial<WAFEvent> = {
      host: host || 'api.example.com',
      path: path || '/search?q=test',
      client_ip: '',
      rule: rule || '',
      id: Number(eventId) || 0,
      action: 'block',
      created_at: '',
    }
    if (eventId) {
      const id = Number(eventId)
      if (!Number.isNaN(id) && id > 0) {
        api
          .wafEvent(id)
          .then((ev) => openTrial(ev))
          .catch(() => openTrial(seed as WAFEvent))
        return
      }
    }
    openTrial(seed as WAFEvent)
  }, [])

  useEffect(() => {
    loadStatus()
  }, [loadStatus])
  useEffect(() => {
    load()
  }, [load])

  useEffect(() => {
    api.wafHosts().then(setWafHosts).catch(() => setWafHosts([]))
    api.wafRules().then(setWafRules).catch(() => setWafRules([]))
  }, [])

  useEffect(() => {
    if (!realtime || sseConnected) {
      if (timerRef.current) clearInterval(timerRef.current)
      timerRef.current = null
      return
    }
    timerRef.current = setInterval(() => {
      load()
      loadStatus()
    }, 3000)
    return () => {
      if (timerRef.current) clearInterval(timerRef.current)
    }
  }, [realtime, load, loadStatus, sseConnected])

  useEffect(() => {
    if (!realtime || !sseData.waf) return
    const incoming = sseData.waf as WAFEvent | WAFEvent[]
    const batch = Array.isArray(incoming) ? incoming : [incoming]
    setEvents((prev) => {
      const existingIds = new Set(prev.map((e) => e.id))
      const fresh = batch.filter((e) => e?.id && !existingIds.has(e.id))
      if (fresh.length === 0) return prev
      return [...fresh, ...prev].slice(0, 200)
    })
  }, [sseData.waf, realtime])

  const handleRefresh = () => {
    load()
    loadStatus()
  }
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

  const runtimeWafEnabled =
    status?.waf_runtime_enabled !== undefined
      ? Boolean(status.waf_runtime_enabled)
      : Boolean(status?.waf_enabled)
  const configWafEnabled = Boolean(status?.waf_enabled)
  const overrideActive = runtimeWafEnabled !== configWafEnabled

  return (
    <div className="page">
      <PageHeader
        title="WAF"
        desc="全局规则、运行时开关与 block/audit 事件；在操作列打开详情"
        actions={
          <button type="button" className="btn btn-sm btn-primary" onClick={() => openTrial()}>
            <FlaskConical size={14} aria-hidden /> 规则试匹配
          </button>
        }
      />
      {err && <p className="err">{err}</p>}

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
          <div className="label">全局处置</div>
          <div className="value">{status?.waf_log_only ? '记录' : '拦截'}</div>
          <div className="sub">waf.log_only</div>
        </div>
      </div>

      {!configWafEnabled && !runtimeWafEnabled && events.length > 0 ? (
        <div className="panel waf-seed-notice">
          <div className="panel-body">
            <p className="match-hint" style={{ margin: 0 }}>
              当前<strong>配置 WAF</strong>与<strong>运行时 WAF</strong>均为关闭。列表中的 block/audit 事件多数来自
              admin 首次启动时写入的<strong>演示种子数据</strong>（约 180 条），用于预览界面，并非真实拦截记录。
              「规则试匹配」只按当前配置模拟，因此通常无法复现列表中的命中。开启 WAF 后产生的新事件才会与试匹配一致。
            </p>
          </div>
        </div>
      ) : null}

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
              实时刷新{sseConnected ? ' (SSE)' : realtime && !sseConnected ? ' (3s 轮询)' : ''}
            </label>
            <button type="button" className="btn btn-sm" onClick={handleRefresh}>
              刷新
            </button>
            <button type="button" className="btn btn-sm btn-ghost" onClick={handleResetFilters}>
              重置
            </button>
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
                <option key={h} value={h}>
                  {h}
                </option>
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
                <option
                  key={r}
                  value={r}
                  title={formatWafRuleTooltip(resolveWafRule(ruleLookup, r), r)}
                >
                  {r}
                </option>
              ))}
            </select>
          </div>
          <div className="filter-field">
            <label>开始时间</label>
            <input type="date" value={filterTimeStart} onChange={(e) => setFilterTimeStart(e.target.value)} />
          </div>
          <div className="filter-field">
            <label>结束时间</label>
            <input type="date" value={filterTimeEnd} onChange={(e) => setFilterTimeEnd(e.target.value)} />
          </div>
        </div>
      </div>

      <div className="panel">
        <div className="panel-head">
          <h2>事件列表 ({events.length})</h2>
        </div>
        <div className="panel-body panel-table-wrap">
          {events.length === 0 ? (
            <EmptyStateGuide title="暂无 WAF 事件" configModule="waf">
              开启全局 WAF 并产生真实流量后，block/audit 会写入事件表。若曾看到大量事件而 WAF 为关闭，多为 admin
              演示种子，可忽略或清空数据库后重启。
            </EmptyStateGuide>
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
                  <th className="col-actions">操作</th>
                </tr>
              </thead>
              <tbody>
                {events.map((e) => (
                  <tr
                    key={e.id}
                    className={drawer === 'detail' && selectedId === e.id ? 'match-highlight' : ''}
                  >
                    <td className="col-time">{formatTime(e.created_at)}</td>
                    <td className="col-action">
                      <span className={`badge badge-${e.action}`}>{e.action}</span>
                    </td>
                    <td className="col-rule">
                      <span className="col-rule-inner">
                        <WafRuleTooltip rule={e.rule} lookup={ruleLookup} />
                        {resolveWafRule(ruleLookup, e.rule)?.source === 'demo' ? (
                          <span className="badge badge-audit waf-demo-badge" title="演示种子事件">
                            演示
                          </span>
                        ) : null}
                      </span>
                    </td>
                    <td className="col-host">{e.host}</td>
                    <td className="col-path">
                      <code>{e.path}</code>
                    </td>
                    <td className="col-ip">{e.client_ip}</td>
                    <td className="col-actions">
                      <button type="button" className="btn btn-ghost btn-sm" onClick={() => openDetail(e.id)}>
                        详情
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>

      <WafEventDetailDrawer
        eventId={selectedId}
        open={drawer === 'detail'}
        onClose={closeDrawer}
        onTrial={(detail) => openTrial(detail)}
      />

      <WafTrialDrawer
        open={drawer === 'trial'}
        onClose={closeDrawer}
        eventId={trialSeed?.id}
        seed={trialSeed}
      />
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
