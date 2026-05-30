import { useCallback, useEffect, useMemo, useState } from 'react'
import { History, Play, Plus, RefreshCw } from 'lucide-react'
import { PageHeader } from '../components/PageHeader'
import { JobFormDrawer } from '../components/jobs/JobFormDrawer'
import { JobHistoryDrawer } from '../components/jobs/JobHistoryDrawer'
import { CronScheduleField } from '../components/jobs/CronScheduleField'
import { FormCheckbox, FormField, FormGrid } from '../components/Form'
import { ToastContainer, useToast } from '../components/Toast'
import {
  api,
  type BuiltinJobPatch,
  type JobView,
  type JobsListResult,
} from '../api/client'
import {
  normalizeScriptEngine,
  scriptEngineKindLabel,
} from '../lib/scriptParams'

type Tab = 'builtins' | 'custom'

type HistoryTarget = {
  source: 'builtin' | 'config'
  jobId: string
  jobName?: string
}

function jobTypeLabel(job: JobView) {
  if (job.kind === 'http_call') return '接口调用'
  if (job.kind === 'script' || job.kind === 'command') {
    const engine = normalizeScriptEngine(job.params.engine)
    const label = scriptEngineKindLabel(engine)
    if (engine === 'shell' && job.params.shell) {
      return `${label} (${job.params.shell})`
    }
    return label
  }
  return job.kind
}

