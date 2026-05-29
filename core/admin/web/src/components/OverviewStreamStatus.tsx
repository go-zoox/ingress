import { memo } from 'react'
import { LayoutDashboard, Radio } from 'lucide-react'
import { useOverviewStreamOptional } from '../context/OverviewStreamContext'
import { metricsSourceLabel, overviewStreamLabel } from '../lib/metricsSource'

export const OverviewStreamStatus = memo(function OverviewStreamStatus() {
  const ctx = useOverviewStreamOptional()
  const stream = ctx?.stream
  if (!stream) return null

  const sseOk = stream.connected
  const sseWarn = !stream.connected && (stream.reconnecting || stream.fallbackPolling)

  return (
    <div className="settings-menu-stream">
      <div className={`status-line${sseOk ? ' ok' : sseWarn ? ' warn' : ''}`}>
        <Radio size={12} aria-hidden />
        <span>{overviewStreamLabel(stream)}</span>
      </div>
      <div className="status-line muted">
        <LayoutDashboard size={12} aria-hidden />
        <span>
          数据源 {metricsSourceLabel(stream.metricsSource)}
          {stream.windowStale ? ' · 历史数据' : ''}
        </span>
      </div>
    </div>
  )
})
