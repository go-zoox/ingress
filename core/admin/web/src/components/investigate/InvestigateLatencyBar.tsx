type Props = {
  durationMs: number
  upstreamDurationMs: number
}

export function InvestigateLatencyBar({ durationMs, upstreamDurationMs }: Props) {
  if (durationMs <= 0) return null
  const upstream = upstreamDurationMs > 0 ? upstreamDurationMs : 0
  const gateway = upstream > 0 ? Math.max(0, durationMs-upstream) : durationMs
  const total = durationMs
  const upPct = upstream > 0 ? Math.min(100, (upstream / total) * 100) : 0
  const gwPct = upstream > 0 ? 100 - upPct : 100

  if (upstream <= 0) {
    return (
      <p className="chart-hint investigate-latency-hint">日志无 upstream_response_time，仅显示总耗时 {formatMs(durationMs)}</p>
    )
  }

  return (
    <div className="investigate-latency">
      <div className="investigate-latency-labels">
        <span>总耗时 {formatMs(total)}</span>
        <span>上游 {formatMs(upstream)} · 网关内 {formatMs(gateway)}</span>
      </div>
      <div className="investigate-latency-track" role="img" aria-label="延迟分布">
        <div className="investigate-latency-upstream" style={{ width: `${upPct}%` }} title={`上游 ${formatMs(upstream)}`} />
        <div className="investigate-latency-gateway" style={{ width: `${gwPct}%` }} title={`网关内 ${formatMs(gateway)}`} />
      </div>
    </div>
  )
}

function formatMs(ms: number) {
  if (ms >= 1000) return `${(ms / 1000).toFixed(2)}s`
  return `${Math.round(ms)}ms`
}
