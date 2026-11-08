import { useEffect, useState } from 'react'
import { Drawer } from '../Drawer'
import { api } from '../../api/client'
import type { BanTtlPreset } from '../../lib/wafBan'
import { BAN_TTL_PRESETS, DEFAULT_BAN_TTL, describeBanError } from '../../lib/wafBan'

type Props = {
  open: boolean
  /** Distinct IPs selected from the WAF event table. */
  ips: string[]
  onClose: () => void
  onCreated: () => void
}

export function BanBulkDrawer({ open, ips, onClose, onCreated }: Props) {
  const [ttl, setTtl] = useState<BanTtlPreset>(DEFAULT_BAN_TTL)
  const [reason, setReason] = useState('批量封禁：近 24h 高频扫描')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')
  const [done, setDone] = useState('')

  useEffect(() => {
    if (!open) return
    setTtl(DEFAULT_BAN_TTL)
    setReason('批量封禁：近 24h 高频扫描')
    setErr('')
    setDone('')
    setBusy(false)
  }, [open])

  const submit = async () => {
    setBusy(true)
    setErr('')
    const preset = BAN_TTL_PRESETS.find((p) => p.value === ttl) ?? BAN_TTL_PRESETS[1]
    const failed: string[] = []
    let ok = 0

    for (const ip of ips) {
      try {
        await api.createWafBan({
          kind: ip.includes('/') ? 'ip_cidr' : 'ip',
          value: ip,
          reason: reason.trim(),
          ttl_seconds: preset.seconds > 0 ? preset.seconds : undefined,
          confirm_permanent: preset.seconds === 0,
        })
        ok++
      } catch (e) {
        failed.push(`${ip}: ${describeBanError(e instanceof Error ? e.message : String(e))}`)
      }
    }

    setBusy(false)
    onCreated()
    if (failed.length === 0) {
      setDone(`已封禁 ${ok} 个 IP`)
      onClose()
      return
    }
    setErr(`成功 ${ok} 个，失败 ${failed.length} 个：${failed.join('；')}`)
  }

  return (
    <Drawer
      open={open}
      title="批量封禁"
      onClose={onClose}
      width={560}
      footer={
        <>
          <button type="button" className="btn btn-ghost" onClick={onClose}>
            取消
          </button>
          <button type="button" className="btn btn-danger" onClick={submit} disabled={busy || ips.length === 0}>
            {busy ? '提交中…' : `封禁 ${ips.length} 个 IP`}
          </button>
        </>
      }
    >
      <div className="ban-notice ban-notice--info">
        <span>将对选中的 {ips.length} 个 IP 逐一创建封禁条目。</span>
      </div>

      <p className="section-title">待封禁 IP</p>
      <div className="ban-target-card">
        <div className="mono" style={{ lineHeight: 1.9 }}>
          {ips.map((ip) => (
            <div key={ip}>• {ip}</div>
          ))}
          {ips.length === 0 ? <span className="muted">未选择任何 IP</span> : null}
        </div>
      </div>

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

      <p className="section-title">原因备注</p>
      <div className="ban-field">
        <input type="text" value={reason} onChange={(e) => setReason(e.target.value)} />
      </div>

      {err ? <p className="err">{err}</p> : null}
      {done ? <p className="muted">{done}</p> : null}
    </Drawer>
  )
}
