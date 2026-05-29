import type { ReactNode } from 'react'

export type EntityFormSection = {
  id: string
  label: string
  description?: string
  disabled?: boolean
  badge?: string | number
}

export function EntityFormLayout({
  sections,
  activeSection,
  onSectionChange,
  subhead,
  children,
}: {
  sections: EntityFormSection[]
  activeSection: string
  onSectionChange: (id: string) => void
  /** Renders below the section title (replaces description when set). */
  subhead?: ReactNode
  children: ReactNode
}) {
  const active = sections.find((s) => s.id === activeSection) ?? sections[0]

  return (
    <div className="entity-form-layout">
      <aside className="entity-form-nav" aria-label="表单分区">
        {sections.map((section) => (
          <button
            key={section.id}
            type="button"
            className={`entity-form-tab ${activeSection === section.id ? 'active' : ''}${section.disabled ? ' disabled' : ''}`}
            onClick={() => onSectionChange(section.id)}
            aria-current={activeSection === section.id ? 'page' : undefined}
          >
            <span>{section.label}</span>
            {section.badge != null && section.badge !== '' && section.badge !== 0 ? (
              <em className="entity-form-badge">{section.badge}</em>
            ) : null}
          </button>
        ))}
      </aside>
      <div className="entity-form-panel">
        {active ? (
          <header className="entity-form-panel-head">
            <h3>{active.label}</h3>
            {subhead ? (
              <div className="entity-form-panel-subhead">{subhead}</div>
            ) : active.description ? (
              <p>{active.description}</p>
            ) : null}
          </header>
        ) : null}
        <div className="entity-form-panel-body">{children}</div>
      </div>
    </div>
  )
}

export function EntityFormUnavailable({
  title,
  detail,
}: {
  title: string
  detail?: string
}) {
  return (
    <div className="entity-form-unavailable">
      <p>{title}</p>
      {detail ? <p className="form-hint">{detail}</p> : null}
    </div>
  )
}
