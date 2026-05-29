import { useEffect, useState } from 'react'
import { api } from '../api/client'
import type { NavBadgeKey } from '../layout/navConfig'
import { countOpenAdminEvents, OPEN_EVENTS_LIST_LIMIT } from '../lib/adminEvents'

export type NavBadges = Record<NavBadgeKey, number>

const empty: NavBadges = { events: 0, healths: 0, tls: 0, waf: 0 }

export function useNavBadges() {
  const [badges, setBadges] = useState<NavBadges>(empty)

  useEffect(() => {
    const load = () => {
      Promise.all([
        api.healthCheck().catch(() => ({ checks: [], summary: { total: 0, up: 0, down: 0, unknown: 0 } })),
        api.tlsCerts().catch(() => [] as Awaited<ReturnType<typeof api.tlsCerts>>),
        api.parseIssues('open', OPEN_EVENTS_LIST_LIMIT).catch(() => []),
        api
          .wafEvents({ action: 'block', status: 'open', limit: OPEN_EVENTS_LIST_LIMIT })
          .catch(() => []),
      ]).then(([health, certs, parseIssues, wafEvents]) => {
        const down = health.summary?.down ?? 0
        const critical = certs.filter((c) => c.days_remaining < 7).length
        const warn = certs.filter((c) => c.days_remaining >= 7 && c.days_remaining < 30).length
        const wafList = Array.isArray(wafEvents) ? wafEvents : []
        const parseList = Array.isArray(parseIssues) ? parseIssues : []
        const certList = Array.isArray(certs) ? certs : []
        setBadges({
          events: countOpenAdminEvents({
            wafEvents: wafList,
            parseIssues: parseList,
            healthChecks: health.checks || [],
            certs: certList,
          }),
          healths: down,
          tls: critical + warn,
          waf: wafList.length,
        })
      })
    }
    load()
    const timer = window.setInterval(load, 60_000)
    return () => window.clearInterval(timer)
  }, [])

  return badges
}
