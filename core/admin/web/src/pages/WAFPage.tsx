import { useCallback, useEffect, useRef, useState } from 'react'
import { FlaskConical } from 'lucide-react'
import { PageHeader } from '../components/PageHeader'
import { EmptyStateGuide } from '../components/EmptyStateGuide'
import { Drawer } from '../components/Drawer'
import { WafRuleTooltip } from '../components/WafRuleTooltip'
import { api, type WAFEvent, type WAFEventDetail, type WAFTrialResult } from '../api/client'
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
  const [detail, setDetail] = useState<WAFEventDetail | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const [trialEventId, setTrialEventId] = useState<number | null>(null)
  const [trialExpectedRule, setTrialExpectedRule] = useState('')

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

  const [trialHost, setTrialHost] = useState('api.example.com')
  const [trialPath, setTrialPath] = useState('/search?q=test')
  const [trialMethod, setTrialMethod] = useState('GET')
  const [trialClientIP, setTrialClientIP] = useState('203.0.113.1')
  const [trialUA, setTrialUA] = useState('scanner/1.0')
  const [trialResult, setTrialResult] = useState<WAFTrialResult | null>(null)
  const [trialErr, setTrialErr] = useState('')
  const [trialLoading, setTrialLoading] = useState(false)

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
    setTrialEventId(null)
    setTrialExpectedRule('')
    if (seed) {
      setTrialHost(seed.host)
      setTrialPath(seed.path)
      setTrialClientIP(seed.client_ip || '')
      setTrialEventId(seed.id)
      setTrialExpectedRule(seed.rule || '')
      const ruleLower = (seed.rule || '').toLowerCase()
      if (ruleLower.includes('scanner') || ruleLower.includes('ua')) {
        setTrialUA('scanner/1.0')
      } else {
        setTrialUA('')
      }
    }
    setTrialResult(null)
    setTrialErr('')
    setDrawer('trial')
  }

  const closeDrawer = () => {
    setDrawer(null)
    setSelectedId(null)
    setTrialEventId(null)
    setTrialExpectedRule('')
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

  useEffect(() => {
    if (drawer !== 'detail' || selectedId == null) {
      setDetail(null)
      return
    }
    setDetailLoading(true)
    api
      .wafEvent(selectedId)
      .then(setDetail)
      .catch((e: Error) => {
        setDetail(null)
        setErr(e.message)
      })
      .finally(() => setDetailLoading(false))
  }, [drawer, selectedId])

  const runTrial = () => {
    setTrialErr('')
    setTrialResult(null)
    setTrialLoading(true)
    const path = trialPath.trim()
    let query = ''
    let pathOnly = path
    if (path.includes('?')) {
      const idx = path.indexOf('?')
      pathOnly = path.slice(0, idx) || '/'
      query = path.slice(idx + 1)
    }
    api
      .wafMatch({
        host: trialHost.trim(),
        path: pathOnly,
        method: trialMethod,
        client_ip: trialClientIP.trim() || undefined,
        query: query || undefined,
        headers: trialUA.trim() ? { 'User-Agent': trialUA.trim() } : undefined,
        event_id: trialEventId ?? undefined,
        expected_rule: trialExpectedRule || undefined,
      })
      .then(setTrialResult)
      .catch((e: Error) => setTrialErr(e.message))
      .finally(() => setTrialLoading(false))
  }

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
        desc="全局规则、运行时开关与 block/audit 事件；行点击查看详情"
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
          <div className="label">模式</div>
          <div className="value">{status?.waf_log_only ? '仅审计' : '拦截'}</div>
          <div className="sub">log_only</div>
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
                </tr>
              </thead>
              <tbody>
                {events.map((e) => (
                  <tr
                    key={e.id}
                    className={drawer === 'detail' && selectedId === e.id ? 'match-highlight' : ''}
                    onClick={() => openDetail(e.id)}
                    style={{ cursor: 'pointer' }}
                    title="点击查看详情"
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
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>

      <Drawer
        open={drawer === 'detail'}
        title={selectedId != null ? `事件详情 #${selectedId}` : '事件详情'}
        onClose={closeDrawer}
        footer={
          detail ? (
            <>
              <button type="button" className="btn btn-ghost" onClick={closeDrawer}>
                关闭
              </button>
              <button
                type="button"
                className="btn btn-primary"
                onClick={() => openTrial(detail)}
              >
                用此请求试匹配
              </button>
            </>
          ) : (
            <button type="button" className="btn btn-ghost" onClick={closeDrawer}>
              关闭
            </button>
          )
        }
      >
        {detailLoading ? (
          <p className="empty-hint">加载中…</p>
        ) : detail ? (
          <>
            <dl className="route-detail-dl">
              <dt>时间</dt>
              <dd>{new Date(detail.created_at).toLocaleString('zh-CN')}</dd>
              <dt>动作</dt>
              <dd>
                <span className={`badge badge-${detail.action}`}>{detail.action}</span>
              </dd>
              <dt>规则标识</dt>
              <dd>
                <WafRuleTooltip rule={detail.rule} lookup={ruleLookup} />
              </dd>
              <dt>Host</dt>
              <dd>
                <code>{detail.host}</code>
              </dd>
              <dt>Path</dt>
              <dd>
                <code>{detail.path}</code>
              </dd>
              <dt>客户端 IP</dt>
              <dd>{detail.client_ip || '—'}</dd>
            </dl>
            {detail.rule_detail ? (
              <div className="waf-rule-detail-box">
                <h3 className="waf-rule-detail-title">命中规则说明</h3>
                <dl className="route-detail-dl">
                  <dt>名称</dt>
                  <dd>{detail.rule_detail.name || '—'}</dd>
                  <dt>类型</dt>
                  <dd>
                    {detail.rule_detail.type || '—'}
                    {detail.rule_detail.source ? (
                      <span className="badge badge-audit" style={{ marginLeft: 8 }}>
                        {ruleSourceLabel(detail.rule_detail.source)}
                      </span>
                    ) : null}
                  </dd>
                  {detail.rule_detail.phase ? (
                    <>
                      <dt>阶段</dt>
                      <dd>{detail.rule_detail.phase}</dd>
                    </>
                  ) : null}
                  {detail.rule_detail.pattern ? (
                    <>
                      <dt>模式</dt>
                      <dd>
                        <code className="waf-rule-pattern">{detail.rule_detail.pattern}</code>
                      </dd>
                    </>
                  ) : null}
                  {detail.rule_detail.targets && detail.rule_detail.targets.length > 0 ? (
                    <>
                      <dt>检测目标</dt>
                      <dd>{detail.rule_detail.targets.join(', ')}</dd>
                    </>
                  ) : null}
                  <dt>说明</dt>
                  <dd>{detail.rule_detail.description}</dd>
                </dl>
              </div>
            ) : null}
            {detail.replay_note ? (
              <p className="match-hint waf-replay-note">{detail.replay_note}</p>
            ) : null}
          </>
        ) : (
          <p className="empty-hint">无法加载事件详情</p>
        )}
      </Drawer>

      <Drawer
        open={drawer === 'trial'}
        title="规则试匹配"
        onClose={closeDrawer}
        width={440}
        footer={
          <>
            <button type="button" className="btn btn-ghost" onClick={closeDrawer}>
              关闭
            </button>
            <button
              type="button"
              className="btn btn-primary"
              onClick={runTrial}
              disabled={trialLoading}
            >
              {trialLoading ? '匹配中…' : '执行试匹配'}
            </button>
          </>
        }
      >
        <p className="match-hint" style={{ marginTop: 0 }}>
          按<strong>当前 ingress 配置</strong>与<strong>运行时 WAF 开关</strong>模拟请求（非历史回放）。
          {trialExpectedRule ? (
            <> 期望复现规则：<code>{trialExpectedRule}</code></>
          ) : null}
        </p>
        <label className="field-label">Host</label>
        <input
          className="field-input-last"
          value={trialHost}
          onChange={(e) => setTrialHost(e.target.value)}
          style={{ width: '100%', marginBottom: 12 }}
        />
        <label className="field-label">Path（可含 query）</label>
        <input
          className="field-input-last"
          value={trialPath}
          onChange={(e) => setTrialPath(e.target.value)}
          style={{ width: '100%', marginBottom: 12 }}
        />
        <label className="field-label">Method</label>
        <select
          value={trialMethod}
          onChange={(e) => setTrialMethod(e.target.value)}
          style={{ width: '100%', marginBottom: 12 }}
        >
          <option value="GET">GET</option>
          <option value="POST">POST</option>
          <option value="PUT">PUT</option>
          <option value="DELETE">DELETE</option>
        </select>
        <label className="field-label">客户端 IP</label>
        <input
          className="field-input-last"
          value={trialClientIP}
          onChange={(e) => setTrialClientIP(e.target.value)}
          style={{ width: '100%', marginBottom: 12 }}
        />
        <label className="field-label">User-Agent</label>
        <input
          className="field-input-last"
          value={trialUA}
          onChange={(e) => setTrialUA(e.target.value)}
          style={{ width: '100%', marginBottom: 12 }}
        />
        {trialErr && <p className="err">{trialErr}</p>}
        {trialResult && (
          <div className={`match-result ${trialResult.matched ? 'hit' : 'miss'}`}>
            <h3>{trialResult.matched ? '命中 WAF' : '未命中'}</h3>
            <dl>
              <dt>规则索引</dt>
              <dd>{trialResult.rule_index}</dd>
              <dt>将拦截</dt>
              <dd>{trialResult.would_block ? '是' : '否（审计或放行）'}</dd>
              <dt>配置 WAF</dt>
              <dd>{trialResult.config_waf_enabled ? '已启用' : '关闭'}</dd>
              <dt>运行时 WAF</dt>
              <dd>{trialResult.runtime_waf_enabled ? '已启用' : '关闭'}</dd>
              <dt>试匹配 WAF</dt>
              <dd>
                {trialResult.waf_enabled
                  ? trialResult.log_only
                    ? '仅审计'
                    : '拦截模式'
                  : '未启用'}
              </dd>
              {trialResult.expected_rule ? (
                <>
                  <dt>期望规则</dt>
                  <dd>
                    <code>{trialResult.expected_rule}</code>
                    {trialResult.expected_rule_hit ? ' ✓ 已复现' : ' ✗ 未复现'}
                  </dd>
                </>
              ) : null}
            </dl>
            {trialResult.hits?.length > 0 && (
              <ul className="waf-trial-hits">
                {trialResult.hits.map((h, i) => (
                  <li key={i}>
                    <span className={`badge badge-${h.action}`}>{h.action}</span>{' '}
                    <WafRuleTooltip rule={h.rule} lookup={ruleLookup} className="inline-tooltip" />
                    {h.client_ip ? ` · ${h.client_ip}` : ''}
                  </li>
                ))}
              </ul>
            )}
            {trialResult.message && <p className="match-hint">{trialResult.message}</p>}
            {trialResult.hint && <p className="match-hint waf-trial-hint">{trialResult.hint}</p>}
          </div>
        )}
      </Drawer>
    </div>
  )
}

function ruleSourceLabel(source: string) {
  switch (source) {
    case 'config':
      return '配置文件'
    case 'builtin':
      return '内置'
    case 'demo':
      return '演示数据'
    case 'phase':
      return '阶段'
    default:
      return source
  }
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
