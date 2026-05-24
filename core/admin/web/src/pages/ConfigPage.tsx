import { useEffect, useRef, useState } from 'react'
import { PageHeader } from '../components/PageHeader'
import { ConfigModulesPanel, type ConfigModulesPanelHandle } from '../components/ConfigModulesPanel'
import { ConfigVersionsPanel } from '../components/ConfigVersionsPanel'
import { DiffModal } from '../components/DiffModal'
import { PreviewModal } from '../components/PreviewModal'
import { PublishModal } from '../components/PublishModal'
import { ToastContainer, useToast } from '../components/Toast'
import { api } from '../api/client'
import { buildDiff, escapeHtml } from '../lib/config'

type ConfigView = 'modules' | 'yaml' | 'versions'

export function ConfigPage() {
  const [content, setContent] = useState('')
  const [saved, setSaved] = useState('')
  const [path, setPath] = useState('')
  const [view, setView] = useState<ConfigView>('modules')
  const [validateOut, setValidateOut] = useState('')
  const [err, setErr] = useState('')
  const [publishOpen, setPublishOpen] = useState(false)
  const [previewOpen, setPreviewOpen] = useState(false)
  const [diffOpen, setDiffOpen] = useState(false)
  const [diffHtml, setDiffHtml] = useState('')
  const { toast, show, clear } = useToast()
  const loaded = useRef(false)
  const modulesRef = useRef<ConfigModulesPanelHandle>(null)

  useEffect(() => {
    api
      .getConfig()
      .then((r) => {
        setContent(r.content)
        setSaved(r.content)
        setPath(r.path)
        loaded.current = true
      })
      .catch((e: Error) => setErr(e.message))
  }, [])

  const validate = () => {
    setErr('')
    setValidateOut('')
    api
      .validateConfig(content)
      .then(() => setValidateOut('<p class="validate-ok">✓ 校验通过</p>'))
      .catch((e: Error) => {
        setValidateOut('<p class="validate-err">' + escapeHtml(e.message) + '</p>')
      })
  }

  const save = async () => {
    setErr('')
    try {
      await modulesRef.current?.autoApplyIfDirty()
      await api.validateConfig(content)
      await api.putConfig(content, 'save')
      setSaved(content)
      show('已保存到 ' + path)
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e)
      setErr(msg)
      show(msg, 'error')
    }
  }

  const showDiff = () => {
    setDiffHtml(buildDiff(saved, content))
    setDiffOpen(true)
  }

  const restoreDraft = (next: string) => {
    setContent(next)
    setValidateOut('')
    show('已加载历史版本到草稿（尚未保存）')
    setView('yaml')
  }

  const dirty = content !== saved

  return (
    <div className="page">
      <PageHeader
        title="配置"
        desc="分模块编辑 ingress.yaml → 预览变更 → 保存版本 → 发布 reload"
      />
      {err && <p className="err">{err}</p>}
      <div className="panel">
        <div className="panel-head config-panel-head">
          <div className="config-view-tabs">
            <button
              type="button"
              className={`config-view-tab ${view === 'modules' ? 'active' : ''}`}
              onClick={() => setView('modules')}
            >
              可视化
            </button>
            <button
              type="button"
              className={`config-view-tab ${view === 'yaml' ? 'active' : ''}`}
              onClick={() => setView('yaml')}
            >
              YAML
            </button>
            <button
              type="button"
              className={`config-view-tab ${view === 'versions' ? 'active' : ''}`}
              onClick={() => setView('versions')}
            >
              版本
            </button>
            {dirty && <span className="config-draft-badge">草稿未保存</span>}
          </div>
          <div className="toolbar">
            <button type="button" className="btn" onClick={validate}>
              校验
            </button>
            <button type="button" className="btn" onClick={showDiff}>
              查看变更
            </button>
            <button type="button" className="btn" onClick={() => setPreviewOpen(true)}>
              预览
            </button>
            <button type="button" className="btn" onClick={save}>
              保存到 YAML
            </button>
            <button type="button" className="btn btn-primary" onClick={async () => {
              try { await modulesRef.current?.autoApplyIfDirty() } catch { /* error already shown */ }
              setPublishOpen(true)
            }}>
              发布
            </button>
          </div>
        </div>
        <div className="panel-body">
          {view === 'modules' && (
            <ConfigModulesPanel
              ref={modulesRef}
              content={content}
              onContentChange={(next) => {
                setContent(next)
                if (loaded.current) setValidateOut('')
              }}
              onError={setErr}
              onSwitchToYaml={() => setView('yaml')}
            />
          )}
          {view === 'yaml' && (
            <>
              <textarea
                className="code"
                spellCheck={false}
                value={content}
                onChange={(e) => {
                  setContent(e.target.value)
                  if (loaded.current) setValidateOut('')
                }}
              />
              <div style={{ marginTop: 12 }} dangerouslySetInnerHTML={{ __html: validateOut }} />
            </>
          )}
          {view === 'versions' && <ConfigVersionsPanel onRestore={restoreDraft} />}
        </div>
      </div>
      <PreviewModal
        open={previewOpen}
        draft={content}
        published={saved}
        onClose={() => setPreviewOpen(false)}
        onPublish={() => setPublishOpen(true)}
      />
      <PublishModal
        open={publishOpen}
        configPath={path}
        content={content}
        onClose={() => setPublishOpen(false)}
        onDone={() => {
          setSaved(content)
          show('配置已发布并 reload')
        }}
      />
      <DiffModal open={diffOpen} diffHtml={diffHtml} onClose={() => setDiffOpen(false)} />
      {toast && <ToastContainer message={toast.message} type={toast.type} onDone={clear} />}
    </div>
  )
}
