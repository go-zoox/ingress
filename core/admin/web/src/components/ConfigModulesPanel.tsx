import { forwardRef, useCallback, useEffect, useImperativeHandle, useRef, useState } from 'react'
import { api, type ConfigModule } from '../api/client'
import { ConfigModuleForm } from './ConfigModuleForm'
import { parseModuleDoc } from '../lib/ingressModuleForms'
import { globalMaintenanceFromDoc, validateGlobalMaintenanceForm } from '../lib/maintenance'

const AUTO_APPLY_DEBOUNCE_MS = 400

export interface ConfigModulesPanelHandle {
  /** Merges the active module into draft YAML when dirty; returns merged content or null. */
  autoApplyIfDirty: () => Promise<string | null>
}

export const ConfigModulesPanel = forwardRef<ConfigModulesPanelHandle, {
  content: string
  onContentChange: (next: string) => void
  onError: (message: string) => void
  onSwitchToYaml?: () => void
  initialModuleId?: string
}>(function ConfigModulesPanel({ content, onContentChange, onError, onSwitchToYaml, initialModuleId }, ref) {
  const [modules, setModules] = useState<ConfigModule[]>([])
  const [activeId, setActiveId] = useState('general')
  const [moduleYAML, setModuleYAML] = useState('')
  const [dirty, setDirty] = useState(false)
  const [applying, setApplying] = useState(false)
  const dirtyRef = useRef(false)
  const moduleYAMLRef = useRef(moduleYAML)
  const contentRef = useRef(content)
  const activeIdRef = useRef(activeId)
  const applyTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const applyingRef = useRef(false)

  useEffect(() => {
    moduleYAMLRef.current = moduleYAML
  }, [moduleYAML])

  useEffect(() => {
    contentRef.current = content
  }, [content])

  useEffect(() => {
    activeIdRef.current = activeId
  }, [activeId])

  useEffect(() => {
    return () => {
      if (applyTimerRef.current) clearTimeout(applyTimerRef.current)
    }
  }, [])

  useEffect(() => {
    if (applyTimerRef.current) clearTimeout(applyTimerRef.current)
    api
      .configModules(content)
      .then((rows) => {
        const list = Array.isArray(rows) ? rows : []
        // Hide empty catch-all module; unknown keys still appear when non-empty.
        const filtered = list.filter((m) => m.id !== 'other' || m.yaml.trim() !== '')
        setModules(filtered)
        if (initialModuleId && filtered.some((m) => m.id === initialModuleId)) {
          setActiveId(initialModuleId)
        }
      })
      .catch((e: Error) => onError(e.message))
  }, [content, onError, initialModuleId])

  useEffect(() => {
    const mod = modules.find((m) => m.id === activeId)
    setModuleYAML(mod?.yaml ?? '')
    setDirty(false)
    dirtyRef.current = false
  }, [modules, activeId])

  useEffect(() => {
    if (!initialModuleId || !modules.some((m) => m.id === initialModuleId)) return
    setActiveId(initialModuleId)
  }, [initialModuleId, modules])

  const doApply = useCallback(async (targetId: string, yaml: string, baseContent: string): Promise<string | null> => {
    const mod = modules.find((m) => m.id === targetId)
    if (!mod) return null
    setApplying(true)
    applyingRef.current = true
    onError('')
    try {
      const res = await api.mergeConfigModule(baseContent, targetId, yaml)
      onContentChange(res.content)
      setDirty(false)
      dirtyRef.current = false
      return res.content
    } finally {
      setApplying(false)
      applyingRef.current = false
    }
  }, [modules, onContentChange, onError])

  const runAutoApply = useCallback(async (): Promise<string | null> => {
    if (!dirtyRef.current || applyingRef.current) return null
    if (activeIdRef.current === 'maintenance') {
      const err = validateGlobalMaintenanceForm(globalMaintenanceFromDoc(parseModuleDoc(moduleYAMLRef.current)))
      if (err) {
        onError(err)
        return null
      }
    }
    try {
      return await doApply(activeIdRef.current, moduleYAMLRef.current, contentRef.current)
    } catch (e) {
      onError(e instanceof Error ? e.message : String(e))
      return null
    }
  }, [doApply, onError])

  const cancelScheduledApply = useCallback(() => {
    if (applyTimerRef.current) {
      clearTimeout(applyTimerRef.current)
      applyTimerRef.current = null
    }
  }, [])

  const scheduleAutoApply = useCallback(() => {
    cancelScheduledApply()
    applyTimerRef.current = setTimeout(() => {
      applyTimerRef.current = null
      void runAutoApply()
    }, AUTO_APPLY_DEBOUNCE_MS)
  }, [cancelScheduledApply, runAutoApply])

  const flushAutoApply = useCallback(async (): Promise<string | null> => {
    cancelScheduledApply()
    return runAutoApply()
  }, [cancelScheduledApply, runAutoApply])

  useImperativeHandle(ref, () => ({
    autoApplyIfDirty: flushAutoApply,
  }), [flushAutoApply])

  const handleTabClick = async (newId: string) => {
    if (newId === activeId) return
    if (applyingRef.current) return
    cancelScheduledApply()
    if (dirtyRef.current) {
      const merged = await flushAutoApply()
      if (dirtyRef.current) return // apply failed; stay on tab
      if (merged != null) {
        contentRef.current = merged
      }
    }
    setActiveId(newId)
  }

  const active = modules.find((m) => m.id === activeId)

  const syncLabel = applying
    ? '正在同步到草稿…'
    : dirty
      ? '等待同步…'
      : '已同步到草稿'

  return (
    <div className="config-modules">
      <aside className="config-module-nav">
        {modules.map((m) => (
          <button
            key={m.id}
            type="button"
            className={`config-module-tab ${activeId === m.id ? 'active' : ''}`}
            onClick={() => void handleTabClick(m.id)}
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
              修改后会自动合并到草稿 YAML，可用「校验」检查后再「保存与发布」。
            </p>
          </div>
          <span
            className={`config-module-sync ${dirty ? 'config-module-sync--pending' : ''} ${applying ? 'config-module-sync--busy' : ''}`}
            role="status"
            aria-live="polite"
          >
            {syncLabel}
          </span>
        </div>
        <ConfigModuleForm
          moduleId={activeId}
          moduleYAML={moduleYAML}
          onYAMLChange={(yaml) => {
            setModuleYAML(yaml)
            setDirty(true)
            dirtyRef.current = true
            scheduleAutoApply()
          }}
          onSwitchToYaml={onSwitchToYaml}
        />
      </div>
    </div>
  )
})
