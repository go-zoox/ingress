import { useCallback, useEffect, useState } from 'react'
import { api } from '../api/client'
import type {
  AccessLogParseIssue,
  HealthCheckResult,
  OverviewMetrics,
  TLSCert,
  WAFEvent,
} from '../api/client'
import { loadPreferences } from '../lib/preferences'
import { parseOverviewRange, rangeToQueryParams } from '../lib/overviewRange'

type Options = {
  metricsWindow?: string
  parseIssueLimit?: number
  wafLimit?: number
  autoRefreshMs?: number
}

const DEFAULT_AUTO_REFRESH_MS = 5000

export function useAttentionData(options: Options = {}) {
  const prefs = loadPreferences()
  const overviewRange = parseOverviewRange(
    prefs.overviewRange,
    options.metricsWindow ?? prefs.metricsWindow,
  )
  const metricsQuery = rangeToQueryParams(overviewRange)
  const parseIssueLimit = options.parseIssueLimit ?? 10
  const wafLimit = options.wafLimit ?? 8
  const autoRefreshMs = options.autoRefreshMs ?? DEFAULT_AUTO_REFRESH_MS

  const [metrics, setMetrics] = useState<OverviewMetrics | null>(null)
  const [certs, setCerts] = useState<TLSCert[]>([])
  const [healthChecks, setHealthChecks] = useState<HealthCheckResult[]>([])
  const [wafBlocks, setWafBlocks] = useState<WAFEvent[]>([])
  const [parseIssues, setParseIssues] = useState<AccessLogParseIssue[]>([])
  const [loading, setLoading] = useState(true)

  const loadParseIssues = useCallback(() => {
    return api
      .parseIssues('open', parseIssueLimit)
      .then((d) => setParseIssues(Array.isArray(d) ? d : []))
      .catch(() => setParseIssues([]))
  }, [parseIssueLimit])

  const refresh = useCallback(() => {
    setLoading(true)
    return Promise.all([
      api.overviewMetrics(metricsQuery),
      api.tlsCerts(),
      api.healthCheck(),
      api.wafEvents({ action: 'block', status: 'open', limit: Math.max(wafLimit, 30) }),
      api.parseIssues('open', parseIssueLimit),
    ])
      .then(([overview, certList, health, waf, issues]) => {
        setMetrics(overview)
        setCerts(Array.isArray(certList) ? certList : [])
        setHealthChecks(health.checks || [])
        const blocks = (Array.isArray(waf) ? waf : []).filter((e) => e.action === 'block')
        setWafBlocks(blocks.slice(0, wafLimit))
        setParseIssues(Array.isArray(issues) ? issues : [])
        setLoading(false)
      })
      .catch(() => {
        setLoading(false)
      })
  }, [metricsQuery, parseIssueLimit, wafLimit])

  const handleParseIssueStatus = useCallback(
    async (id: number, status: 'ignored' | 'resolved') => {
      await api.updateParseIssueStatus(id, status)
      await loadParseIssues()
      const overview = await api.overviewMetrics(metricsQuery).catch(() => null)
      if (overview) setMetrics(overview)
    },
    [loadParseIssues, metricsQuery],
  )

  const handleWafEventStatus = useCallback(
    async (id: number, status: 'ignored' | 'resolved') => {
      await api.updateWafEventStatus(id, status)
      const waf = await api
        .wafEvents({ action: 'block', status: 'open', limit: Math.max(wafLimit, 30) })
        .catch(() => [])
      const blocks = (Array.isArray(waf) ? waf : []).filter((e) => e.action === 'block')
      setWafBlocks(blocks.slice(0, wafLimit))
    },
    [wafLimit],
  )

  useEffect(() => {
    refresh()
    if (autoRefreshMs <= 0) return
    const timer = window.setInterval(refresh, autoRefreshMs)
    return () => window.clearInterval(timer)
  }, [refresh, autoRefreshMs])

  return {
    metrics,
    certs,
    healthChecks,
    wafBlocks,
    parseIssues,
    loading,
    refresh,
    handleParseIssueStatus,
    handleWafEventStatus,
    metricsQuery,
  }
}
