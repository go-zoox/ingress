/** Client-side access log line parser (aligned with core/admin/service/accesslog.go). */

export type ParsedAccessLine = {
  host: string
  method: string
  path: string
  status: number
  durationMs: number
  cacheHit: boolean
  wafBlock: boolean
  raw: string
}

const reANSI = /\x1b\[[0-9;]*m/g
const reLogTime = /^(\d{4}\/\d{2}\/\d{2} \d{2}:\d{2}:\d{2})\s+/
const reLogLev = /^(DEBUG|INFO|WARN|ERROR|FATAL)\s+/
const reHostTag = /\[host:\s*([^,\]]+)/
const reArrowHost = /^\S+\s+(\S+)\s+->/
const reRequest = /"([A-Z]+)\s+([^\s]+)\s+HTTP\/[^"]+"\s+(\d{3})\s+(\d+(?:\.\d+)?)(ms|s)?/

export function parseAccessLogLine(line: string): ParsedAccessLine | null {
  let s = line.trim()
  if (!s) return null
  s = s.replace(reANSI, '')
  if (reLogTime.test(s)) {
    s = s.replace(reLogTime, '').trim()
  }
  if (reLogTime.test(s)) {
    s = s.replace(reLogTime, '').trim()
  }
  s = s.replace(reLogLev, '')

  let host = ''
  const hostTag = reHostTag.exec(s)
  if (hostTag?.[1]) host = hostTag[1].trim()
  else {
    const arrow = reArrowHost.exec(s)
    if (arrow?.[1]) host = arrow[1].trim()
  }
  if (!host) return null

  const req = reRequest.exec(s)
  if (!req) return null

  const status = Number(req[3])
  let durationMs = Number(req[4] || 0)
  if (req[5] === 's') durationMs *= 1000

  return {
    host,
    method: req[1],
    path: req[2],
    status,
    durationMs,
    cacheHit: s.includes('cache_hit=1'),
    wafBlock: s.includes('waf_block=1'),
    raw: line,
  }
}

export function logLineStatusClass(status: number) {
  if (status >= 500 || status >= 400) return 'status-4xx'
  if (status >= 200 && status < 300) return 'status-2xx'
  return ''
}
