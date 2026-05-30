import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Plus, RefreshCw } from 'lucide-react'
import { PageHeader } from '../components/PageHeader'
import { PublishModal } from '../components/PublishModal'
import { SavePublishDrawer } from '../components/SavePublishDrawer'
import { ScenariosActiveSelect } from '../components/scenarios/ScenariosActiveSelect'
import { ScenariosEditor, type ScenariosEditorHandle } from '../components/scenarios/ScenariosEditor'
import { ToastContainer, useToast } from '../components/Toast'
import { api } from '../api/client'
import { useIngressConfigModule } from '../hooks/useIngressConfigModule'
import { buildPersistDiff } from '../lib/configPersistDiff'
import { parseModuleDoc } from '../lib/ingressModuleForms'
import { scenariosFromModuleDoc, validateScenariosFormState, DEFAULT_SCENARIO_LABEL, isDefaultScenario } from '../lib/scenarios'

function hostOptionsFromRulesYAML(yaml: string): string[] {
  const doc = parseModuleDoc(yaml)
  const rules = doc.rules
  if (!Array.isArray(rules)) return []
  const hosts = new Set<string>()
  for (const row of rules) {
    if (row && typeof row === 'object' && 'host' in row) {
      const host = String((row as { host?: string }).host ?? '').trim()
      if (host) hosts.add(host)
    }
  }
  return [...hosts].sort()
}

export function ScenariosPage() {
  const {
    configPath,
    doc,
    setDoc,
    savedDoc,
    savedYAML,
    loading,
    saving,
    err,
    setErr,
    dirty,
    load,
    save,
    buildMergedContent,
  } = useIngressConfigModule('scenarios')

  const [hostOptions, setHostOptions] = useState<string[]>([])
  const [activatingId, setActivatingId] = useState('')
  const [savePublishOpen, setSavePublishOpen] = useState(false)
  const [publishOpen, setPublishOpen] = useState(false)
  const [publishContent, setPublishContent] = useState('')
  const [diffHtml, setDiffHtml] = useState('')
  const editorRef = useRef<ScenariosEditorHandle>(null)
  const { toast, show, clear } = useToast()

  const refreshHosts = useCallback(() => {
    api
      .getConfig()
      .then((cfg) => api.configModules(cfg.content))
      .then((modules) => {
        const rules = modules.find((m) => m.id === 'rules')
        setHostOptions(hostOptionsFromRulesYAML(rules?.yaml ?? ''))
      })
      .catch(() => setHostOptions([]))
  }, [])

  useEffect(() => {
    refreshHosts()
  }, [refreshHosts])

  const state = useMemo(() => scenariosFromModuleDoc(doc), [doc])

  const openSavePublish = async () => {
    const validationErr = validateScenariosFormState(state)
    if (validationErr) {
      show(validationErr, 'error')
      return
    }
    try {
      const merged = await buildMergedContent()
      setPublishContent(merged)
      setDiffHtml(
        buildPersistDiff({
          savedYAML,
          nextYAML: merged,
          savedDoc,
          doc,
          moduleLabel: '场景',
        }),
      )
      setSavePublishOpen(true)
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e)
      setErr(msg)
      show(msg, 'error')
    }
  }

  const onSaveOnly = async () => {
    setSavePublishOpen(false)
    const msg = await save()
    if (msg) show(msg, 'error')
    else show('已保存场景配置')
  }

  const onSaveAndPublish = () => {
    setSavePublishOpen(false)
    setPublishOpen(true)
  }

  const activate = async (id: string) => {
    const label = isDefaultScenario(id)
      ? `${DEFAULT_SCENARIO_LABEL}（根配置）`
      : state.items.find((s) => s.id === id)?.label || id
    if (
      !window.confirm(
        `切换到场景「${label}」？\n\n将更新 scenarios.active 并热加载 ingress。`,
      )
    ) {
      return
    }
    setActivatingId(id)
    setErr('')
    try {
      if (dirty) {
        const validationErr = validateScenariosFormState(state)
        if (validationErr) {
          show(validationErr, 'error')
          return
        }
        const saveErr = await save()
        if (saveErr) {
          show(saveErr, 'error')
          return
        }
      }
      await api.setScenarioActive(id)
      show(`已切换到场景「${label}」并已 reload`)
      await load()
      refreshHosts()
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e)
      setErr(msg)
      show(msg, 'error')
    } finally {
      setActivatingId('')
    }
  }

  const actionsDisabled = !dirty || saving || loading

  return (
    <div className="page">
      <PageHeader
        title="场景管理"
        desc="日常 / 直播等运行场景；支持增删改查与 overlay 可视化配置"
        actions={
          <button
            type="button"
            className="btn btn-sm"
            onClick={() => {
              void load()
              refreshHosts()
            }}
            disabled={loading}
          >
            <RefreshCw size={14} aria-hidden /> 刷新
          </button>
        }
      />
      {err ? <p className="err">{err}</p> : null}

      <div className="panel">
        <div className="panel-head">
          <h2>场景列表</h2>
          <div className="toolbar scenarios-panel-toolbar">
            {dirty ? <span className="config-draft-badge">未保存</span> : null}
            <ScenariosActiveSelect doc={doc} onChange={setDoc} disabled={loading} />
            <button
              type="button"
              className="btn btn-sm btn-primary"
              disabled={loading}
              onClick={() => editorRef.current?.openCreate()}
            >
              <Plus size={14} aria-hidden /> 新建场景
            </button>
            <button
              type="button"
              className="btn btn-sm btn-primary"
              disabled={actionsDisabled}
              onClick={() => void openSavePublish()}
            >
              保存与发布
            </button>
          </div>
        </div>
        <div className="panel-body">
          {loading ? (
            <p className="empty-hint">加载中…</p>
          ) : (
            <ScenariosEditor
              ref={editorRef}
              doc={doc}
              onChange={setDoc}
              hostOptions={hostOptions}
              activatingId={activatingId}
              onActivate={activate}
              onNotify={show}
              showToolbar={false}
            />
          )}
        </div>
      </div>

      <SavePublishDrawer
        open={savePublishOpen}
        diffHtml={diffHtml}
        busy={saving}
        onClose={() => setSavePublishOpen(false)}
        onSaveOnly={() => void onSaveOnly()}
        onSaveAndPublish={onSaveAndPublish}
      />

      <PublishModal
        open={publishOpen}
        configPath={configPath}
        content={publishContent}
        onClose={() => setPublishOpen(false)}
        onDone={() => {
          void load()
          refreshHosts()
          show('已发布场景配置并 reload')
        }}
      />

      {toast && <ToastContainer message={toast.message} type={toast.type} onDone={clear} />}
    </div>
  )
}
