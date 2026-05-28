import { useEffect, useState } from 'react'
import { api } from '../api/client'
import type { NavBadgeKey } from '../layout/navConfig'
import { countAttentionItems } from './useAttentionData'

export type NavBadges = Record<NavBadgeKey, number>

const empty: NavBadges = { attention: 0, healths: 0, tls: 0, waf: 0 }

export function useNavBadges() {
  const [badges, setBadges] = useState<NavBadges>(empty)

  useEffect(() => {
    const load = () => {
      Promise.all([
        api.healthCheck().catch(() => ({ checks: [], summary: { total: 0, up: 0, down: 0, unknown: 0 } })),
        api.tlsCerts().catch(() => [] as Awaited<ReturnType<typeof api.tlsCerts>>),
        api.parseIssues('open', 50).catch(() => []),
        api.wafEvents({ action: 'block', limit: 30 }).catch(() => []),
      ]).then(([health, certs, parseIssues, wafEvents]) => {
        const down = health.summary?.down ?? 0
        const critical = certs.filter((c) => c.days_remaining < 7).length
        const warn = certs.filter((c) => c.days_remaining >= 7 && c.days_remaining < 30).length
        const wafBlocks = Array.isArray(wafEvents) ? wafEvents.length : 0
        const parseOpen = Array.isArray(parseIssues) ? parseIssues.length : 0
        setBadges({
          attention: countAttentionItems({
            healthDown: down,
            parseOpen,
            tlsCritical: critical,
            tlsWarn: warn,
            wafBlocks,
          }),
          healths: down,
          tls: critical + warn,
          waf: wafBlocks,
        })
      })
    }
    load()
    const timer = window.setInterval(load, 60_000)
    return () => window.clearInterval(timer)
  }, [])

  return badges
}
