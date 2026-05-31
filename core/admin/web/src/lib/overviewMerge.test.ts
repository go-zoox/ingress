import { describe, expect, it } from 'vitest'
import { mergeOverviewPatch, overviewPatchWindowMismatch } from './overviewMerge'
import type { OverviewSnapshot } from '../api/client'

const baseSnapshot: OverviewSnapshot = {
  window: '15m',
  status: {} as OverviewSnapshot['status'],
  metrics: {
    window: '15m',
    source: 'access_log',
    total: 100,
    rpm: 6.7,
    error_rate: 1.2,
    p50_ms: 10,
    p95_ms: 50,
    cache_hit_rate: 20,
    waf_blocks: 0,
    status_counts: { '2xx': 100, '3xx': 0, '4xx': 0, '5xx': 0 },
    timeline: [],
    top_hosts: [],
    top_hosts_error: [],
    top_paths: [],
    latency_histogram: [],
    latency_slo: [],
    delta: { has_previous: false, total_pct: 0, rpm_pct: 0, error_rate_delta: 0, cache_hit_delta: 0, waf_blocks_delta: 0, p95_delta_ms: 0 },
  },
  system: {
    window: '15m',
    cpu_pct: 1,
    memory_mb: 100,
    goroutines: 10,
    num_cpu: 4,
    timeline: [],
  },
  certs: [],
  health_checks: [],
  health_summary: { total: 0, up: 0, down: 0, unknown: 0 },
  waf_blocks: [],
  parse_issues: [],
  revisions: [],
}

describe('mergeOverviewPatch', () => {
  it('returns null without a REST base snapshot', () => {
    expect(
      mergeOverviewPatch(undefined, {
        seq: 1,
        window: '15m',
        metrics: { total: 200 },
      }),
    ).toBeNull()
  })

  it('ignores scalar-only metrics patch without timeline', () => {
    const merged = mergeOverviewPatch(baseSnapshot, {
      seq: 2,
      window: '15m',
      metrics: { total: 200 },
    })
    expect(merged?.metrics.total).toBe(100)
    expect(merged?.metrics.rpm).toBe(6.7)
  })

  it('merges metrics when timeline is included', () => {
    const merged = mergeOverviewPatch(baseSnapshot, {
      seq: 2,
      window: '15m',
      metrics: { total: 200, timeline: baseSnapshot.metrics.timeline },
    })
    expect(merged?.metrics.total).toBe(200)
    expect(merged?.metrics.rpm).toBe(6.7)
  })

  it('rejects patch when window differs from base snapshot', () => {
    expect(
      mergeOverviewPatch(baseSnapshot, {
        seq: 4,
        window: '24h',
        metrics: { total: 999 },
      }),
    ).toBeNull()
  })

  it('rejects timeline patch from a different window', () => {
    const timeline24 = Array.from({ length: 24 }, (_, i) => ({
      label: String(i),
      count: i + 1,
      '2xx': i + 1,
      '3xx': 0,
      '4xx': 0,
      '5xx': 0,
      error_rate: 0,
      cache_hit_rate: 0,
      waf_blocks: 0,
    }))
    const base24: OverviewSnapshot = {
      ...baseSnapshot,
      window: '24h',
      metrics: {
        ...baseSnapshot.metrics,
        window: '24h',
        timeline: timeline24,
      },
    }
    const merged = mergeOverviewPatch(base24, {
      seq: 3,
      metrics: {
        window: '24h',
        timeline: Array.from({ length: 12 }, () => ({
          label: 'x',
          count: 0,
          '2xx': 0,
          '3xx': 0,
          '4xx': 0,
          '5xx': 0,
          error_rate: 0,
          cache_hit_rate: 0,
          waf_blocks: 0,
        })),
      },
    })
    expect(merged?.metrics.timeline).toBe(timeline24)
  })
})

describe('overviewPatchWindowMismatch', () => {
  it('detects stale window patches', () => {
    expect(
      overviewPatchWindowMismatch({ seq: 1, window: '5m' }, '15m', '15m'),
    ).toBe(true)
  })
})
