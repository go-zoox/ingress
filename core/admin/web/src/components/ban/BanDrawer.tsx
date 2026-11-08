import { useCallback, useEffect, useMemo, useState } from 'react'
import { Drawer } from '../Drawer'
import { api } from '../../api/client'
import type { BanTtlPreset, WAFBanPreview } from '../../lib/wafBan'
import {
  BAN_TTL_PRESETS,
  DEFAULT_BAN_TTL,
  buildBanPayload,
  buildImpactWarnings,
  describeBanError,
  resolveBanKind,
  validateBanValue,
} from '../../lib/wafBan'

export type BanDrawerMode = 'create' | 'from-event'

type Props = {
  open: boolean
  mode: BanDrawerMode
  /** Prefill from a WAF event (mode = from-event). */
  seed?: {
    kind: 'ip' | 'ua'
    value: string
    host: string
    eventId?: number
    rule?: string
  } | null
  availableHosts: string[]
  onClose: () => void
  onCreated: () => void
}

export function BanDrawer({ open, mode, seed, availableHosts, onClose, onCreated }: Props) {
  const [mode2, setMode2] = useState<'ip' | 'ua'>('ip')
  const [uaMode, setUaMode] = useState<'contains' | 'regex'>('contains')
  const [value, setValue] = useState('')
  const [reason, setReason] = useState('')
  const [ttl, setTtl] = useState<BanTtlPreset>(DEFAULT_BAN_TTL)
  const [scope, setScope] = useState<'global' | 'hosts'>('global')
  const [hosts, setHosts] = useState<string[]>([])
  const [preview, setPreview] = useState<WAFBanPreview | null>(null)
  const [previewLoading, setPreviewLoading] = useState(false)
  const [err, setErr] = useState('')
  const [submitting, setSubmitting] = useState(false)

  // Reset when the drawer opens.
  useEffect(() => {
    if (!open) return
    const kind = seed?.kind ?? 'ip'
    setMode2(kind)
    setUaMode('contains')
    setValue(seed?.value ?? '')
    setReason(kind === 'ip' ? '疑似扫描，应急处置' : `按 UA 封禁：${(seed?.value ?? '').slice(0, 40)}`)
    setTtl(DEFAULT_BAN_TTL)
    setScope('global')
    setHosts([])
    setPreview(null)
    setErr('')
    setSubmitting(false)
  }, [open, seed])

  const validation = useMemo(() => {
    const kind = resolveBanKind(mode2, value, uaMode)
    return validateBanValue(kind, value)
  }, [mode2, value, uaMode])

  const blocked = Boolean(preview?.blocked)
  const warnings = buildImpactWarnings(preview)
  const isPermanent = ttl === 'forever'

  // Refresh the impact preview whenever the target is valid.
  const runPreview = useCallback(() => {
    if (!validation.ok) {
      setPreview(null)
      return
    }
    setPreviewLoading(true)
    api
      .previewWafBan({ kind: resolveBanKind(mode2, value, uaMode), value: validation.normalized })
      .then(setPreview)
      .catch(() => setPreview(null))
      .finally(() => setPreviewLoading(false))
  }, [mode2, uaMode, value, validation])

  useEffect(() => {
    if (!open) return
    const timer = setTimeout(runPreview, 350)
    return () => clearTimeout(timer)
  }, [open, runPreview])

  const submit = async () => {
    if (!validation.ok) {
      setErr(validation.error)
      return
    }
    setSubmitting(true)
    setErr('')
    try {
      const payload = buildBanPayload({
        mode: mode2,
        value: validation.normalized,
        uaMode,
        ttl,
        reason,
        hosts: scope === 'hosts' ? hosts : [],
        confirmPermanent: isPermanent,
      })
      await api.createWafBan(payload)
      onCreated()
      onClose()
    } catch (e) {
      setErr(describeBanError(e instanceof Error ? e.message : String(e)))
    } finally {
      setSubmitting(false)
    }
  }

  const toggleHost = (host: string) => {
    setHosts((prev) => (prev.includes(host) ? prev.filter((h) => h !== host) : [...prev, host]))
  }

  return (
    <Drawer
      open={open}
      title={mode === 'from-event' ? (mode2 === 'ip' ? '封禁 IP' : '封禁 UA') : '新增封禁'}
      onClose={onClose}
      width={560}
      footer={
        <>
          <button type="button" className="btn btn-ghost" onClick={onClose}>
            取消
          </button>
          <button
            type="button"
            className="btn btn-danger"
            onClick={submit}
            disabled={!validation.ok || blocked || submitting}
          >
            {submitting ? '提交中…' : isPermanent ? '永久封禁（再次确认）' : '确认封禁'}
          </button>
        </>
      }
    >
      {mode === 'create' ? (
        <>
          <p className="section-title">封禁类型</p>
          <div className="ban-segmented">
            <button
              type="button"
              className={`ban-seg${mode2 === 'ip' ? ' active' : ''}`}
              onClick={() => {
                setMode2('ip')
                setValue('')
              }}
            >
              IP / CIDR
            </button>
            <button
              type="button"
              className={`ban-seg${mode2 === 'ua' ? ' active' : ''}`}
              onClick={() => {
                setMode2('ua')
                setValue('')
              }}
            >
              User-Agent
            </button>
          </div>
        </>
      ) : null}

      {mode2 === 'ua' ? (
        <div className="ban-field">
          <label>匹配方式</label>
          <select value={uaMode} onChange={(e) => setUaMode(e.target.value as 'contains' | 'regex')}>
            <option value="contains">包含子串（contains）</option>
            <option value="regex">正则（regex）</option>
          </select>
        </div>
      ) : null}

      <div className={`ban-field${value && !validation.ok ? ' invalid' : ''}`}>
        <label>{mode2 === 'ip' ? 'IP 或 CIDR' : 'User-Agent 关键词'}</label>
        <input
          type="text"
          value={value}
          placeholder={mode2 === 'ip' ? '203.0.113.42 或 203.0.113.0/24' : '例：sqlmap 或 (?i)(nikto|masscan)'}
          onChange={(e) => setValue(e.target.value)}
        />
        {value && !validation.ok ? <p className="field-error">{validation.error}</p> : null}
        <p className="hint">
          {mode2 === 'ip'
            ? '支持单个 IP 或 CIDR 网段；IPv6 自动归一化。'
            : '按子串或正则匹配 User-Agent 请求头，大小写不敏感。'}
        </p>
      </div>

      {mode === 'from-event' && seed?.host ? (
        <div className="ban-notice ban-notice--muted">
          <span>
            来源事件{seed.eventId ? ` #${seed.eventId}` : ''} · {seed.rule || '—'} · {seed.host}
          </span>
        </div>
      ) : null}

      {preview || previewLoading ? (
        <>
          <p className="section-title">影响面评估</p>
          <div className="ban-metrics">
            <div className="ban-metric">
              <div className="m-label">近 24h 命中</div>
              <div className="m-value danger">{previewLoading ? '…' : preview?.hits_24h ?? 0}</div>
            </div>
            <div className="ban-metric">
              <div className="m-label">涉及 Host</div>
              <div className="m-value">{previewLoading ? '…' : preview?.hosts ?? 0}</div>
            </div>
            <div className="ban-metric">
              <div className="m-label">独立 UA</div>
              <div className="m-value">{previewLoading ? '…' : preview?.distinct_ua ?? 0}</div>
            </div>
            <div className="ban-metric">
              <div className="m-label">独立路径</div>
              <div className="m-value">{previewLoading ? '…' : preview?.paths ?? 0}</div>
            </div>
          </div>
        </>
      ) : null}

      {warnings.map((w) => (
        <div key={w} className={`ban-notice ban-notice--${blocked ? 'danger' : 'warn'}`}>
          <span>{w}</span>
        </div>
      ))}

      <p className="section-title">生效范围</p>
      <div className="ban-radio-row">
        <label>
          <input
            type="radio"
            name="ban-scope"
            checked={scope === 'global'}
            onChange={() => {
              setScope('global')
              // Scope drives the payload: leaving stale host chips selected
              // would submit a host-scoped ban while the UI says global.
              setHosts([])
            }}
          />
          全部 Host（全局）
        </label>
        <label>
          <input type="radio" name="ban-scope" checked={scope === 'hosts'} onChange={() => setScope('hosts')} />
          指定 Host
        </label>
      </div>
      {scope === 'hosts' ? (
        <div className="ban-host-chips">
          {availableHosts.map((host) => (
            <button
              key={host}
              type="button"
              className={`ban-host-chip${hosts.includes(host) ? ' on' : ''}`}
              onClick={() => toggleHost(host)}
            >
              {host}
            </button>
          ))}
          {availableHosts.length === 0 ? <span className="hint">暂无可用 Host</span> : null}
        </div>
      ) : null}

      <p className="section-title">封禁时长</p>
      <div className="ban-segmented">
        {BAN_TTL_PRESETS.map((preset) => (
          <button
            key={preset.value}
            type="button"
            className={`ban-seg${ttl === preset.value ? ' active' : ''}${preset.value === 'forever' ? ' danger' : ''}`}
            onClick={() => setTtl(preset.value)}
          >
            {preset.label}
          </button>
        ))}
      </div>
      {isPermanent ? (
        <div className="ban-notice ban-notice--warn">
          <span>永久条目不会自动过期，建议优先使用 7 天并观察命中情况。</span>
        </div>
      ) : null}

      <p className="section-title">原因备注</p>
      <div className="ban-field">
        <input type="text" value={reason} onChange={(e) => setReason(e.target.value)} placeholder="例：持续 SQL 注入扫描" />
      </div>

      {err ? <p className="err">{err}</p> : null}

      <div className="ban-notice ban-notice--muted">
        <span>
          封禁立即生效，不触发配置 reload、不清空响应缓存、不影响限流计数器。
          {mode2 === 'ua' ? ' UA 是客户端可控字段，换 UA 即可绕过，仅作止血手段。' : ''}
        </span>
      </div>
    </Drawer>
  )
}
