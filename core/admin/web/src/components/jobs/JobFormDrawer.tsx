import { useEffect, useState } from 'react'
import { CronScheduleField } from './CronScheduleField'
import { Drawer } from '../Drawer'
import {
  FormCheckbox,
  FormCodeEditorField,
  FormField,
  FormGrid,
  FormSelectField,
  FormTextareaField,
} from '../Form'
import {
  api,
  type JobItemInput,
  type JobView,
  type JobsListResult,
} from '../../api/client'
import {
  DEFAULT_JOB_SCRIPT,
  SCRIPT_ENGINES,
  SCRIPT_SHELL_PRESETS,
  defaultScriptEngine,
  defaultScriptShell,
  isPresetShell,
  normalizeScriptEngine,
  scriptEngineHint,
  scriptEngineKindLabel,
  scriptParamsFromJob,
  scriptParamsToJob,
  scriptEditorHint,
  scriptWhenEngineChanges,
  type ScriptEngine,
} from '../../lib/scriptParams'

type CustomKind = 'http_call' | 'script' | ''
type CreateStep = 'kind' | 'form'

const emptyCustomForm = (): JobItemInput => ({
  name: '',
  kind: 'http_call',
  schedule: '0 * * * *',
  enabled: true,
  timeout_sec: 60,
  on_failure: 'log',
  params: { method: 'GET', url: '' },
})

function kindLabel(kind: string, engine?: ScriptEngine) {
  if (kind === 'http_call') return '接口调用'
  if (kind === 'script' || kind === 'command') {
    return engine ? scriptEngineKindLabel(engine) : '脚本执行'
  }
  return kind
}

function shellStateFromParams(params: JobView['params']) {
  const scriptFields = scriptParamsFromJob(params)
  if (scriptFields.engine === 'shell' && scriptFields.shell) {
    const shell = scriptFields.shell
    if (isPresetShell(shell)) {
      return { shellPreset: shell, customShell: '' }
    }
    return { shellPreset: 'custom', customShell: shell }
  }
  return { shellPreset: defaultScriptShell(), customShell: '' }
}

function customFormFromJob(job: JobView): JobItemInput {
  const kind = (job.kind === 'command' ? 'script' : job.kind) as 'http_call' | 'script'
  if (kind === 'script') {
    const scriptFields = scriptParamsFromJob(job.params)
    return {
      id: job.id,
      name: job.name,
      kind: 'script',
      schedule: job.schedule,
      enabled: job.enabled,
      timeout_sec: job.timeout_sec || 60,
      on_failure: job.on_failure || 'log',
      params: scriptParamsToJob(scriptFields),
    }
  }
  return {
    id: job.id,
    name: job.name,
    kind: 'http_call',
    schedule: job.schedule,
    enabled: job.enabled,
    timeout_sec: job.timeout_sec || 60,
    on_failure: job.on_failure || 'log',
    params: { ...job.params },
  }
}

type Props = {
  open: boolean
  editJob: JobView | null
  capabilities: JobsListResult['capabilities'] | undefined
  onClose: () => void
  onSaved: () => void
  onDeleted: () => void
}

