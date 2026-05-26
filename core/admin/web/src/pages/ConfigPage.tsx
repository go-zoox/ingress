import { useEffect, useRef, useState } from 'react'
import { PageHeader } from '../components/PageHeader'
import { ConfigModulesPanel, type ConfigModulesPanelHandle } from '../components/ConfigModulesPanel'
import { ConfigVersionsPanel } from '../components/ConfigVersionsPanel'
import { ConfigGovernanceBanner } from '../components/ConfigGovernanceBanner'
import { ConfigChangeTimeline } from '../components/ConfigChangeTimeline'
import { RollbackConfirmModal } from '../components/RollbackConfirmModal'
import { api, type ConfigRevisionSummary, type IngressStatus } from '../api/client'
import { DiffModal } from '../components/DiffModal'
import { PreviewModal } from '../components/PreviewModal'
import { PublishModal } from '../components/PublishModal'
import { ToastContainer, useToast } from '../components/Toast'
import { buildDiff, escapeHtml } from '../lib/config'
import { useUndo } from '../hooks/useUndo'

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
  const [gov, setGov] = useState<IngressStatus | null>(null)
  const [rollbackRevision, setRollbackRevision] = useState<ConfigRevisionSummary | null>(null)
  const { toast, show, clear } = useToast()
  const loaded = useRef(false)
  const modulesRef = useRef<ConfigModulesPanelHandle>(null)

  // Undo/redo hook
  const undoState = useUndo('')

  useEffect(() => {
    api
      .getConfig()
      .then((r) => {
        setContent(r.content)
        setSaved(r.content)
        setPath(r.path)
        undoState.reset(r.content)
        loaded.current = true
      })
      .catch((e: Error) => setErr(e.message))
    api.status().then(setGov).catch(() => setGov(null))
  }, [])

  const refreshGovernance = () => {
    api.status().then(setGov).catch(() => setGov(null))
  }

  // Keyboard shortcuts for undo/redo
  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      // Skip if user is not in YAML view or focus is outside the textarea
      if (view !== 'yaml') return

      const isUndo = (e.ctrlKey || e.metaKey) && e.key === 'z' && !e.shiftKey
      const isRedo = (e.ctrlKey || e.metaKey) && e.key === 'z' && e.shiftKey

      if (isUndo) {
        e.preventDefault()
        const val = undoState.undo()
        if (val !== '') {
          setContent(val)
          setValidateOut('')
        }
      } else if (isRedo) {
        e.preventDefault()
        const val = undoState.redo()
        if (val !== '') {
          setContent(val)
          setValidateOut('')
        }
      }
    }

    document.addEventListener('keydown', onKeyDown)
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [view, undoState])

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
      undoState.reset(content)
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
    undoState.reset(next)
    setValidateOut('')
    show('已加载历史版本到草稿（尚未保存）')
    setView('yaml')
  }

  const dirty = content !== saved

  // Count changes from saved state
  const changeCount = undoState.canUndo ? 1 : 0

  return (
    <div className="page">
      <PageHeader
        title="配置"
        desc="分模块编辑 ingress.yaml → 预览变更 → 保存版本 → 发布 reload"
      />
      {err && <p className="err">{err}</p>}
      <ConfigGovernanceBanner
        runtimeDrift={gov?.runtime_drift}
        revisionDrift={gov?.revision_drift}
        reloadReady={gov?.reload_ready}
        fileHash={gov?.file_hash || gov?.config_hash}
        runtimeHash={gov?.runtime_hash}
        latestRevisionHash={gov?.latest_revision_hash}
        onReload={async () => {
          try {
            await api.reload()
            show('已触发 reload')
            refreshGovernance()
          } catch (e: unknown) {
            show(e instanceof Error ? e.message : String(e), 'error')
          }
        }}
      />
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
              版本与变更
            </button>
            {dirty && (
              <span className="config-draft-badge">
                草稿未保存{changeCount > 0 ? '（有变更）' : ''}
              </span>
            )}
          </div>
          <div className="toolbar">
            <button
              type="button"
              className="btn btn-undo"
              disabled={!undoState.canUndo}
              onClick={() => {
                const val = undoState.undo()
                if (val !== '') {
                  setContent(val)
                  setValidateOut('')
                }
              }}
              title="撤销 (Ctrl+Z)"
            >
              ↩ 撤销
            </button>
            <button
              type="button"
              className="btn btn-redo"
              disabled={!undoState.canRedo}
              onClick={() => {
                const val = undoState.redo()
                if (val !== '') {
                  setContent(val)
                  setValidateOut('')
                }
              }}
              title="重做 (Ctrl+Shift+Z)"
            >
              ↪ 重做
            </button>
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
                undoState.push(next)
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
                  const val = e.target.value
                  setContent(val)
                  undoState.push(val)
                  if (loaded.current) setValidateOut('')
                }}
              />
              <div style={{ marginTop: 12 }} dangerouslySetInnerHTML={{ __html: validateOut }} />
            </>
          )}
          {view === 'versions' && (
            <>
              <ConfigChangeTimeline
                onRestore={restoreDraft}
                onRollback={(rev) => setRollbackRevision(rev)}
              />
              <ConfigVersionsPanel onRestore={restoreDraft} />
            </>
          )}
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
          undoState.reset(content)
          show('配置已发布并 reload')
          refreshGovernance()
        }}
      />
      {rollbackRevision ? (
        <RollbackConfirmModal
          revision={rollbackRevision}
          onConfirm={() => {
            setRollbackRevision(null)
            api.getConfig().then((r) => {
              setContent(r.content)
              setSaved(r.content)
              undoState.reset(r.content)
            })
            refreshGovernance()
            show('已回滚并 reload')
          }}
          onCancel={() => setRollbackRevision(null)}
        />
      ) : null}
      <DiffModal open={diffOpen} diffHtml={diffHtml} onClose={() => setDiffOpen(false)} />
      {toast && <ToastContainer message={toast.message} type={toast.type} onDone={clear} />}
    </div>
  )
}
