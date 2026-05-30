import { useEffect, useState } from 'react'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { Drawer } from '../Drawer'
import { api, type JobRunRow } from '../../api/client'

function formatTime(v?: string) {
  if (!v) return '—'
  try {
    return new Date(v).toLocaleString()
  } catch {
    return v
  }
}

function statusBadge(status?: string) {
  if (status === 'success') return <span className="badge badge-ok">成功</span>
  if (status === 'failed') return <span className="badge badge-block">失败</span>
  if (status === 'running') return <span className="badge badge-wildcard">运行中</span>
  return <span className="badge">—</span>
}

function RunResultView({ run }: { run: JobRunRow }) {
  const http = run.result?.http
  const command = run.result?.command
  const message = run.result?.message

  if (http) {
    return (
      <div className="jobs-run-detail">
        <div className="jobs-run-detail-row">
          <span className="label">状态码</span>
          <code>{http.status_code}</code>
        </div>
        <div className="jobs-run-detail-block">
          <div className="label">响应头</div>
          <pre className="jobs-run-pre">
            {Object.keys(http.headers || {}).length === 0
              ? '(无)'
              : Object.entries(http.headers || {})
                  .map(([k, v]) => `${k}: ${v}`)
                  .join('\n')}
          </pre>
        </div>
        <div className="jobs-run-detail-block">
          <div className="label">响应体</div>
          <pre className="jobs-run-pre">{http.body || '(空)'}</pre>
        </div>
      </div>
    )
  }

  if (command) {
    return (
      <div className="jobs-run-detail">
        <div className="jobs-run-detail-block">
          <div className="label">脚本输出</div>
          <pre className="jobs-run-pre">{command.log || '(无输出)'}</pre>
        </div>
      </div>
    )
  }

  return (
    <div className="jobs-run-detail">
      <pre className="jobs-run-pre">{message || run.output_preview || run.error || '(无详情)'}</pre>
    </div>
  )
}

type Props = {
  open: boolean
  onClose: () => void
  source: 'builtin' | 'config'
  jobId: string
  jobName?: string
  refreshKey?: number
}

export function JobHistoryDrawer({ open, onClose, source, jobId, jobName, refreshKey = 0 }: Props) {
  const [runs, setRuns] = useState<JobRunRow[]>([])
  const [loading, setLoading] = useState(false)
  const [err, setErr] = useState('')
  const [expandedRunId, setExpandedRunId] = useState<number | null>(null)
  const [runDetails, setRunDetails] = useState<Record<number, JobRunRow>>({})
  const [detailLoading, setDetailLoading] = useState<number | null>(null)

  useEffect(() => {
    if (!open) {
      setExpandedRunId(null)
      setRunDetails({})
      return
    }
    setLoading(true)
    setErr('')
    api
      .jobRunsForJob(source, jobId, 30)
      .then(setRuns)
      .catch((e: Error) => setErr(e.message))
      .finally(() => setLoading(false))
  }, [open, source, jobId, refreshKey])

  const toggleRun = async (runId: number) => {
    if (expandedRunId === runId) {
      setExpandedRunId(null)
      return
    }
    setExpandedRunId(runId)
    if (runDetails[runId]) return
    setDetailLoading(runId)
    try {
      const detail = await api.jobRunDetail(runId)
      setRunDetails((d) => ({ ...d, [runId]: detail }))
    } catch (e: unknown) {
      setErr(e instanceof Error ? e.message : String(e))
    } finally {
      setDetailLoading(null)
    }
  }

  const title = jobName ? `执行历史 · ${jobName}` : `执行历史 · ${jobId}`

  return (
    <Drawer
      open={open}
      title={title}
      onClose={onClose}
      width={680}
      footer={
        <button type="button" className="btn btn-ghost" onClick={onClose}>
          关闭
        </button>
      }
    >
      <div className="jobs-history-drawer">
        {loading && <p className="settings-note">加载中…</p>}
        {err && <p className="err">{err}</p>}
        {!loading && runs.length === 0 && !err && <p className="settings-note">尚无执行记录</p>}
        {runs.length > 0 && (
          <ul className="jobs-history-list">
            {runs.map((run) => (
              <li key={run.id} className="jobs-history-item">
                <button type="button" className="jobs-history-row" onClick={() => toggleRun(run.id)}>
                  {expandedRunId === run.id ? (
                    <ChevronDown size={14} aria-hidden />
                  ) : (
                    <ChevronRight size={14} aria-hidden />
                  )}
                  <span className="jobs-history-time">{formatTime(run.finished_at || run.started_at)}</span>
                  {statusBadge(run.status)}
                  <span className="jobs-history-trigger">{run.trigger === 'manual' ? '手动' : '定时'}</span>
                  <span className="jobs-history-ms">{Math.round(run.duration_ms)} ms</span>
                  <span className="jobs-history-preview">{run.error || run.output_preview || ''}</span>
                </button>
                {expandedRunId === run.id && (
                  <div className="jobs-history-detail">
                    {detailLoading === run.id && <p className="settings-note">加载详情…</p>}
                    {runDetails[run.id] ? <RunResultView run={runDetails[run.id]} /> : null}
                    {!runDetails[run.id] && detailLoading !== run.id && run.error && (
                      <pre className="jobs-run-pre">{run.error}</pre>
                    )}
                  </div>
                )}
              </li>
            ))}
          </ul>
        )}
      </div>
    </Drawer>
  )
}
