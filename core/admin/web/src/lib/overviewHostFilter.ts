import type { OverviewMetrics } from '../api/client'

export type HostTimelineSeries = NonNullable<OverviewMetrics['host_timeline']>[number]

/** Unique host names for the overview domain filter (sorted). */
export function mergeOverviewHostNames(metrics: Pick<OverviewMetrics, 'host_traffic' | 'top_hosts' | 'host_timeline'>): string[] {
  const set = new Set<string>()
  for (const row of metrics.host_traffic ?? []) {
    if (row.name) set.add(row.name)
  }
  for (const row of metrics.top_hosts ?? []) {
    if (row.name) set.add(row.name)
  }
  for (const row of metrics.host_timeline ?? []) {
    if (row.name && row.name !== 'Other') set.add(row.name)
  }
  return [...set].sort((a, b) => a.localeCompare(b))
}

/** Empty selection = show all series (including Other when present). */
export function filterHostTimeline(series: HostTimelineSeries[], selectedHosts: string[]): HostTimelineSeries[] {
  if (selectedHosts.length === 0) return series
  const pick = new Set(selectedHosts)
  return series.filter((row) => row.name !== 'Other' && pick.has(row.name))
}

export function overviewHostFilterSummary(selectedHosts: string[], totalHosts: number): string {
  if (selectedHosts.length === 0) return '全部'
  if (selectedHosts.length === 1) return selectedHosts[0]
  if (selectedHosts.length === totalHosts) return '全部'
  return `${selectedHosts.length} 个域名`
}
