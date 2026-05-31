import { describe, expect, it } from 'vitest'
import {
  formatOverviewRangeLabel,
  isRangeLiveEligible,
  parseOverviewLiveEnabled,
  parseOverviewRange,
  rangeQueryKey,
  rangeToQueryParams,
  snapshotMatchesRange,
} from './overviewRange'

describe('overviewRange', () => {
  it('migrates legacy live to preset 5m', () => {
    expect(parseOverviewRange(undefined, 'live')).toEqual({ kind: 'preset', preset: '5m' })
    expect(parseOverviewRange(JSON.stringify({ kind: 'live' }))).toEqual({
      kind: 'preset',
      preset: '5m',
    })
  })

  it('defaults live enabled and migrates legacy live view', () => {
    expect(parseOverviewLiveEnabled(undefined)).toBe(true)
    expect(parseOverviewLiveEnabled(undefined, 'live')).toBe(true)
    expect(parseOverviewLiveEnabled(false)).toBe(false)
  })

  it('builds preset query params with from/to', () => {
    const q = rangeToQueryParams({ kind: 'preset', preset: '6h' })
    expect(q.from).toBeTruthy()
    expect(q.to).toBeTruthy()
    expect(new Date(q.to).getTime()).toBeGreaterThan(new Date(q.from).getTime())
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
  })

  it('detects live-eligible ranges', () => {
    const now = new Date('2026-05-31T12:00:00.000Z')
    expect(isRangeLiveEligible({ kind: 'preset', preset: '5m' }, now)).toBe(true)
    expect(
      isRangeLiveEligible(
        { kind: 'absolute', from: '2026-05-31T09:00:00.000Z', to: '2026-05-31T11:59:00.000Z' },
        now,
      ),
    ).toBe(true)
    expect(
      isRangeLiveEligible(
        { kind: 'absolute', from: '2026-05-30T00:00:00.000Z', to: '2026-05-30T23:59:59.000Z' },
        now,
      ),
    ).toBe(false)
  })
})
