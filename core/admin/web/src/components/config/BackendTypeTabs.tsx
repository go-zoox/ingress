import type { RuleBackendType } from '../../lib/configEntities'

type TabDef = {
  id: RuleBackendType
  label: string
  hint: string
}

const ALL_TABS: TabDef[] = [
  { id: 'service', label: '上游代理', hint: 'service' },
  { id: 'handler', label: '直接响应', hint: 'handler' },
  { id: 'redirect', label: '重定向', hint: 'redirect' },
]

type Props = {
  value: RuleBackendType | string
  onChange: (value: RuleBackendType) => void
  /** fallback 仅 service / redirect */
  modes?: RuleBackendType[]
}

export function BackendTypeTabs({ value, onChange, modes }: Props) {
  const allowed = modes ?? (['service', 'handler', 'redirect'] as RuleBackendType[])
  const tabs = ALL_TABS.filter((t) => allowed.includes(t.id))

  return (
    <div className="backend-type-tabs" role="tablist" aria-label="Backend 类型">
      {tabs.map((tab) => {
        const active = value === tab.id
        return (
          <button
            key={tab.id}
            type="button"
            role="tab"
            aria-selected={active}
            className={`backend-type-tab${active ? ' active' : ''}`}
            onClick={() => onChange(tab.id)}
          >
            <span className="backend-type-tab-label">{tab.label}</span>
            <span className="backend-type-tab-hint">{tab.hint}</span>
          </button>
        )
      })}
    </div>
  )
}
