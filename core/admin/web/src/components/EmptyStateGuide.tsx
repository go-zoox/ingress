import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'

type Props = {
  title: string
  children: ReactNode
  configModule?: string
  linkTo?: string
  linkLabel?: string
}

export function EmptyStateGuide({ title, children, configModule, linkTo = '/config', linkLabel }: Props) {
  return (
    <div className="empty-state-guide">
      <p className="empty-state-title">{title}</p>
      <p className="empty-hint">{children}</p>
      {configModule ? (
        <p className="empty-hint">
          可在 <Link to={linkTo}>配置</Link> 的模块编辑器中添加{' '}
          <code>{configModule}</code> 段。
        </p>
      ) : null}
      {linkLabel ? (
        <p>
          <Link to={linkTo} className="btn btn-ghost btn-sm">
            {linkLabel}
          </Link>
        </p>
      ) : null}
    </div>
  )
}
