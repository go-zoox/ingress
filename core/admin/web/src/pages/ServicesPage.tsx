import { RefreshCw } from 'lucide-react'
import { Link, useNavigate } from 'react-router-dom'
import { useCallback, useEffect, useRef, useState } from 'react'
import { PageHeader } from '../components/PageHeader'
import { PublishModal } from '../components/PublishModal'
import { SavePublishDrawer } from '../components/SavePublishDrawer'
import { ServiceRouteSyncConflictsModal } from '../components/services/ServiceRouteSyncConflictsModal'
import { ServicesEditor, type ServicesEditorHandle } from '../components/config/ServicesEditor'
import { ToastContainer, useToast } from '../components/Toast'
import { useIngressConfigModule } from '../hooks/useIngressConfigModule'
import { serviceDetailLink } from '../lib/deepLinks'
import { buildPersistDiff } from '../lib/configPersistDiff'
import {
  detectServiceRouteConflicts,
  formatServiceRouteSyncMessage,
  type ServiceRouteConflict,
  type ServiceRouteSyncResolutionMap,
  type ServiceRouteSyncResult,
} from '../lib/serviceRouteSync'
import { buildServicesPersistContent } from '../lib/serviceRouteSyncApi'
import { serviceUsageCount, servicesFromDoc } from '../lib/services'
import { parseModuleDoc } from '../lib/ingressModuleForms'
import { api } from '../api/client'

