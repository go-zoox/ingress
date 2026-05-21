export function escapeHtml(s: string) {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

type DiffOp = { type: 'equal' | 'add' | 'del'; line: string }

/** Longest-common-subsequence line diff (Myers-style backtrack). */
function diffLines(a: string[], b: string[]): DiffOp[] {
  const n = a.length
  const m = b.length
  const dp: number[][] = Array.from({ length: n + 1 }, () => Array(m + 1).fill(0))

  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      if (a[i] === b[j]) dp[i][j] = dp[i + 1][j + 1] + 1
      else dp[i][j] = Math.max(dp[i + 1][j], dp[i][j + 1])
    }
  }

  const ops: DiffOp[] = []
  let i = 0
  let j = 0
  while (i < n && j < m) {
    if (a[i] === b[j]) {
      ops.push({ type: 'equal', line: a[i] })
      i++
      j++
    } else if (dp[i + 1][j] >= dp[i][j + 1]) {
      ops.push({ type: 'del', line: a[i] })
      i++
    } else {
      ops.push({ type: 'add', line: b[j] })
      j++
    }
  }
  while (i < n) {
    ops.push({ type: 'del', line: a[i] })
    i++
  }
  while (j < m) {
    ops.push({ type: 'add', line: b[j] })
    j++
  }
  return ops
}

export function buildDiff(baseline: string, current: string): string {
  if (baseline === current) return '(无变更)'

  const a = baseline.split('\n')
  const b = current.split('\n')
  const ops = diffLines(a, b)
  const lines: string[] = []

  for (const op of ops) {
    if (op.type === 'equal') lines.push('  ' + escapeHtml(op.line))
    else if (op.type === 'del') lines.push('<span class="del">- ' + escapeHtml(op.line) + '</span>')
    else lines.push('<span class="add">+ ' + escapeHtml(op.line) + '</span>')
  }

  const hasChange = ops.some((op) => op.type !== 'equal')
  return hasChange ? lines.join('\n') : '(无变更)'
}

export const CONFIG_MODULE_LABELS: Record<string, string> = {
  general: '基础',
  cache: '缓存',
  logging: '日志',
  waf: 'WAF',
  healthcheck: '健康检查',
  https: 'HTTPS / TLS',
  fallback: 'Fallback',
  rules: '路由规则',
  other: '其他',
}
