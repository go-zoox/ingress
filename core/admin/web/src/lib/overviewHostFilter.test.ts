import { describe, expect, it } from 'vitest'
import { filterHostTimeline, mergeOverviewHostNames, overviewHostFilterSummary } from './overviewHostFilter'

describe('mergeOverviewHostNames', () => {
  it('merges and dedupes host sources', () => {
    expect(
      mergeOverviewHostNames({
        host_traffic: [{ name: 'b.example.com', pv: 1, uv: 1 }],
        top_hosts: [{ name: 'a.example.com', count: 2 }],
        host_timeline: [{ name: 'c.example.com', total: 1, points: [] }],
      }),
    ).toEqual(['a.example.com', 'b.example.com', 'c.example.com'])
  })
})

describe('filterHostTimeline', () => {
  const series = [
    { name: 'a.example.com', total: 10, points: [{ label: '12:00', count: 10 }] },
    { name: 'b.example.com', total: 5, points: [{ label: '12:00', count: 5 }] },
    { name: 'Other', total: 2, points: [{ label: '12:00', count: 2 }] },
  ]

  it('returns all when selection empty', () => {
    expect(filterHostTimeline(series, [])).toEqual(series)
  })

  it('filters to selected hosts and drops Other', () => {
    expect(filterHostTimeline(series, ['a.example.com'])).toEqual([series[0]])
  })
})

describe('overviewHostFilterSummary', () => {
  it('shows 全部 when empty or all picked', () => {
    expect(overviewHostFilterSummary([], 3)).toBe('全部')
    expect(overviewHostFilterSummary(['a', 'b', 'c'], 3)).toBe('全部')
  })
})
