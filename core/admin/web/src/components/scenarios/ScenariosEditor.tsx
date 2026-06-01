import { forwardRef, useImperativeHandle, useMemo, useState } from 'react'
import { CheckCircle2, ChevronDown, ChevronUp, Pencil, Plus, Trash2 } from 'lucide-react'
import { ScenariosActiveSelect } from './ScenariosActiveSelect'
import { ScenarioItemDrawer } from './ScenarioItemDrawer'
import {
  DEFAULT_SCENARIO_DESCRIPTION,
  DEFAULT_SCENARIO_ID,
  DEFAULT_SCENARIO_LABEL,
  emptyScenarioItem,
  isDefaultScenario,
  scenarioIdFromLabel,
  overlaySummaryKeys,
  patchModuleDocScenarios,
  scenariosFromModuleDoc,
  validateScenarioMaintenanceOverlay,
  validateScenariosFormState,
  type ScenarioItemForm,
  type ScenariosFormState,
} from '../../lib/scenarios'

type Props = {
  doc: Record<string, unknown>
  onChange: (doc: Record<string, unknown>) => void
  hostOptions?: string[]
  activatingId?: string
  onActivate?: (id: string) => void
  onNotify?: (message: string, type?: 'error' | 'success') => void
  showToolbar?: boolean
}

export type ScenariosEditorHandle = {
  openCreate: () => void
}

function cloneScenarioItem(item: ScenarioItemForm): ScenarioItemForm {
  return JSON.parse(JSON.stringify(item)) as ScenarioItemForm
}

