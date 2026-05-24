interface VersionConsistencyBadgeProps {
  runningHash: string
  latestHash: string
}

export function VersionConsistencyBadge({ runningHash, latestHash }: VersionConsistencyBadgeProps) {
  const consistent = runningHash === latestHash

  if (consistent) {
    return (
      <span className="version-badge version-badge-ok" title="运行版本与最新保存版本一致">
        ✓ 配置一致
      </span>
    )
  }

  return (
    <span className="version-badge version-badge-warn" title="运行版本与最新保存版本不一致，可能存在未发布的变更">
      ⚠ 配置已变更未发布
    </span>
  )
}
