import { Link, useNavigate } from 'react-router-dom'
import { useRef, useCallback, useEffect, useState } from 'react'
import { RulesEditor, type RulesEditorHandle } from '../config/RulesEditor'
import { PublishModal } from '../PublishModal'
import { SavePublishDrawer } from '../SavePublishDrawer'
import { useIngressConfigModule } from '../../hooks/useIngressConfigModule'
import { parseModuleDoc } from '../../lib/ingressModuleForms'
import { buildPersistDiff } from '../../lib/configPersistDiff'
import { serviceFormListFromDoc } from '../../lib/services'
import { api } from '../../api/client'

type Props = {
  onPublished?: () => void
  onSaveError?: (message: string) => void
  onSaveSuccess?: (message: string) => void
}

export function RouteRulesManageTab({ onPublished, onSaveError, onSaveSuccess }: Props) {
  const navigate = useNavigate()
  const rulesEditorRef = useRef<RulesEditorHandle>(null)
  const {
    configPath,
    doc,
    setDoc,
    loading,
    saving,
    dirty,
    save,
    load,
    savedYAML,
    savedDoc,
    buildMergedContent,
  } = useIngressConfigModule('rules')
  const [servicesDoc, setServicesDoc] = useState<Record<string, unknown>>({})
  const [publishOpen, setPublishOpen] = useState(false)
  const [publishContent, setPublishContent] = useState('')
  const [savePublishOpen, setSavePublishOpen] = useState(false)
  const [diffHtml, setDiffHtml] = useState('')

  const loadServices = useCallback(() => {
    api
      .getConfig()
      .then((cfg) =>
        api.configModules(cfg.content).then((modules) => {
          const svc = modules.find((m) => m.id === 'services')
          setServicesDoc(parseModuleDoc(svc?.yaml ?? ''))
        }),
      )
      .catch(() => setServicesDoc({}))
  }, [])

  useEffect(() => {
    loadServices()
  }, [loadServices])

  const serviceCatalog = serviceFormListFromDoc(servicesDoc)

  const onSave = async () => {
    const errMsg = await save()
    if (!errMsg) onSaveSuccess?.('已保存路由规则')
    else onSaveError?.(errMsg)
  }

  const openSavePublish = async () => {
    try {
      const merged = await buildMergedContent()
      setPublishContent(merged)
      setDiffHtml(
        buildPersistDiff({
          savedYAML,
          nextYAML: merged,
          savedDoc,
          doc,
          moduleLabel: '路由规则',
        }),
      )
      setSavePublishOpen(true)
    } catch (e: unknown) {
      onSaveError?.(e instanceof Error ? e.message : String(e))
    }
  }

  const onSaveOnly = async () => {
    setSavePublishOpen(false)
    await onSave()
  }

  const onSaveAndPublish = () => {
    setSavePublishOpen(false)
    setPublishOpen(true)
  }

  const actionsDisabled = !dirty || saving

  return (
    <div className="panel">
      <div className="panel-head">
        <h2>路由规则</h2>
        <div className="toolbar">
          {dirty ? <span className="config-draft-badge">未保存</span> : null}
          <button
            type="button"
            className="btn btn-sm btn-ghost"
            disabled={loading}
            onClick={() => rulesEditorRef.current?.openAdd()}
          >
            + 添加
          </button>
          <button
            type="button"
            className="btn btn-sm btn-primary"
            disabled={loading || actionsDisabled}
            onClick={() => void openSavePublish()}
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
            <RulesEditor
              ref={rulesEditorRef}
              doc={doc}
              onChange={setDoc}
              serviceCatalog={serviceCatalog}
              serviceFieldMode="catalog-select"
              hideTableChrome
              onOpenDetail={(ri, pi = -1) => navigate(`/routes/${ri}/${pi}`)}
            />
            <p className="form-hint" style={{ marginTop: '1rem' }}>
              与 <Link to="/config">配置 → 路由规则</Link> 编辑同一模块；也可在{' '}
              <Link to="/services">服务</Link> 维护 upstream 目录后在 backend 中选用。
            </p>
          </>
        )}
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
          loadServices()
          onPublished?.()
          onSaveSuccess?.('配置已发布并 reload')
        }}
      />
    </div>
  )
}