export function JobsPage() {
  const [tab, setTab] = useState<Tab>('custom')
  const [data, setData] = useState<JobsListResult | null>(null)
  const [loading, setLoading] = useState(true)
  const [err, setErr] = useState('')
  const [runningId, setRunningId] = useState('')
  const [historyRefresh, setHistoryRefresh] = useState(0)
  const [builtinDraft, setBuiltinDraft] = useState<
    Record<string, { enabled: boolean; schedule: string; retain_days?: number }>
  >({})
  const [formOpen, setFormOpen] = useState(false)
  const [editJob, setEditJob] = useState<JobView | null>(null)
  const [historyTarget, setHistoryTarget] = useState<HistoryTarget | null>(null)
  const { toast, show, clear } = useToast()

  const load = useCallback(() => {
    setLoading(true)
    setErr('')
    api
      .jobs()
      .then((jobs) => {
        setData(jobs)
        const draft: Record<string, { enabled: boolean; schedule: string; retain_days?: number }> = {}
        for (const b of jobs.builtins) {
          draft[b.id] = {
            enabled: b.enabled,
            schedule: b.schedule,
            retain_days: b.params.retain_days,
          }
        }
        setBuiltinDraft(draft)
        setLoading(false)
      })
      .catch((e: Error) => {
        setErr(e.message)
        setLoading(false)
      })
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const capabilities = data?.capabilities
  const customItems = useMemo(() => data?.items ?? [], [data])

  const bumpHistory = () => setHistoryRefresh((n) => n + 1)

  const saveBuiltin = async (id: string) => {
    const draft = builtinDraft[id]
    if (!draft) return
    const patch: BuiltinJobPatch = {
      enabled: draft.enabled,
      schedule: draft.schedule,
    }
    if (draft.retain_days != null) {
      patch.params = { retain_days: draft.retain_days }
    }
    try {
      await api.updateBuiltinJob(id, patch)
      show(`已保存内置任务 ${id}`)
      load()
    } catch (e: unknown) {
      show(e instanceof Error ? e.message : String(e), 'error')
    }
  }

  const runJob = async (source: 'builtin' | 'config', id: string) => {
    setRunningId(`${source}:${id}`)
    try {
      const row = await api.runJob(source, id)
      show(
        row.status === 'success' ? `任务 ${id} 执行成功` : `任务 ${id} 失败：${row.error || ''}`,
        row.status === 'success' ? 'success' : 'error',
      )
      load()
      bumpHistory()
    } catch (e: unknown) {
      show(e instanceof Error ? e.message : String(e), 'error')
    } finally {
      setRunningId('')
    }
  }

  const openCreate = () => {
    setEditJob(null)
    setFormOpen(true)
  }

  const openEdit = (job: JobView) => {
    setEditJob(job)
    setFormOpen(true)
  }

  const openHistory = (target: HistoryTarget) => {
    setHistoryTarget(target)
  }

  return (
    <div className="page">
      <PageHeader
        title="定时任务"
        desc="在此直接管理内置运维任务与自定义调度，无需进入配置中心"
        actions={
          <>
            {tab === 'custom' ? (
              <button type="button" className="btn btn-sm" onClick={openCreate}>
                <Plus size={14} aria-hidden /> 新建任务
              </button>
            ) : null}
            <button type="button" className="btn btn-sm" onClick={load} disabled={loading}>
              <RefreshCw size={14} aria-hidden /> 刷新
            </button>
          </>
        }
      />
      {err && <p className="err">{err}</p>}

      <div className="jobs-tabs">
        <button type="button" className={tab === 'custom' ? 'active' : ''} onClick={() => setTab('custom')}>
          自定义任务
        </button>
        <button type="button" className={tab === 'builtins' ? 'active' : ''} onClick={() => setTab('builtins')}>
          内置任务
        </button>
      </div>

      {tab === 'custom' && (
        <div className="jobs-custom-list">
          {capabilities?.command_restricted ? (
            <p className="settings-note">
              脚本执行已限制为 allowlist（仅 Shell 引擎的解释器路径）：
              {(capabilities.command_allowlist ?? []).map((c) => (
                <code key={c}> {c}</code>
              ))}
            </p>
          ) : null}
          {!capabilities?.allow_command ? (
            <p className="settings-note">
              脚本执行已在 ingress.yaml 关闭（<code>admin.jobs.allow_command: false</code>）。
            </p>
          ) : null}
          {customItems.length === 0 ? (
            <div className="panel">
              <div className="panel-body empty-cell">
                暂无自定义任务。点击「新建任务」，先选择「接口调用」或「脚本执行」。
              </div>
            </div>
          ) : (
            customItems.map((job) => (
              <div key={job.id} className="panel jobs-list-item">
                <div className="jobs-list-head">
                  <div>
                    <h2>{job.name}</h2>
                    <div className="sub mono">
                      {job.id} · {jobTypeLabel(job)} · {job.schedule} · {job.enabled ? '已启用' : '已停用'}
                    </div>
                  </div>
                  <div className="jobs-actions">
                    <button type="button" className="btn btn-sm btn-ghost" onClick={() => openEdit(job)}>
                      编辑
                    </button>
                    <button
                      type="button"
                      className="btn btn-sm btn-ghost"
                      onClick={() => openHistory({ source: 'config', jobId: job.id, jobName: job.name })}
                    >
                      <History size={14} aria-hidden /> 执行历史
                    </button>
                    <button
                      type="button"
                      className="btn btn-sm btn-ghost"
                      disabled={runningId === `config:${job.id}`}
                      onClick={() => runJob('config', job.id)}
                    >
                      <Play size={14} aria-hidden /> 执行
                    </button>
                  </div>
                </div>
              </div>
            ))
          )}
        </div>
      )}

      {tab === 'builtins' && (
        <div className="jobs-builtin-grid">
          {(data?.builtins ?? []).map((job) => {
            const draft = builtinDraft[job.id]
            return (
              <div key={job.id} className="panel jobs-card">
                <div className="panel-head">
                  <h2>{job.name}</h2>
                  <span className="chart-hint">内置 · {job.kind}</span>
                </div>
                <div className="panel-body">
                  <p className="jobs-desc">{job.description}</p>
                  <FormGrid columns={1}>
                    <FormCheckbox
                      label="启用"
                      checked={draft?.enabled ?? job.enabled}
                      onChange={(enabled) =>
                        setBuiltinDraft((d) => ({
                          ...d,
                          [job.id]: {
                            ...d[job.id],
                            enabled,
                            schedule: d[job.id]?.schedule ?? job.schedule,
                          },
                        }))
                      }
                    />
                    <CronScheduleField
                      value={draft?.schedule ?? job.schedule}
                      onChange={(schedule) =>
                        setBuiltinDraft((d) => ({
                          ...d,
                          [job.id]: {
                            ...d[job.id],
                            schedule,
                            enabled: d[job.id]?.enabled ?? job.enabled,
                          },
                        }))
                      }
                    />
                    {job.params.retain_days != null || draft?.retain_days != null ? (
                      <FormField
                        label="保留天数"
                        type="number"
                        min={1}
                        full
                        value={draft?.retain_days ?? job.params.retain_days ?? 30}
                        onChange={(e) =>
                          setBuiltinDraft((d) => ({
                            ...d,
                            [job.id]: {
                              ...d[job.id],
                              enabled: d[job.id]?.enabled ?? job.enabled,
                              schedule: d[job.id]?.schedule ?? job.schedule,
                              retain_days: Number(e.target.value),
                            },
                          }))
                        }
                      />
                    ) : null}
                  </FormGrid>
                  <div className="jobs-actions">
                    <button type="button" className="btn btn-sm" onClick={() => saveBuiltin(job.id)}>
                      保存
                    </button>
                    <button
                      type="button"
                      className="btn btn-sm btn-ghost"
                      onClick={() => openHistory({ source: 'builtin', jobId: job.id, jobName: job.name })}
                    >
                      <History size={14} aria-hidden /> 执行历史
                    </button>
                    <button
                      type="button"
                      className="btn btn-sm btn-ghost"
                      disabled={runningId === `builtin:${job.id}`}
                      onClick={() => runJob('builtin', job.id)}
                    >
                      <Play size={14} aria-hidden /> 立即执行
                    </button>
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      )}

      <JobFormDrawer
        open={formOpen}
        editJob={editJob}
        capabilities={capabilities}
        onClose={() => setFormOpen(false)}
        onSaved={() => {
          show(editJob ? '已更新任务' : '已创建任务')
          load()
        }}
        onDeleted={() => {
          show('已删除')
          load()
        }}
      />

      <JobHistoryDrawer
        open={historyTarget != null}
        source={historyTarget?.source ?? 'config'}
        jobId={historyTarget?.jobId ?? ''}
        jobName={historyTarget?.jobName}
        refreshKey={historyRefresh}
        onClose={() => setHistoryTarget(null)}
      />

      {toast && <ToastContainer message={toast.message} type={toast.type} onDone={clear} />}
    </div>
  )
}