export const ScenariosEditor = forwardRef<ScenariosEditorHandle, Props>(function ScenariosEditor(
  {
    doc,
    onChange,
    hostOptions = [],
    activatingId = '',
    onActivate,
    onNotify,
    showToolbar = true,
  },
  ref,
) {
  const state = useMemo(() => scenariosFromModuleDoc(doc), [doc])
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [editIndex, setEditIndex] = useState(-1)
  const [draft, setDraft] = useState<ScenarioItemForm | null>(null)
  const [isNew, setIsNew] = useState(false)

  const applyState = (next: ScenariosFormState) => {
    onChange(patchModuleDocScenarios(doc, next))
  }

  const openCreate = () => {
    setDraft(emptyScenarioItem())
    setIsNew(true)
    setEditIndex(-1)
    setDrawerOpen(true)
  }

  useImperativeHandle(ref, () => ({ openCreate }), [])

  const openEdit = (index: number) => {
    setDraft(cloneScenarioItem(state.items[index]))
    setIsNew(false)
    setEditIndex(index)
    setDrawerOpen(true)
  }

  const saveDrawer = () => {
    if (!draft) return
    if (!draft.label.trim()) {
      onNotify?.('请填写显示名称', 'error')
      return
    }

    const existingIds = state.items.map((i) => i.id)
    const resolved: ScenarioItemForm = {
      ...draft,
      id: isNew ? scenarioIdFromLabel(draft.label, existingIds) : draft.id.trim(),
      label: draft.label.trim(),
    }

    if (resolved.id === DEFAULT_SCENARIO_ID) {
      onNotify?.(`场景 ID「${DEFAULT_SCENARIO_ID}」为系统保留，表示根配置`, 'error')
      return
    }

    const nextItems = isNew
      ? [...state.items, resolved]
      : state.items.map((item, i) => (i === editIndex ? resolved : item))

    const dup = nextItems.filter((item) => item.id === resolved.id)
    if (dup.length > 1) {
      onNotify?.(`场景 ID「${resolved.id}」已存在`, 'error')
      return
    }

    const nextState: ScenariosFormState = {
      active: state.active,
      items: nextItems,
    }
    const maintErr = validateScenarioMaintenanceOverlay(resolved)
    if (maintErr) {
      onNotify?.(maintErr, 'error')
      return
    }

    const validationErr = validateScenariosFormState(nextState)
    if (validationErr) {
      onNotify?.(validationErr, 'error')
      return
    }

    applyState({ ...nextState, active: state.active })
    setDrawerOpen(false)
    setDraft(null)
    onNotify?.(
      isNew
        ? `已添加场景「${resolved.label}」（ID: ${resolved.id}，未写入磁盘，请保存与发布）`
        : '已更新场景（未写入磁盘，请保存与发布）',
    )
  }

  const removeItem = (index: number) => {
    const item = state.items[index]
    if (!window.confirm(`删除场景「${item.label || item.id}」？`)) return
    const items = state.items.filter((_, i) => i !== index)
    let active = state.active
    if (active === item.id) active = DEFAULT_SCENARIO_ID
    applyState({ active, items })
  }

  const moveItem = (index: number, dir: -1 | 1) => {
    const j = index + dir
    if (j < 0 || j >= state.items.length) return
    const items = [...state.items]
    ;[items[index], items[j]] = [items[j], items[index]]
    applyState({ ...state, items })
  }

  const defaultActive = isDefaultScenario(state.active)

  return (
    <>
      {showToolbar ? (
        <div className="scenarios-editor-toolbar">
          <ScenariosActiveSelect doc={doc} onChange={onChange} />
          <button type="button" className="btn btn-sm btn-primary" onClick={openCreate}>
            <Plus size={14} aria-hidden /> 新建场景
          </button>
        </div>
      ) : null}

      <div className="scenario-card-list">
        <article className={`scenario-row-default${defaultActive ? ' scenario-row-default-active' : ''}`}>
          <div className="scenario-row-default-main">
            <strong>{DEFAULT_SCENARIO_LABEL}</strong>
            <code>{DEFAULT_SCENARIO_ID}</code>
            <span className="muted">{DEFAULT_SCENARIO_DESCRIPTION}</span>
            {defaultActive ? (
              <span className="badge badge-ok">
                <CheckCircle2 size={12} aria-hidden /> 当前
              </span>
            ) : null}
          </div>
          {!defaultActive && onActivate ? (
            <button
              type="button"
              className="btn btn-sm btn-primary"
              disabled={activatingId !== ''}
              onClick={() => onActivate(DEFAULT_SCENARIO_ID)}
            >
              {activatingId === DEFAULT_SCENARIO_ID ? '切换中…' : '切换生效'}
            </button>
          ) : null}
        </article>

      {state.items.length === 0 ? (
        <p className="empty-hint" style={{ gridColumn: '1 / -1' }}>
          暂无 overlay 场景。点击「新建场景」添加直播等差异配置。
        </p>
      ) : (
          <>
          {state.items.map((item, index) => {
            const keys = overlaySummaryKeys(item)
            const isActive = item.id === state.active
            return (
              <article key={item.id} className={`scenario-card${isActive ? ' scenario-card-active' : ''}`}>
                <div className="scenario-card-main">
                  <div className="scenario-card-title">
                    <strong>{item.label || item.id}</strong>
                    <code>{item.id}</code>
                    {isActive ? (
                      <span className="badge badge-ok">
                        <CheckCircle2 size={12} aria-hidden /> 当前
                      </span>
                    ) : null}
                  </div>
                  {item.description ? <p className="scenario-card-desc">{item.description}</p> : null}
                  <div className="scenario-card-tags">
                    {keys.length === 0 ? (
                      <span className="muted">无 overlay</span>
                    ) : (
                      keys.map((k) => (
                        <code key={k} className="tag-inline">{k}</code>
                      ))
                    )}
                  </div>
                </div>
                <div className="scenario-card-actions">
                  <button type="button" className="btn btn-sm btn-ghost" title="上移" disabled={index === 0} onClick={() => moveItem(index, -1)}>
                    <ChevronUp size={14} aria-hidden />
                  </button>
                  <button type="button" className="btn btn-sm btn-ghost" title="下移" disabled={index === state.items.length - 1} onClick={() => moveItem(index, 1)}>
                    <ChevronDown size={14} aria-hidden />
                  </button>
                  <button type="button" className="btn btn-sm btn-ghost" onClick={() => openEdit(index)}>
                    <Pencil size={14} aria-hidden /> 编辑
                  </button>
                  {!isActive && onActivate ? (
                    <button
                      type="button"
                      className="btn btn-sm btn-primary"
                      disabled={activatingId !== ''}
                      onClick={() => onActivate(item.id)}
                    >
                      {activatingId === item.id ? '切换中…' : '切换生效'}
                    </button>
                  ) : null}
                  <button type="button" className="btn btn-sm btn-ghost" onClick={() => removeItem(index)}>
                    <Trash2 size={14} aria-hidden />
                  </button>
                </div>
              </article>
            )
          })}
          </>
      )}
      </div>

      <ScenarioItemDrawer
        open={drawerOpen}
        form={draft}
        isNew={isNew}
        existingIds={state.items.map((i) => i.id)}
        hostOptions={hostOptions}
        onChange={setDraft}
        onClose={() => {
          setDrawerOpen(false)
          setDraft(null)
        }}
        onSave={saveDrawer}
      />
    </>
  )
})
