export type ChartColors = {
  ok: string
  accent: string
  warn: string
  danger: string
  muted: string
  grid: string
  text: string
}

export function readChartColors(): ChartColors {
  const s = getComputedStyle(document.documentElement)
  const pick = (name: string, fallback: string) => s.getPropertyValue(name).trim() || fallback
  return {
    ok: pick('--ok', '#22c55e'),
    accent: pick('--accent', '#3b82f6'),
    warn: pick('--warn', '#eab308'),
    danger: pick('--danger', '#ef4444'),
    muted: pick('--text-muted', '#94a3b8'),
    grid: pick('--border', '#334155'),
    text: pick('--text', '#e2e8f0'),
  }
}
