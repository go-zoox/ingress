/** Composite health score (0–100) for the overview KPI strip. */
export function computeHealthScore(params: {
  errorRate: number
  p95Ms: number
  healthDown: number
  certCritical: number
  certWarn: number
  wafBlocks: number
}): number {
  let score = 100
  if (params.errorRate > 5) score -= 15
  if (params.errorRate > 10) score -= 15
  if (params.p95Ms > 500) score -= 10
  if (params.p95Ms > 2000) score -= 15
  if (params.healthDown > 0) {
    score -= Math.min(30, params.healthDown * 15)
  }
  if (params.certCritical > 0) score -= 25
  else if (params.certWarn > 0) score -= 10
  if (params.wafBlocks > 5) score -= 10
  else if (params.wafBlocks > 0) score -= 5
  return Math.max(0, Math.min(100, Math.round(score)))
}

export function healthScoreClass(score: number): 'ok' | 'warn' | 'danger' {
  if (score >= 80) return 'ok'
  if (score >= 50) return 'warn'
  return 'danger'
}
