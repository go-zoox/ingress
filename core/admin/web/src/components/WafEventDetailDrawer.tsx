import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Drawer } from './Drawer'
import { WafRuleTooltip } from './WafRuleTooltip'
import { api, type WAFEventDetail } from '../api/client'
import { investigateLink, wafLink } from '../lib/deepLinks'
import { useWafRuleLookup } from '../hooks/useWafRuleLookup'

type Props = {
  eventId: number | null
  open: boolean
  onClose: () => void
  onTrial?: (detail: WAFEventDetail) => void
  onStatusChange?: (id: number, status: 'ignored' | 'resolved') => void
}

export function WafEventDetailDrawer({ eventId, open, onClose, onTrial, onStatusChange }: Props) {
  const { lookup: ruleLookup } = useWafRuleLookup()
  const [detail, setDetail] = useState<WAFEventDetail | null>(null)
  const [loading, setLoading] = useState(false)
  const [err, setErr] = useState('')

  useEffect(() => {
    if (!open || eventId == null) {
      setDetail(null)
      setErr('')
      return
    }
    setLoading(true)
    setErr('')
    api
      .wafEvent(eventId)
      .then((d) => {
        setDetail(d)
        setLoading(false)
      })
      .catch((e: Error) => {
        setErr(e.message)
        setDetail(null)
        setLoading(false)
      })
  }, [open, eventId])

  const handleStatus = async (status: 'ignored' | 'resolved') => {
    if (eventId == null) return
    try {
      await api.updateWafEventStatus(eventId, status)
      onStatusChange?.(eventId, status)
      onClose()
    } catch {
      // keep drawer open on failure
    }
  }

  const isOpen = detail ? wafEventNeedsAttention(detail.status) : false

  return (
    <Drawer
      open={open}
      title={eventId != null ? `WAF 事件详情 #${eventId}` : 'WAF 事件详情'}
      onClose={onClose}
      width={520}
      footer={
        detail ? (
          <>
            <button type="button" className="btn btn-ghost" onClick={onClose}>
              关闭
            </button>
            {isOpen && onStatusChange ? (
              <>
                <button type="button" className="btn btn-ghost" onClick={() => handleStatus('ignored')}>
                  忽略
                </button>
                <button type="button" className="btn btn-primary" onClick={() => handleStatus('resolved')}>
                  已处理
                </button>
              </>
            ) : null}
            <Link
              to={investigateLink({
                host: detail.host,
                path: detail.path || '/',
                client_ip: detail.client_ip,
              })}
              className={isOpen && onStatusChange ? 'btn btn-ghost' : 'btn btn-primary'}
            >
              调查此请求
            </Link>
            {onTrial ? (
              <button type="button" className="btn btn-ghost" onClick={() => onTrial(detail)}>
                调试
              </button>
            ) : (
              <Link
                to={wafLink({
                  host: detail.host,
                  path: detail.path,
                  trial: true,
                  eventId: detail.id,
                })}
                className="btn btn-ghost"
              >
                调试
              </Link>
            )}
          </>
        ) : (
          <button type="button" className="btn btn-ghost" onClick={onClose}>
            关闭
          </button>
        )
      }
    >
      {loading ? <p className="empty-hint">加载中…</p> : null}
      {err ? <p className="err">{err}</p> : null}
      {detail ? (
        <>
          <dl className="route-detail-dl">
            <dt>时间</dt>
            <dd>{new Date(detail.created_at).toLocaleString('zh-CN')}</dd>
            <dt>动作</dt>
            <dd>
              <span className={`badge badge-${detail.action}`}>{detail.action}</span>
            </dd>
            {!wafEventNeedsAttention(detail.status) ? (
              <>
                <dt>处置状态</dt>
                <dd>{wafEventStatusLabel(detail.status)}</dd>
              </>
            ) : null}
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
            <dt>User Agent</dt>
            <dd>
              {detail.user_agent ? (
                <code className="waf-user-agent">{detail.user_agent}</code>
              ) : (
                '—'
              )}
            </dd>
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
      ) : !loading && !err ? (
        <p className="empty-hint">无法加载事件详情</p>
      ) : null}
    </Drawer>
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

function wafEventNeedsAttention(status?: string) {
  const s = (status || 'open').trim()
  return s === '' || s === 'open'
}

function wafEventStatusLabel(status?: string) {
  switch ((status || '').trim()) {
    case 'resolved':
      return '已处理'
    case 'ignored':
      return '已忽略'
    default:
      return '待处理'
  }
}
