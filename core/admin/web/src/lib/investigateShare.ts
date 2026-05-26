import { investigateLink } from './deepLinks'

export function investigateShareUrl(params: Parameters<typeof investigateLink>[0]) {
  const path = investigateLink(params)
  const base = import.meta.env.BASE_URL?.replace(/\/$/, '') || ''
  return `${window.location.origin}${base}${path}`
}

export async function copyInvestigateLink(params: Parameters<typeof investigateLink>[0]) {
  const url = investigateShareUrl(params)
  try {
    await navigator.clipboard.writeText(url)
    return true
  } catch {
    return false
  }
}
