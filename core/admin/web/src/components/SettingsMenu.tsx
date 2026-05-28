import { memo, useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { Settings } from 'lucide-react'
import { SidebarGlobalStatus } from './SidebarGlobalStatus'
import { SettingsAboutSection } from './SettingsAboutSection'

type Props = {
  configPath: string
  version: string
  reloadReady: boolean
  configHash: string
  runtimeHash: string
  latestHash: string
  runtimeDrift: boolean
  revisionDrift: boolean
  sseConnected: boolean
}

export const SettingsMenu = memo(function SettingsMenu({
  configPath,
  version,
  reloadReady,
  configHash,
  runtimeHash,
  latestHash,
  runtimeDrift,
  revisionDrift,
  sseConnected,
}: Props) {
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const onDoc = (e: MouseEvent) => {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onDoc)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDoc)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  return (
    <div className="settings-menu" ref={rootRef}>
      <button
        type="button"
        className={`settings-menu-trigger app-chrome-trigger${open ? ' active' : ''}`}
        aria-expanded={open}
        aria-haspopup="dialog"
        aria-label="设置"
        title="设置"
        onClick={() => setOpen((v) => !v)}
      >
        <Settings size={18} aria-hidden />
      </button>
      {open ? (
        <div className="settings-menu-panel" role="dialog" aria-label="设置">
          <div className="settings-menu-head">
            <Settings size={16} aria-hidden />
            <span>设置</span>
          </div>
          <SidebarGlobalStatus
            reloadReady={reloadReady}
            configHash={configHash}
            runtimeHash={runtimeHash}
            latestHash={latestHash}
            runtimeDrift={runtimeDrift}
            revisionDrift={revisionDrift}
            sseConnected={sseConnected}
          />
          <div className="settings-menu-path">
            <span className="settings-menu-path-label">配置路径</span>
            <code title={configPath}>{configPath}</code>
          </div>
          <SettingsAboutSection version={version} />
          <div className="settings-menu-foot">
            <Link to="/settings" className="btn btn-ghost btn-sm" onClick={() => setOpen(false)}>
              全部设置
            </Link>
          </div>
        </div>
      ) : null}
    </div>
  )
})
