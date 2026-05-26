import { memo } from 'react'
import { Link } from 'react-router-dom'
import { Circle, Radio, RefreshCw } from 'lucide-react'
import { ConfigGovernanceBadges } from './ConfigGovernanceBadges'

type Props = {
  reloadReady: boolean
  configHash: string
  runtimeHash?: string
  latestHash: string
  runtimeDrift?: boolean
  revisionDrift?: boolean
  sseConnected: boolean
}

export const SidebarGlobalStatus = memo(function SidebarGlobalStatus({
  reloadReady,
  configHash,
  runtimeHash,
  latestHash,
  runtimeDrift,
  revisionDrift,
  sseConnected,
}: Props) {
  return (
    <div className="sidebar-global-status">
      <div className={`status-line ${reloadReady ? 'ok' : 'warn'}`}>
        <RefreshCw size={12} aria-hidden />
        <span>{reloadReady ? '热加载就绪' : '热加载不可用'}</span>
      </div>
      <div className="status-line">
        <span className="status-label">配置 hash</span>
        <code>{configHash ? configHash.slice(0, 8) : '—'}</code>
        <ConfigGovernanceBadges
          fileHash={configHash}
          runtimeHash={runtimeHash}
          latestRevisionHash={latestHash}
          runtimeDrift={runtimeDrift}
          revisionDrift={revisionDrift}
        />
      </div>
      <div className={`status-line ${sseConnected ? 'ok' : ''}`}>
        <Radio size={12} aria-hidden />
        <span>{sseConnected ? 'SSE 已连接' : 'SSE 未连接'}</span>
      </div>
      {!reloadReady ? (
        <p className="status-hint">
          需先 <code>ingress run</code> 且 pid 与 admin 一致。见{' '}
          <Link to="/settings">设置</Link>。
        </p>
      ) : null}
      <div className="status-line muted">
        <Circle size={8} fill="currentColor" aria-hidden />
        <span>单机部署 · YAML 配置</span>
      </div>
    </div>
  )
})
