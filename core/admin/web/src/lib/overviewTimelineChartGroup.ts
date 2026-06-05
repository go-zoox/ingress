import * as echarts from 'echarts'
import type { ECharts } from 'echarts'

/** ECharts connect group for overview time-series panels (shared crosshair / tooltip). */
export const OVERVIEW_TIMELINE_LINK_GROUP = 'ingress-overview-timeline'

const groups = new Map<string, Set<ECharts>>()

function syncConnect(group: string) {
  const set = groups.get(group)
  if (!set || set.size < 2) {
    echarts.disconnect(group)
    return
  }
  echarts.connect(group)
}

/** Register a chart instance for cross-panel timeline linking. */
export function registerOverviewTimelineChart(group: string, chart: ECharts) {
  chart.group = group
  let set = groups.get(group)
  if (!set) {
    set = new Set()
    groups.set(group, set)
  }
  set.add(chart)
  syncConnect(group)
}

/** Remove a chart from its link group (call before dispose). */
export function unregisterOverviewTimelineChart(group: string, chart: ECharts) {
  const set = groups.get(group)
  if (!set) return
  set.delete(chart)
  if (set.size === 0) {
    groups.delete(group)
  }
  syncConnect(group)
}

/** @internal Test helper */
export function overviewTimelineChartCount(group: string): number {
  return groups.get(group)?.size ?? 0
}

/** @internal Test helper */
export function resetOverviewTimelineChartGroups() {
  for (const group of groups.keys()) {
    echarts.disconnect(group)
  }
  groups.clear()
}
