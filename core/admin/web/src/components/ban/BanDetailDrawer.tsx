import { useEffect, useState } from 'react'
import { Drawer } from '../Drawer'
import { api } from '../../api/client'
import type { WAFBanRow } from '../../lib/wafBan'
import { BAN_KIND_LABELS, describeBanError, describeScope, formatRemaining } from '../../lib/wafBan'

type Props = {
  banId: number | null
  open: boolean
  onClose: () => void
  onChanged: () => void
}

export function BanDetailDrawer({ banId, open, onClose, onChanged }: Props) {
  const [row, setRow] = useState<WAFBanRow | null>(null)
  const [loading, setLoading] = useState(false)
  const [err, setErr] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    if (!open || banId == null) {
      setRow(null)
      setErr('')
      return
    }
    setLoading(true)
    api
      .wafBan(banId)
      .then((r) => {
        setRow(r)
        setErr('')
      })
      .catch((e: Error) => setErr(describeBanError(e.message)))
      .finally(() => setLoading(false))
  }, [open, banId])

  const unban = async () => {
    if (banId == null) return
    setBusy(true)
    try {
      await api.deleteWafBan(banId)
      onChanged()
      onClose()
    } catch (e) {
      setErr(describeBanError(e instanceof Error ? e.message : String(e)))
    } finally {
      setBusy(false)
    }
  }

  const renew = async () => {
    if (banId == null) return
    setBusy(true)
    try {
      await api.updateWafBan(banId, { ttl_seconds: 86400 })
      const fresh = await api.wafBan(banId)
      setRow(fresh)
      onChanged()
    } catch (e) {
      setErr(describeBanError(e instanceof Error ? e.message : String(e)))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Drawer
      open={open}
      title={banId != null ? `封禁详情 #${banId}` : '封禁详情'}
      onClose={onClose}
      width={520}
      footer={
        <>
          <button type="button" className="btn btn-danger" onClick={unban} disabled={busy || !row}>
            立即解封
          </button>
          <button type="button" className="btn btn-ghost" onClick={renew} disabled={busy || !row}>
            续期 24 小时
          </button>
          <button type="button" className="btn btn-ghost" onClick={onClose}>
            关闭
          </button>
        </>
      }
    >
      {loading ? <p className="muted">加载中…</p> : null}
      {err ? <p className="err">{err}</p> : null}
      {row ? (
        <>
          <p className="section-title">封禁目标</p>
          <div className="ban-target-card">
            <div className="ban-target-row">
              <span className="badge badge-kind">{BAN_KIND_LABELS[row.kind]}</span>
              <span className="ban-target-value">{row.value}</span>
            </div>
            {row.reason ? <div className="ban-target-meta">{row.reason}</div> : null}
          </div>

          <p className="section-title">状态</p>
          <div className="ban-metrics">
            <div className="ban-metric">
              <div className="m-label">命中次数</div>
              <div className="m-value">{row.hits.toLocaleString()}</div>
            </div>
            <div className="ban-metric">
              <div className="m-label">到期</div>
              <div className={`m-value${row.active ? '' : ' muted'}`}>
                {row.expires_at ? new Date(row.expires_at).toLocaleString() : '永久'}
              </div>
            </div>
            <div className="ban-metric">
              <div className="m-label">剩余</div>
              <div className="m-value">{formatRemaining(row.expires_at)}</div>
            </div>
            <div className="ban-metric">
              <div className="m-label">生效范围</div>
              <div className="m-value" style={{ fontSize: 13 }}>
                {describeScope(row)}
              </div>
            </div>
          </div>

          <p className="section-title">来源</p>
          <div className="ban-target-card">
            <table className="data" style={{ fontSize: 12 }}>
              <tbody>
                <tr>
                  <td style={{ color: 'var(--text-muted)', width: 90 }}>创建时间</td>
                  <td>{new Date(row.created_at).toLocaleString()}</td>
                </tr>
                <tr>
                  <td style={{ color: 'var(--text-muted)' }}>操作人</td>
                  <td>{row.actor || '—'}</td>
                </tr>
                <tr>
                  <td style={{ color: 'var(--text-muted)' }}>来源</td>
                  <td>{row.source === 'manual' ? '手动封禁' : '自动封禁'}</td>
                </tr>
                <tr>
                  <td style={{ color: 'var(--text-muted)' }}>状态</td>
                  <td>{row.active ? '生效中' : row.enabled ? '已过期' : '已停用'}</td>
                </tr>
              </tbody>
            </table>
          </div>

          <div className="ban-notice ban-notice--muted">
            <span>该封禁独立于 waf.enabled 与配置 reload，WAF 总开关关闭时仍然生效。</span>
          </div>
        </>
      ) : null}
    </Drawer>
  )
}