export function JobFormDrawer({ open, editJob, capabilities, onClose, onSaved, onDeleted }: Props) {
  const editId = editJob?.id ?? null
  const [createStep, setCreateStep] = useState<CreateStep>('kind')
  const [createKind, setCreateKind] = useState<CustomKind>('')
  const [customForm, setCustomForm] = useState<JobItemInput>(emptyCustomForm())
  const [shellPreset, setShellPreset] = useState<string>(defaultScriptShell())
  const [customShell, setCustomShell] = useState('')
  const [saving, setSaving] = useState(false)
  const [deleteConfirm, setDeleteConfirm] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [err, setErr] = useState('')
  /** Bumps after form state is hydrated so CodeMirror mounts with the loaded script. */
  const [editorSession, setEditorSession] = useState(0)

  useEffect(() => {
    if (!open) {
      setEditorSession(0)
      return
    }
    setErr('')
    setDeleteConfirm(false)
    setSaving(false)
    setDeleting(false)
    if (editJob) {
      const kind = (editJob.kind === 'command' ? 'script' : editJob.kind) as 'http_call' | 'script'
      setCreateStep('form')
      setCreateKind(kind)
      if (kind === 'script') {
        const shellState = shellStateFromParams(editJob.params)
        setShellPreset(shellState.shellPreset)
        setCustomShell(shellState.customShell)
      } else {
        setShellPreset(defaultScriptShell())
        setCustomShell('')
      }
      setCustomForm(customFormFromJob(editJob))
    } else {
      setCreateStep('kind')
      setCreateKind('')
      setShellPreset(defaultScriptShell())
      setCustomShell('')
      setCustomForm(emptyCustomForm())
    }
    setEditorSession((s) => s + 1)
  }, [open, editJob])

  const pickKind = (kind: CustomKind) => {
    setCreateKind(kind)
    setCreateStep('form')
    const base = emptyCustomForm()
    base.kind = kind as 'http_call' | 'script'
    if (kind === 'script') {
      setShellPreset(defaultScriptShell())
      setCustomShell('')
      const engine = defaultScriptEngine()
      base.params = scriptParamsToJob({
        engine,
        script: DEFAULT_JOB_SCRIPT[engine],
        shell: defaultScriptShell(),
      })
    }
    setCustomForm(base)
    setEditorSession((s) => s + 1)
  }

  const effectiveShell = () => (shellPreset === 'custom' ? customShell.trim() : shellPreset)

  const saveCustom = async () => {
    if (!customForm.name.trim()) {
      setErr('请填写名称')
      return
    }
    const body = { ...customForm, name: customForm.name.trim() }
    if (body.kind === 'script') {
      if (!body.params.script?.trim()) {
        setErr('请填写脚本内容')
        return
      }
      const engine = normalizeScriptEngine(body.params.engine)
      if (engine === 'shell') {
        const shell = effectiveShell()
        if (!shell) {
          setErr('请选择 Shell 类型')
          return
        }
        body.params = scriptParamsToJob({
          engine,
          script: body.params.script,
          shell,
          workdir: body.params.workdir,
        })
      } else {
        body.params = scriptParamsToJob({
          engine,
          script: body.params.script,
          workdir: body.params.workdir,
        })
      }
    }
    if (!editId) {
      delete body.id
    }
    setSaving(true)
    setErr('')
    try {
      if (editId) {
        await api.updateJobItem(editId, body)
      } else {
        await api.createJobItem(body)
      }
      onSaved()
      onClose()
    } catch (e: unknown) {
      setErr(e instanceof Error ? e.message : String(e))
    } finally {
      setSaving(false)
    }
  }

  const deleteCustom = async () => {
    if (!editId) return
    setDeleting(true)
    setErr('')
    try {
      await api.deleteJobItem(editId)
      onDeleted()
      onClose()
    } catch (e: unknown) {
      setErr(e instanceof Error ? e.message : String(e))
    } finally {
      setDeleting(false)
    }
  }

  const showForm = editId != null || createStep === 'form'
  const title = deleteConfirm
    ? '确认删除'
    : editId
      ? '编辑任务'
      : showForm
        ? '新建任务'
        : '新建任务'

  const footer =
    deleteConfirm && editId ? (
      <>
        <button type="button" className="btn btn-ghost" onClick={() => setDeleteConfirm(false)} disabled={deleting}>
          取消
        </button>
        <button type="button" className="btn btn-danger" onClick={deleteCustom} disabled={deleting}>
          {deleting ? '删除中…' : '确认删除'}
        </button>
      </>
    ) : showForm ? (
      <>
        {editId ? (
          <button type="button" className="btn btn-ghost btn-danger" onClick={() => setDeleteConfirm(true)} disabled={saving}>
            删除
          </button>
        ) : null}
        <button type="button" className="btn btn-ghost" onClick={onClose} disabled={saving}>
          取消
        </button>
        <button type="button" className="btn btn-primary" onClick={saveCustom} disabled={saving}>
          {saving ? '保存中…' : '保存'}
        </button>
      </>
    ) : (
      <button type="button" className="btn btn-ghost" onClick={onClose}>
        取消
      </button>
    )

  const kind = createKind || customForm.kind
  const scriptEngine = normalizeScriptEngine(customForm.params.engine)
  const drawerWidth = kind === 'script' ? 720 : 560

  return (
    <Drawer open={open} title={title} onClose={onClose} width={drawerWidth} footer={footer}>
      {err ? <p className="err">{err}</p> : null}

      {deleteConfirm && editId ? (
        <div className="jobs-delete-confirm">
          <p>确定删除任务 <code>{editId}</code>？此操作不可撤销。</p>
        </div>
      ) : !editId && createStep === 'kind' ? (
        <div className="jobs-kind-picker">
          <p className="form-hint">请先选择任务类型，再填写调度与参数。</p>
          <button type="button" className="jobs-kind-card" onClick={() => pickKind('http_call')}>
            <strong>接口调用</strong>
            <span>向 HTTP(S) 地址发送请求，记录状态码、响应头与响应体</span>
          </button>
          <button
            type="button"
            className={`jobs-kind-card${capabilities?.allow_command ? '' : ' disabled'}`}
            disabled={!capabilities?.allow_command}
            onClick={() => pickKind('script')}
          >
            <strong>脚本执行</strong>
            <span>
              {capabilities?.allow_command
                ? capabilities.command_restricted
                  ? 'Shell 须在 command_allowlist 中'
                  : '支持 Shell / JavaScript(goja) / Go(yaegi) 内置解释器'
                : capabilities?.command_reason || '脚本执行已关闭'}
            </span>
          </button>
        </div>
      ) : showForm ? (
        <FormGrid columns={1}>
          {!editId ? (
            <p className="form-hint">
              <button type="button" className="btn btn-ghost btn-sm" onClick={() => setCreateStep('kind')}>
                ← 重新选择类型
              </button>
              <span style={{ marginLeft: 8 }}>类型：{kindLabel(kind, scriptEngine)}</span>
            </p>
          ) : (
            <p className="form-hint">类型：{kindLabel(kind, scriptEngine)}</p>
          )}

          {editId ? (
            <FormField label="任务 ID" hint="系统自动生成，不可修改" full value={editId} readOnly />
          ) : null}

          <FormField
            label="名称"
            full
            value={customForm.name}
            onChange={(e) => setCustomForm({ ...customForm, name: e.target.value })}
          />

          <CronScheduleField
            value={customForm.schedule}
            onChange={(schedule) => setCustomForm({ ...customForm, schedule })}
          />

          <FormField
            label="超时（秒）"
            type="number"
            min={1}
            full
            value={customForm.timeout_sec ?? 60}
            onChange={(e) => setCustomForm({ ...customForm, timeout_sec: Number(e.target.value) })}
          />

          <FormCheckbox
            label="启用"
            checked={customForm.enabled}
            onChange={(enabled) => setCustomForm({ ...customForm, enabled })}
          />

          {kind === 'http_call' && (
            <>
              <FormSelectField
                label="Method"
                full
                value={customForm.params.method || 'GET'}
                onChange={(e) =>
                  setCustomForm({
                    ...customForm,
                    params: { ...customForm.params, method: e.target.value },
                  })
                }
              >
                {['GET', 'POST', 'PUT', 'DELETE'].map((m) => (
                  <option key={m} value={m}>
                    {m}
                  </option>
                ))}
              </FormSelectField>
              <FormField
                label="URL"
                full
                value={customForm.params.url || ''}
                onChange={(e) =>
                  setCustomForm({
                    ...customForm,
                    params: { ...customForm.params, url: e.target.value },
                  })
                }
              />
              <FormTextareaField
                label="Body（可选）"
                full
                mono
                rows={4}
                value={customForm.params.body || ''}
                onChange={(e) =>
                  setCustomForm({
                    ...customForm,
                    params: { ...customForm.params, body: e.target.value },
                  })
                }
              />
            </>
          )}

          {kind === 'script' && (
            <>
              <FormSelectField
                label="脚本引擎"
                hint={scriptEngineHint(scriptEngine)}
                full
                value={scriptEngine}
                onChange={(e) => {
                  const engine = normalizeScriptEngine(e.target.value) as ScriptEngine
                  const nextScript = scriptWhenEngineChanges(
                    customForm.params.script || '',
                    scriptEngine,
                    engine,
                  )
                  if (engine === 'shell') {
                    setShellPreset(defaultScriptShell())
                    setCustomShell('')
                  }
                  setCustomForm({
                    ...customForm,
                    params: scriptParamsToJob({
                      engine,
                      script: nextScript,
                      shell: engine === 'shell' ? defaultScriptShell() : undefined,
                      workdir: customForm.params.workdir,
                    }),
                  })
                }}
              >
                {SCRIPT_ENGINES.map((item) => (
                  <option key={item.value} value={item.value}>
                    {item.label}
                  </option>
                ))}
              </FormSelectField>
              {scriptEngine === 'shell' ? (
                <>
                  <FormSelectField
                    label="Shell 类型"
                    hint="仅 Shell 引擎需要；allowlist 限制的是解释器可执行文件路径"
                    full
                    value={shellPreset}
                    onChange={(e) => setShellPreset(e.target.value)}
                  >
                    {SCRIPT_SHELL_PRESETS.map((shell) => (
                      <option key={shell} value={shell}>
                        {shell}
                      </option>
                    ))}
                    <option value="custom">自定义路径</option>
                  </FormSelectField>
                  {shellPreset === 'custom' ? (
                    <FormField
                      label="Shell 解释器路径"
                      hint="例如 /bin/bash"
                      full
                      value={customShell}
                      onChange={(e) => setCustomShell(e.target.value)}
                    />
                  ) : null}
                </>
              ) : null}
              {scriptEngine !== 'shell' ? (
                <p className="form-hint">{scriptEditorHint(scriptEngine)}</p>
              ) : null}
              {editorSession > 0 ? (
                <FormCodeEditorField
                  editorKey={editorSession}
                  label="脚本内容"
                  hint={
                    scriptEngine === 'shell'
                      ? '保存为 params.script，由 Shell 以 -c 方式执行'
                      : '可用 API 见上方说明'
                  }
                  full
                  minHeight="220px"
                  language={scriptEngine}
                  value={customForm.params.script || ''}
                  onChange={(script) =>
                    setCustomForm({
                      ...customForm,
                      params: { ...customForm.params, script },
                    })
                  }
                />
              ) : null}
              <FormField
                label="工作目录（可选）"
                full
                value={customForm.params.workdir || ''}
                onChange={(e) =>
                  setCustomForm({
                    ...customForm,
                    params: { ...customForm.params, workdir: e.target.value },
                  })
                }
              />
            </>
          )}
        </FormGrid>
      ) : null}
    </Drawer>
  )
}