export function ServicesPage() {
  const navigate = useNavigate()
  const servicesEditorRef = useRef<ServicesEditorHandle>(null)
  const {
    configPath,
    doc,
    setDoc,
    savedDoc,
    savedYAML,
    loading,
    err,
    setErr,
    dirty,
    load,
  } = useIngressConfigModule('services')
  const { toast, show, clear } = useToast()
  const [rulesDoc, setRulesDoc] = useState<Record<string, unknown>>({})
  const [publishOpen, setPublishOpen] = useState(false)
  const [publishContent, setPublishContent] = useState('')
  const [busy, setBusy] = useState(false)
  const [conflictOpen, setConflictOpen] = useState(false)
  const [conflicts, setConflicts] = useState<ServiceRouteConflict[]>([])
  const [savePublishOpen, setSavePublishOpen] = useState(false)
  const [diffHtml, setDiffHtml] = useState('')
  const [pendingContent, setPendingContent] = useState('')
  const [pendingSync, setPendingSync] = useState<ServiceRouteSyncResult | null>(null)

  const loadRules = useCallback(() => {
    api
      .getConfig()
      .then((cfg) =>
        api.configModules(cfg.content).then((modules) => {
          const rules = modules.find((m) => m.id === 'rules')
          setRulesDoc(parseModuleDoc(rules?.yaml ?? ''))
        }),
      )
      .catch(() => setRulesDoc({}))
  }, [])

  useEffect(() => {
    loadRules()
  }, [loadRules, doc])

  const loadSavedRulesDoc = useCallback(async () => {
    const modules = await api.configModules(savedYAML)
    return parseModuleDoc(modules.find((m) => m.id === 'rules')?.yaml ?? '')
  }, [savedYAML])

  const proceedWithResolutions = useCallback(
    async (resolutions: ServiceRouteSyncResolutionMap) => {
      setBusy(true)
      setErr('')
      try {
        const plan = await buildServicesPersistContent(savedYAML, doc, savedDoc, resolutions)
        setPendingContent(plan.content)
        setPendingSync(plan.sync)
        setDiffHtml(
          buildPersistDiff({
            savedYAML,
            nextYAML: plan.content,
            savedDoc,
            doc,
            moduleLabel: '服务',
          }),
        )
        setSavePublishOpen(true)
      } catch (e: unknown) {
        const msg = e instanceof Error ? e.message : String(e)
        setErr(msg)
        show(msg, 'error')
      } finally {
        setBusy(false)
      }
    },
    [doc, savedDoc, savedYAML, setErr, show],
  )

  const startPersistFlow = useCallback(async () => {
    setBusy(true)
    setErr('')
    try {
      const rulesDocFromSaved = await loadSavedRulesDoc()
      const found = detectServiceRouteConflicts(
        rulesDocFromSaved,
        servicesFromDoc(savedDoc),
        servicesFromDoc(doc),
      )
      if (found.length > 0) {
        setConflicts(found)
        setConflictOpen(true)
        return
      }
      await proceedWithResolutions({})
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e)
      setErr(msg)
      show(msg, 'error')
    } finally {
      setBusy(false)
    }
  }, [doc, savedDoc, loadSavedRulesDoc, proceedWithResolutions, setErr, show])

  const onConflictConfirm = async (resolutions: ServiceRouteSyncResolutionMap) => {
    setConflictOpen(false)
    await proceedWithResolutions(resolutions)
  }

  const onSaveOnly = async () => {
    if (!pendingContent) return
    setBusy(true)
    setErr('')
    try {
      await api.validateConfig(pendingContent)
      await api.putConfig(pendingContent, 'save services')
      await load()
      loadRules()
      setSavePublishOpen(false)
      const syncMsg = formatServiceRouteSyncMessage(pendingSync ?? { touched: 0, skipped: 0, details: [] })
      show(syncMsg ? `已保存服务目录；${syncMsg}` : '已保存服务目录')
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e)
      setErr(msg)
      show(msg, 'error')
    } finally {
      setBusy(false)
    }
  }

  const onSaveAndPublish = () => {
    if (!pendingContent) return
    setSavePublishOpen(false)
    setPublishContent(pendingContent)
    setPublishOpen(true)
  }

  const actionsDisabled = !dirty || busy

  const serviceCount = Array.isArray(doc.services) ? doc.services.length : 0

  return (
    <div className="page">
      <PageHeader
        title="服务"
        desc="可复用的 upstream Service 目录；配置路由 backend 时从此选择"
        actions={
          <button type="button" className="btn btn-sm" onClick={() => void load()} disabled={loading}>
            <RefreshCw size={14} aria-hidden /> 刷新
          </button>
        }
      />
      {err && <p className="err">{err}</p>}

      <div className="cards">
        <div className="card">
          <div className="label">服务数量</div>
          <div className="value">{serviceCount}</div>
          <div className="sub">ingress.yaml services[]</div>
        </div>
        <div className="card">
          <div className="label">配置文件</div>
          <div className="value" style={{ fontSize: '0.85rem', wordBreak: 'break-all' }}>
            {configPath || '—'}
          </div>
          <div className="sub">
            <Link to="/config">打开配置中心</Link>
          </div>
        </div>
      </div>

      <div className="panel">
        <div className="panel-head">
          <h2>服务目录</h2>
          <div className="toolbar">
            {dirty ? <span className="config-draft-badge">未保存</span> : null}
            <button
              type="button"
              className="btn btn-sm btn-ghost"
              disabled={loading}
              onClick={() => servicesEditorRef.current?.openAdd()}
            >
              + 添加
            </button>
            <button
              type="button"
              className="btn btn-sm btn-primary"
              disabled={loading || actionsDisabled}
              onClick={() => void startPersistFlow()}
            >
              保存与发布
            </button>
          </div>
        </div>
        <div className="panel-body panel-table-wrap">
          {loading ? (
            <p className="empty-hint">加载中…</p>
          ) : (
            <>
              <ServicesEditor
                ref={servicesEditorRef}
                doc={doc}
                onChange={setDoc}
                usageForName={(name) => serviceUsageCount(name, rulesDoc)}
                hideTableChrome
                onOpenDetail={(name) => navigate(serviceDetailLink(name))}
              />
              <p className="form-hint" style={{ marginTop: '1rem' }}>
                与 <Link to="/config">配置 → 服务</Link> 编辑同一模块。保存与发布前会展示完整配置 diff；若路由
                backend 曾手动改过且与目录不一致，会提示选择保留或覆盖。在{' '}
                <Link to="/routes">路由</Link> 编辑 backend 时可从目录选用。
              </p>
            </>
          )}
        </div>
      </div>

      <ServiceRouteSyncConflictsModal
        open={conflictOpen}
        conflicts={conflicts}
        onClose={() => setConflictOpen(false)}
        onConfirm={(resolutions) => void onConflictConfirm(resolutions)}
      />

      <SavePublishDrawer
        open={savePublishOpen}
        diffHtml={diffHtml}
        busy={busy}
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
          loadRules()
          const syncMsg = formatServiceRouteSyncMessage(pendingSync ?? { touched: 0, skipped: 0, details: [] })
          show(syncMsg ? `配置已发布并 reload；${syncMsg}` : '配置已发布并 reload')
        }}
      />

      {toast && <ToastContainer message={toast.message} type={toast.type} onDone={clear} />}
    </div>
  )
}
