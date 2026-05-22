export function HostBadge({ t }: { t: string }) {
  const c =
    t === 'regex' ? 'badge-regex' : t === 'wildcard' ? 'badge-wildcard' : 'badge-exact'
  return <span className={`badge ${c}`}>{t}</span>
}
