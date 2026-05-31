import { describe, expect, it } from 'vitest'
import {
  formatOverviewRangeLabel,
  parseOverviewRange,
  rangeQueryKey,
  rangeToQueryParams,
  snapshotMatchesRange,
} from './overviewRange'

describe('overviewRange', () => {
  it('parses live from legacy metricsWindow', () => {
    expect(parseOverviewRange(undefined, 'live')).toEqual({ kind: 'live' })
  })

  it('builds preset query params', () => {
    expect(rangeToQueryParams({ kind: 'preset', preset: '6h' })).toEqual({ window: '6h' })
  })

  it('builds absolute query params', () => {
    const q = rangeToQueryParams({
      kind: 'absolute',
      from: '2026-05-31T09:00:00.000Z',
      to: '2026-05-31T17:00:00.000Z',
    })
    expect(q.from).toBe('2026-05-31T09:00:00.000Z')
    expect(q.to).toBe('2026-05-31T17:00:00.000Z')
  })

  it('matches absolute metrics snapshot', () => {
    const range = {
      kind: 'absolute' as const,
      from: '2026-05-31T09:00:00.000Z',
      to: '2026-05-31T17:00:00.000Z',
    }
    expect(
      snapshotMatchesRange(
        { window: 'custom', range_from: range.from, range_to: range.to },
        range,
      ),
    ).toBe(true)
  })

  it('formats preset label', () => {
    expect(formatOverviewRangeLabel({ kind: 'preset', preset: '1h' })).toBe('最近 1 小时')
  })

  it('uses stable cache keys', () => {
    expect(rangeQueryKey({ kind: 'preset', preset: '15m' })).toBe('15m')
    expect(rangeQueryKey({ kind: 'live' })).toBe('live')
  })
})
