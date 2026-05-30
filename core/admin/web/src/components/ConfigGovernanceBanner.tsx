import { Link } from 'react-router-dom'
import { AlertTriangle } from 'lucide-react'

type Props = {
  runtimeDrift?: boolean
  revisionDrift?: boolean
  reloadReady?: boolean
  fileHash?: string
  runtimeHash?: string
  latestRevisionHash?: string
  onReload?: () => void
}

export function ConfigGovernanceBanner({
  runtimeDrift,
  revisionDrift,
  reloadReady,
  fileHash,
  runtimeHash,
  latestRevisionHash,
  onReload,
}: Props) {
  if (!runtimeDrift && !revisionDrift) return null

  return (
    <div className="config-governance-banner">
      <AlertTriangle size={16} aria-hidden />
      <div className="config-governance-banner-body">
        {runtimeDrift ? (
          <p>
            <strong>配置漂移：</strong>磁盘上的 ingress.yaml（<code>{fileHash?.slice(0, 8)}</code>）与当前运行配置（
            <code>{runtimeHash?.slice(0, 8) || '—'}</code>）不一致。请发布并 reload，或在外部改回文件后 reload。
            {reloadReady && onReload ? (
              <>
                {' '}
                <button type="button" className="btn btn-sm btn-primary" onClick={onReload}>
                  立即 reload
                </button>
              </>
            ) : null}
          </p>
        ) : null}
        {revisionDrift ? (
          <p>
            <strong>版本未对齐：</strong>磁盘 hash 与最新发布记录（<code>{latestRevisionHash?.slice(0, 8)}</code>）不同。
            可能为「仅保存未发布」或外部编辑；可在下方版本时间线对比或继续发布。
          </p>
        ) : null}
        <p className="config-governance-banner-hint">
          详见 <Link to="/config">配置中心</Link> 的版本与变更 Tab。
        </p>
      </div>
    </div>
  )
}
