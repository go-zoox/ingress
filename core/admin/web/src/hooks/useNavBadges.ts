import { useEffect, useState } from 'react'
import { api } from '../api/client'
import type { NavBadgeKey } from '../layout/navConfig'

export type NavBadges = Record<NavBadgeKey, number>

const empty: NavBadges = { events: 0, healths: 0, tls: 0, waf: 0 }

export function useNavBadges() {
  const [badges, setBadges] = useState<NavBadges>(empty)

  useEffect(() => {
    const load = () => {
      Promise.all([
        api.eventsSummary('open').catch(() => null),
        api.healthCheck().catch(() => ({ checks: [], summary: { total: 0, up: 0, down: 0, unknown: 0 } })),
        api.tlsCerts().catch(() => [] as Awaited<ReturnType<typeof api.tlsCerts>>),
        api
          .wafEvents({ action: 'block', status: 'open', limit: 1 })
          .catch(() => []),
      ]).then(([eventsSummary, health, certs, wafEvents]) => {
        const down = health.summary?.down ?? 0
        const critical = certs.filter((c) => c.days_remaining < 7).length
        const warn = certs.filter((c) => c.days_remaining >= 7 && c.days_remaining < 30).length
        const wafList = Array.isArray(wafEvents) ? wafEvents : []
        setBadges({
          events: eventsSummary?.total ?? 0,
          healths: down,
          tls: critical + warn,
          waf: eventsSummary?.waf_block ?? wafList.length,
        })
      })
    }
    load()
    const timer = window.setInterval(load, 60_000)
    return () => window.clearInterval(timer)
  }, [])

  return badges
}
