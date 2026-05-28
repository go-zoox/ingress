import { useEffect, useState } from 'react'
import { Drawer } from './Drawer'
import { api, type AccessLogParseIssueDetail } from '../api/client'

type Props = {
  issueId: number | null
  open: boolean
  onClose: () => void
  onStatusChange?: (id: number, status: 'ignored' | 'resolved') => void
}

export function ParseIssueDetailDrawer({ issueId, open, onClose, onStatusChange }: Props) {
  const [detail, setDetail] = useState<AccessLogParseIssueDetail | null>(null)
  const [loading, setLoading] = useState(false)
  const [err, setErr] = useState('')

  useEffect(() => {
    if (!open || issueId == null) {
      setDetail(null)
      setErr('')
      return
    }
    setLoading(true)
    setErr('')
    api
      .parseIssueDetail(issueId)
      .then((d) => {
        setDetail(d)
        setLoading(false)
      })
      .catch((e: Error) => {
        setErr(e.message)
        setDetail(null)
        setLoading(false)
      })
  }, [open, issueId])

  const handleStatus = async (status: 'ignored' | 'resolved') => {
    if (issueId == null) return
    try {
      await api.updateParseIssueStatus(issueId, status)
      onStatusChange?.(issueId, status)
      onClose()
    } catch {
      // keep drawer open on failure
    }
  }

  return (
    <Drawer
      open={open}
      title="日志解析详情"
      onClose={onClose}
      width={720}
      footer={
        detail?.status === 'open' ? (
          <>
            <button type="button" className="btn btn-ghost" onClick={onClose}>
              关闭
            </button>
            <button type="button" className="btn btn-ghost" onClick={() => handleStatus('ignored')}>
              忽略
            </button>
            <button type="button" className="btn btn-primary" onClick={() => handleStatus('resolved')}>
              已处理
            </button>
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
        <div className="parse-issue-detail">
          <section className="parse-issue-section">
            <h3>解析失败原因</h3>
            <div className="parse-issue-diagnosis">
              <div className="parse-issue-reason">{detail.diagnosis.reason_label}</div>
              <p className="parse-issue-hint">{detail.diagnosis.hint}</p>
              <dl className="parse-issue-checks">
                <div>
                  <dt>Host 段</dt>
                  <dd className={detail.diagnosis.has_host ? 'ok' : 'bad'}>
                    {detail.diagnosis.has_host ? '已识别' : '未识别'}
                  </dd>
                </div>
                <div>
                  <dt>HTTP 请求段</dt>
                  <dd className={detail.diagnosis.has_request ? 'ok' : 'bad'}>
                    {detail.diagnosis.has_request ? '已识别' : '未识别'}
                  </dd>
                </div>
                <div>
                  <dt>命中次数</dt>
                  <dd>{detail.hit_count}</dd>
                </div>
                <div>
                  <dt>最近出现</dt>
                  <dd>{formatTime(detail.last_seen_at)}</dd>
                </div>
              </dl>
            </div>
          </section>

          <section className="parse-issue-section">
            <h3>问题行样本</h3>
            <pre className="parse-issue-sample">{detail.diagnosis.sample_line || detail.sample_line}</pre>
          </section>

          <section className="parse-issue-section">
            <h3>上下文日志</h3>
            <p className="parse-issue-context-hint">在 access.log 中定位到匹配行，并展示前后各 3 行。</p>
            {detail.context.length === 0 ? (
              <p className="empty-hint">未在 recent tail 中找到匹配行（可能已轮转或样本已截断）。</p>
            ) : (
              <ul className="parse-issue-context">
                {detail.context.map((row, i) => (
                  <li
                    key={i}
                    className={`parse-issue-context-line${row.match ? ' match' : ''}${row.parsed ? ' parsed' : ' unparsed'}`}
                  >
                    <span className="parse-issue-context-tag">
                      {row.match ? '问题行' : row.parsed ? '可解析' : '不可解析'}
                    </span>
                    <code>{row.line}</code>
                  </li>
                ))}
              </ul>
            )}
          </section>
        </div>
      ) : null}
    </Drawer>
  )
}

function formatTime(iso: string) {
  try {
    return new Date(iso).toLocaleString('zh-CN', { hour12: false })
  } catch {
    return iso
  }
}
