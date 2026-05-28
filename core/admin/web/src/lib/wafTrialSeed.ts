import type { WAFEvent } from '../api/client'

export type WafTrialSeed = Pick<WAFEvent, 'id' | 'host' | 'path' | 'client_ip' | 'rule'>

export function wafTrialFormFromSeed(seed?: Partial<WafTrialSeed> | null) {
  const ruleLower = (seed?.rule || '').toLowerCase()
  return {
    host: seed?.host || 'api.example.com',
    path: seed?.path || '/search?q=test',
    method: 'GET',
    clientIP: seed?.client_ip || '',
    userAgent:
      ruleLower.includes('scanner') || ruleLower.includes('ua') ? 'scanner/1.0' : '',
    eventId: seed?.id && seed.id > 0 ? seed.id : null,
    expectedRule: seed?.rule || '',
  }
}
