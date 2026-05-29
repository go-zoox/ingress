/** Display normalization for log lines (aligned with core/admin/service/accesslog.go NormalizeLogLine). */

const reANSI = /\x1b\[[0-9;]*m/g
const reLogTime = /^(\d{4}\/\d{2}\/\d{2} \d{2}:\d{2}:\d{2})\s+/

export function normalizeLogLine(line: string): string {
  let s = line.trim()
  if (!s) return s
  s = s.replace(reANSI, '').trim()

  const m1 = reLogTime.exec(s)
  if (!m1) return s

  const ts1 = m1[1]
  const rest = s.slice(m1[0].length).trim()
  const m2 = reLogTime.exec(rest)
  if (!m2) return s
  if (ts1 !== m2[1]) return s
  return rest
}
