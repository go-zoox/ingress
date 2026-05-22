import { useEffect, useState } from 'react'
import { api, type ConfigModule } from '../api/client'
import { ConfigModuleForm } from './ConfigModuleForm'

export function ConfigModulesPanel({
  content,
  onContentChange,
  onError,
  onSwitchToYaml,
}: {
  content: string
  onContentChange: (next: string) => void
  onError: (message: string) => void
  onSwitchToYaml?: () => void
}) {
  const [modules, setModules] = useState<ConfigModule[]>([])
  const [activeId, setActiveId] = useState('general')
  const [moduleYAML, setModuleYAML] = useState('')
  const [dirty, setDirty] = useState(false)
  const [applying, setApplying] = useState(false)

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
  }, [modules, activeId])

  const applyModule = async () => {
    setApplying(true)
    onError('')
    try {
      const res = await api.mergeConfigModule(content, activeId, moduleYAML)
      onContentChange(res.content)
      setDirty(false)
    } catch (e) {
      onError(e instanceof Error ? e.message : String(e))
    } finally {
      setApplying(false)
    }
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
            onClick={() => setActiveId(m.id)}
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
            <p className="config-module-keys">按模块编辑 ingress 配置，修改后点击「应用模块」</p>
          </div>
          <button
              type="button"
              className="btn btn-primary"
              disabled={!dirty || applying}
              onClick={applyModule}
            >
              {applying ? '应用中…' : '应用模块'}
            </button>
        </div>
        <ConfigModuleForm
          moduleId={activeId}
          moduleYAML={moduleYAML}
          onYAMLChange={(yaml) => {
            setModuleYAML(yaml)
            setDirty(true)
          }}
          onSwitchToYaml={onSwitchToYaml}
        />
        <p className="config-module-hint">
            修改表单后点击「应用模块」，变更会合并进完整 ingress 草稿。
          </p>
      </div>
    </div>
  )
}
