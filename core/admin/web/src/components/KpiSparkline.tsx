type Props = {
  values: number[]
  tone?: string
  className?: string
}

/** Minimal bar sparkline used on overview and route detail KPI cards. */
export function KpiSparkline({ values, tone, className }: Props) {
  if (values.length <= 1) {
    return null
  }
  const max = Math.max(1, ...values)
  return (
    <div className={className ?? 'kpi-sparkline'} aria-hidden>
      {values.map((v, i) => (
        <span
          key={i}
          style={{
            height: `${Math.max(4, (v / max) * 100)}%`,
            background: tone ?? 'var(--accent)',
          }}
        />
      ))}
    </div>
  )
}
