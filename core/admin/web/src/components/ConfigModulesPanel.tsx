import { forwardRef, useEffect, useImperativeHandle, useRef, useState } from 'react'
import { api, type ConfigModule } from '../api/client'
import { ConfigModuleForm } from './ConfigModuleForm'

export interface ConfigModulesPanelHandle {
  autoApplyIfDirty: () => Promise<void>
}

export const ConfigModulesPanel = forwardRef<ConfigModulesPanelHandle, {
  content: string
  onContentChange: (next: string) => void
  onError: (message: string) => void
  onSwitchToYaml?: () => void
}>(function ConfigModulesPanel({ content, onContentChange, onError, onSwitchToYaml }, ref) {
  const [modules, setModules] = useState<ConfigModule[]>([])
  const [activeId, setActiveId] = useState('general')
  const [moduleYAML, setModuleYAML] = useState('')
  const [dirty, setDirty] = useState(false)
  const [applying, setApplying] = useState(false)
  const dirtyRef = useRef(false)

  useEffect(() => {
    api
      .configModules(content)
      .then((rows) => {
        setModules(Array.isArray(rows) ? rows : [])
      })
      .catch((e: Error) => onError(e.message))
  }, [content, onError])

  useEffect(() => {
    const mod = modules.find((m) => m.id === activeId)
    setModuleYAML(mod?.yaml ?? '')
    setDirty(false)
    dirtyRef.current = false
  }, [modules, activeId])

  const doApply = async (targetId: string): Promise<void> => {
    const mod = modules.find((m) => m.id === targetId)
    if (!mod) return
    setApplying(true)
    onError('')
    try {
      const res = await api.mergeConfigModule(content, targetId, moduleYAML)
      onContentChange(res.content)
      setDirty(false)
      dirtyRef.current = false
    } catch (e) {
      throw e
    } finally {
      setApplying(false)
    }
  }

  const applyModule = async () => {
    try {
      await doApply(activeId)
    } catch (e) {
      onError(e instanceof Error ? e.message : String(e))
    }
  }

  const autoApplyIfDirty = async (): Promise<void> => {
    if (!dirtyRef.current) return
    await doApply(activeId)
  }

  useImperativeHandle(ref, () => ({ autoApplyIfDirty }), [activeId, moduleYAML, content])

  const handleTabClick = async (newId: string) => {
    if (newId === activeId) return
    if (applying) return
    if (dirtyRef.current) {
      try {
        await doApply(activeId)
      } catch (e) {
        onError(e instanceof Error ? e.message : String(e))
        return // stay on current tab on error
      }
    }
    setActiveId(newId)
  }

  const active = modules.find((m) => m.id === activeId)

  return (
    <div className="config-modules">
      <aside className="config-module-nav">
        {modules.map((m) => (
          <button
            key={m.id}
            type="button"
            className={`config-module-tab ${activeId === m.id ? 'active' : ''}`}
            onClick={() => handleTabClick(m.id)}
            disabled={applying}
          >
            <span>{m.label}</span>
            {!m.yaml.trim() && <em className="config-module-empty">空</em>}
          </button>
        ))}
      </aside>
      <div className="config-module-editor">
        <div className="config-module-head">
          <div>
            <h3>{active?.label || '模块'}</h3>
            <p className="config-module-keys">
              {dirty
                ? '模块有未应用的变更，切换 Tab 时将自动应用。'
                : '切换 Tab 时自动应用变更；也可手动点击应用。'}
            </p>
          </div>
          <button
            type="button"
            className={`btn ${dirty ? 'btn-primary' : ''}`}
            disabled={!dirty || applying}
            onClick={applyModule}
          >
            {applying ? '应用中…' : dirty ? '应用模块' : '已保存'}
          </button>
        </div>
        <ConfigModuleForm
          moduleId={activeId}
          moduleYAML={moduleYAML}
          onYAMLChange={(yaml) => {
            setModuleYAML(yaml)
            setDirty(true)
            dirtyRef.current = true
          }}
          onSwitchToYaml={onSwitchToYaml}
        />
        <p className="config-module-hint">
          修改表单后切换左侧模块 Tab、或点击「保存到 YAML」/「发布」时将自动应用变更。
        </p>
      </div>
    </div>
  )
})
