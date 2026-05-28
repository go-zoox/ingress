import { useEffect, useState } from 'react'
import { Drawer } from './Drawer'
import { WafRuleTooltip } from './WafRuleTooltip'
import { api, type WAFEvent, type WAFEventDetail, type WAFTrialResult } from '../api/client'
import { useWafRuleLookup } from '../hooks/useWafRuleLookup'
import { wafTrialFormFromSeed, type WafTrialSeed } from '../lib/wafTrialSeed'

type Props = {
  open: boolean
  onClose: () => void
  eventId?: number | null
  seed?: Partial<WafTrialSeed> | WAFEvent | WAFEventDetail | null
}

export function WafTrialDrawer({ open, onClose, eventId, seed }: Props) {
  const { lookup: ruleLookup } = useWafRuleLookup()
  const [trialHost, setTrialHost] = useState('api.example.com')
  const [trialPath, setTrialPath] = useState('/search?q=test')
  const [trialMethod, setTrialMethod] = useState('GET')
  const [trialClientIP, setTrialClientIP] = useState('')
  const [trialUA, setTrialUA] = useState('')
  const [trialEventId, setTrialEventId] = useState<number | null>(null)
  const [trialExpectedRule, setTrialExpectedRule] = useState('')
  const [trialResult, setTrialResult] = useState<WAFTrialResult | null>(null)
  const [trialErr, setTrialErr] = useState('')
  const [trialLoading, setTrialLoading] = useState(false)

  useEffect(() => {
    if (!open) {
      setTrialResult(null)
      setTrialErr('')
      return
    }

    const applySeed = (next?: Partial<WafTrialSeed> | null) => {
      const form = wafTrialFormFromSeed(next)
      setTrialHost(form.host)
      setTrialPath(form.path)
      setTrialMethod(form.method)
      setTrialClientIP(form.clientIP)
      setTrialUA(form.userAgent)
      setTrialEventId(form.eventId)
      setTrialExpectedRule(form.expectedRule)
      setTrialResult(null)
      setTrialErr('')
    }

    const id = eventId ?? seed?.id
    if (id != null && id > 0) {
      api
        .wafEvent(id)
        .then((ev) => applySeed(ev))
        .catch(() => applySeed(seed))
      return
    }
    applySeed(seed)
  }, [open, eventId, seed])

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

  return (
    <Drawer
      open={open}
      title="WAF 调试"
      onClose={onClose}
      width={440}
      footer={
        <>
          <button type="button" className="btn btn-ghost" onClick={onClose}>
            关闭
          </button>
          <button type="button" className="btn btn-primary" onClick={runTrial} disabled={trialLoading}>
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
      {trialErr ? <p className="err">{trialErr}</p> : null}
      {trialResult ? (
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
          {trialResult.hits?.length > 0 ? (
            <ul className="waf-trial-hits">
              {trialResult.hits.map((h, i) => (
                <li key={i}>
                  <span className={`badge badge-${h.action}`}>{h.action}</span>{' '}
                  <WafRuleTooltip rule={h.rule} lookup={ruleLookup} className="inline-tooltip" />
                  {h.client_ip ? ` · ${h.client_ip}` : ''}
                </li>
              ))}
            </ul>
          ) : null}
          {trialResult.message ? <p className="match-hint">{trialResult.message}</p> : null}
          {trialResult.hint ? <p className="match-hint waf-trial-hint">{trialResult.hint}</p> : null}
        </div>
      ) : null}
    </Drawer>
  )
}
