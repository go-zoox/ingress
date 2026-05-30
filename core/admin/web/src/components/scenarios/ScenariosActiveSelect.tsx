import {
  DEFAULT_SCENARIO_ID,
  DEFAULT_SCENARIO_LABEL,
  isDefaultScenario,
  patchModuleDocScenarios,
  scenariosFromModuleDoc,
} from '../../lib/scenarios'

type Props = {
  doc: Record<string, unknown>
  onChange: (doc: Record<string, unknown>) => void
  disabled?: boolean
}

export function ScenariosActiveSelect({ doc, onChange, disabled }: Props) {
  const state = scenariosFromModuleDoc(doc)
  const defaultActive = isDefaultScenario(state.active)

  const setActive = (active: string) => {
    onChange(patchModuleDocScenarios(doc, { ...state, active }))
  }

  return (
    <label className="scenario-active-inline">
      <span className="scenario-active-inline-label">当前场景</span>
      <select
        className="form-control scenario-active-inline-select"
        value={defaultActive ? DEFAULT_SCENARIO_ID : state.active}
        disabled={disabled}
        onChange={(e) => setActive(e.target.value)}
      >
        <option value={DEFAULT_SCENARIO_ID}>{DEFAULT_SCENARIO_LABEL}（根配置）</option>
        {state.items.map((item) => (
          <option key={item.id} value={item.id}>
            {item.label || item.id}
          </option>
        ))}
      </select>
    </label>
  )
}
