import type { ReactNode } from 'react'

type Props = {
  title: string
  desc: string
  actions?: ReactNode
}

export function PageHeader({ title, desc, actions }: Props) {
  return (
    <header className={`page-header${actions ? ' page-header-with-actions' : ''}`}>
      <div className="page-header-main">
        <h1>{title}</h1>
        <p>{desc}</p>
      </div>
      {actions ? <div className="page-header-actions">{actions}</div> : null}
    </header>
  )
}
