interface ConfigGovernanceBadgesProps {
  fileHash: string
  runtimeHash?: string
  latestRevisionHash: string
  runtimeDrift?: boolean
  revisionDrift?: boolean
}

export function ConfigGovernanceBadges({
  fileHash,
  runtimeHash,
  latestRevisionHash,
  runtimeDrift,
  revisionDrift,
}: ConfigGovernanceBadgesProps) {
  const items: Array<{ key: string; className: string; title: string; label: string }> = []

  if (runtimeDrift) {
    items.push({
      key: 'runtime',
      className: 'version-badge version-badge-danger',
      title: `磁盘配置 (${fileHash}) 与运行中 (${runtimeHash || '—'}) 不一致，需 reload`,
      label: '需 reload',
    })
  }

  if (revisionDrift) {
    items.push({
      key: 'revision',
      className: 'version-badge version-badge-warn',
      title: `磁盘 (${fileHash}) 与最新发布版本 (${latestRevisionHash}) 不一致`,
      label: '未对齐版本',
    })
  }

  if (items.length === 0 && fileHash && latestRevisionHash && fileHash === latestRevisionHash) {
    return (
      <span className="version-badge version-badge-ok" title="磁盘、运行与最新版本一致">
        ✓ 一致
      </span>
    )
  }

  if (items.length === 0) {
    return null
  }

  return (
    <>
      {items.map((it) => (
        <span key={it.key} className={it.className} title={it.title}>
          ⚠ {it.label}
        </span>
      ))}
    </>
  )
}
