import { type ReactNode } from 'react'
import { Cpu, HardDrive, Zap } from 'lucide-react'
import type { OverviewMetrics, SystemMetrics } from '../api/client'
import { type OverviewRange, formatOverviewRangeLabel, snapshotMatchesRange } from '../lib/overviewRange'
import { KpiSparkline } from './KpiSparkline'

type Props = {
  system: SystemMetrics | null
  chartSystem?: SystemMetrics | null
  metrics: OverviewMetrics | null
  chartMetrics?: OverviewMetrics | null
  overviewRange: OverviewRange
  loading?: boolean
  refreshing?: boolean
}

export function SystemResourceStrip({
  system,
  chartSystem,
  metrics,
  chartMetrics,
  overviewRange,
  loading,
  refreshing,
}: Props) {
  const liveSystem = snapshotMatchesRange(system, overviewRange) ? system : null
  const liveMetrics = snapshotMatchesRange(metrics, overviewRange) ? metrics : null
  const sparkSystem = liveSystem ?? chartSystem
  const sparkMetrics = liveMetrics ?? chartMetrics
  const displayMetrics = liveMetrics ?? chartMetrics
  const cpuSpark = sparkSystem?.timeline?.map((p) => p.cpu_pct) ?? []
  const memSpark = sparkSystem?.timeline?.map((p) => p.memory_mb) ?? []
  const throughputSpark = sparkMetrics?.timeline?.map((b) => b.count) ?? []

  const cpuTone =
    (system?.cpu_pct ?? 0) > 80 ? 'var(--danger)' : (system?.cpu_pct ?? 0) > 50 ? 'var(--warn)' : 'var(--accent)'
  const rpm = displayMetrics?.rpm ?? 0
  const windowLabel = formatOverviewRangeLabel(overviewRange)

  return (
    <div className={refreshing ? 'overview-resource-strip is-refreshing' : 'overview-resource-strip'}>
      <div className="overview-resource-head">
        <h3>进程资源</h3>
        <span className="overview-resource-hint">本进程 · 每 10s 采样</span>
      </div>
      <div className="overview-resource-grid">
        <ResourceCard
          icon={<Cpu size={16} aria-hidden />}
          label="CPU"
          value={loading && !system ? '—' : `${(system?.cpu_pct ?? 0).toFixed(1)}%`}
          sub={system ? `${system.num_cpu} 核 · ${system.goroutines} goroutines` : '采样中…'}
          spark={cpuSpark}
          sparkTone={cpuTone}
        />
        <ResourceCard
          icon={<HardDrive size={16} aria-hidden />}
          label="内存"
          value={loading && !system ? '—' : `${(system?.memory_mb ?? 0).toFixed(1)} MB`}
          sub="RSS"
          spark={memSpark}
          sparkTone="var(--ok)"
        />
        <ResourceCard
          icon={<Zap size={16} aria-hidden />}
          label="L7 吞吐"
          value={displayMetrics ? `${rpm.toFixed(1)} 次/分` : '—'}
          sub={displayMetrics ? `${displayMetrics.total} 请求 · ${windowLabel}` : '等待 access.log'}
          spark={throughputSpark}
          sparkTone="var(--accent)"
        />
      </div>
    </div>
  )
}

function ResourceCard({
  icon,
  label,
  value,
  sub,
  spark,
  sparkTone,
}: {
  icon: ReactNode
  label: string
  value: string
  sub: string
  spark: number[]
  sparkTone?: string
}) {
  return (
    <div className="overview-resource-card">
      <div className="overview-resource-card-head">
        {icon}
        <span>{label}</span>
      </div>
      <div className="overview-resource-value">{value}</div>
      <div className="overview-resource-sub">{sub}</div>
      <KpiSparkline values={spark} tone={sparkTone} className="kpi-sparkline overview-resource-sparkline" />
    </div>
  )
}
