import { describe, it, expect, vi, beforeEach } from 'vitest'

const { connect, disconnect } = vi.hoisted(() => ({
  connect: vi.fn(),
  disconnect: vi.fn(),
}))

vi.mock('echarts', () => ({
  connect,
  disconnect,
}))

import {
  OVERVIEW_TIMELINE_LINK_GROUP,
  overviewTimelineChartCount,
  registerOverviewTimelineChart,
  resetOverviewTimelineChartGroups,
  unregisterOverviewTimelineChart,
} from './overviewTimelineChartGroup'

function fakeChart() {
  return { group: '' } as { group: string }
}

describe('overviewTimelineChartGroup', () => {
  beforeEach(() => {
    resetOverviewTimelineChartGroups()
    connect.mockClear()
    disconnect.mockClear()
  })

  it('does not connect with fewer than two charts', () => {
    const chart = fakeChart()
    registerOverviewTimelineChart(OVERVIEW_TIMELINE_LINK_GROUP, chart as never)
    expect(chart.group).toBe(OVERVIEW_TIMELINE_LINK_GROUP)
    expect(connect).not.toHaveBeenCalled()
    expect(overviewTimelineChartCount(OVERVIEW_TIMELINE_LINK_GROUP)).toBe(1)
  })

  it('connects when a second chart registers', () => {
    registerOverviewTimelineChart(OVERVIEW_TIMELINE_LINK_GROUP, fakeChart() as never)
    registerOverviewTimelineChart(OVERVIEW_TIMELINE_LINK_GROUP, fakeChart() as never)
    expect(connect).toHaveBeenCalledWith(OVERVIEW_TIMELINE_LINK_GROUP)
  })

  it('disconnects when only one chart remains', () => {
    const a = fakeChart()
    const b = fakeChart()
    registerOverviewTimelineChart(OVERVIEW_TIMELINE_LINK_GROUP, a as never)
    registerOverviewTimelineChart(OVERVIEW_TIMELINE_LINK_GROUP, b as never)
    unregisterOverviewTimelineChart(OVERVIEW_TIMELINE_LINK_GROUP, a as never)
    expect(disconnect).toHaveBeenCalledWith(OVERVIEW_TIMELINE_LINK_GROUP)
    expect(overviewTimelineChartCount(OVERVIEW_TIMELINE_LINK_GROUP)).toBe(1)
  })
})
