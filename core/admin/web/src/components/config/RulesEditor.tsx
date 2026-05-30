import { forwardRef, useImperativeHandle, useState } from 'react'
import {
  ConfigEntityModal,
  EntityRowActions,
  EntityTableToolbar,
} from '../ConfigEntityModal'
import { RulePathsModal } from './RulePathsModal'
import {
  defaultRuleEntitySection,
  RuleEntityFormSections,
  type RuleEntitySectionId,
} from './RuleEntityFormSections'
import {
  emptyRuleForm,
  formToRule,
  ruleSaveDisabled,
  ruleSummary,
  ruleToForm,
  rulesFromDoc,
  type RuleForm,
} from '../../lib/configEntities'
import { arr, str } from '../../lib/ingressModuleForms'
import { moveAdjacent } from '../../lib/arrayMove'
import type { ServiceForm } from '../../lib/services'

export type RulesEditorHandle = {
  openAdd: () => void
}

export const RulesEditor = forwardRef<
  RulesEditorHandle,
  {
    doc: Record<string, unknown>
    onChange: (doc: Record<string, unknown>) => void
    serviceCatalog?: ServiceForm[]
    serviceFieldMode?: 'manual' | 'catalog-select'
    /** pathIndex omitted or -1 = Host 级规则详情 */
    onOpenDetail?: (ruleIndex: number, pathIndex?: number) => void
    /** 路由页隐藏 rules 标签与优先级说明，添加按钮由外部工具栏提供 */
    hideTableChrome?: boolean
  }
>(function RulesEditor(
  {
    doc,
    onChange,
    serviceCatalog,
    serviceFieldMode = 'manual',
    onOpenDetail,
    hideTableChrome = false,
  },
  ref,
) {
  const rules = rulesFromDoc(doc)
  const [modalOpen, setModalOpen] = useState(false)
  const [pathsModalIndex, setPathsModalIndex] = useState<number | null>(null)
  const [editIndex, setEditIndex] = useState<number | null>(null)
  const [draft, setDraft] = useState<RuleForm>(emptyRuleForm())
  const [activeSection, setActiveSection] = useState<RuleEntitySectionId>('basic')

  const patchRules = (rows: Record<string, unknown>[]) => {
    onChange({ rules: rows })
  }

  const openAdd = () => {
    setEditIndex(null)
    setDraft(emptyRuleForm())
    setActiveSection(defaultRuleEntitySection('rule'))
    setModalOpen(true)
  }

  useImperativeHandle(ref, () => ({ openAdd }), [])

  const openEdit = (index: number) => {
    setEditIndex(index)
    setDraft(ruleToForm(rules[index]))
    setActiveSection(defaultRuleEntitySection('rule'))
    setModalOpen(true)
  }

  const openPaths = (index: number) => {
    setPathsModalIndex(index)
  }

  const save = () => {
    if (!draft.host.trim()) return
    const row = formToRule(draft, editIndex == null ? undefined : rules[editIndex])
    const next = [...rules]
    if (editIndex == null) next.push(row)
    else next[editIndex] = row
    patchRules(next)
    setModalOpen(false)
  }

  const remove = (index: number) => {
    const host = str(rules[index]?.host)
    if (!window.confirm(`删除路由规则 ${host || `#${index + 1}`}？`)) return
    patchRules(rules.filter((_, i) => i !== index))
  }

  const moveRule = (index: number, delta: -1 | 1) => {
    const j = index + delta
    if (j < 0 || j >= rules.length) return
    patchRules(moveAdjacent(rules, index, delta))
    if (pathsModalIndex === index) setPathsModalIndex(j)
    else if (pathsModalIndex === j) setPathsModalIndex(index)
  }

  const savePaths = (nextRule: Record<string, unknown>) => {
    if (pathsModalIndex == null) return
    const next = [...rules]
    next[pathsModalIndex] = nextRule
    patchRules(next)
  }

  const pathsRule = pathsModalIndex == null ? null : rules[pathsModalIndex]

  return (
    <>
      {!hideTableChrome ? (
        <>
          <EntityTableToolbar label="rules" onAdd={openAdd} />
          <p className="form-hint">列表顺序即匹配优先级，排在前面的规则优先匹配。</p>
        </>
      ) : null}
      <table className="data config-rules-table">
        <thead>
          <tr>
            <th>#</th>
            <th>Host</th>
            <th>类型</th>
            <th>Backend</th>
            <th>Paths</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          {rules.length === 0 ? (
            <tr>
              <td colSpan={6} className="empty-hint">
                无路由规则，点击「添加」
              </td>
            </tr>
          ) : (
            rules.map((rule, i) => {
              const pathCount = arr(rule.paths).length
              return (
                <tr key={`${str(rule.host)}-${i}`}>
                  <td>{i + 1}</td>
                  <td>
                    {onOpenDetail ? (
                      <button
                        type="button"
                        className="action-link config-host-detail-link"
                        title="查看路由详情"
                        onClick={() => onOpenDetail(i, -1)}
                      >
                        <code>{str(rule.host)}</code>
                      </button>
                    ) : (
                      <code>{str(rule.host)}</code>
                    )}
                  </td>
                  <td>{str(rule.host_type, 'auto')}</td>
                  <td>{ruleSummary(rule)}</td>
                  <td>
                    <button
                      type="button"
                      className="action-link config-paths-link"
                      onClick={() => openPaths(i)}
                    >
                      {pathCount > 0 ? `${pathCount} 条` : '配置'}
                    </button>
                  </td>
                  <td>
                    <EntityRowActions
                      onEdit={() => openEdit(i)}
                      onDelete={() => remove(i)}
                      onMoveUp={() => moveRule(i, -1)}
                      onMoveDown={() => moveRule(i, 1)}
                      disableMoveUp={i === 0}
                      disableMoveDown={i === rules.length - 1}
                      menuItems={[
                        { label: 'Paths', onClick: () => openPaths(i) },
                        ...(onOpenDetail
                          ? [{ label: '详情', onClick: () => onOpenDetail(i, -1) }]
                          : []),
                      ]}
                    />
                  </td>
                </tr>
              )
            })
          )}
        </tbody>
      </table>

      <ConfigEntityModal
        open={modalOpen}
        title={editIndex == null ? '添加路由规则' : '编辑路由规则'}
        wide
        onClose={() => setModalOpen(false)}
        onSave={save}
        disableSave={ruleSaveDisabled(draft)}
      >
        <RuleEntityFormSections
          form={draft}
          onChange={setDraft}
          activeSection={activeSection}
          onSectionChange={setActiveSection}
          variant="rule"
          serviceCatalog={serviceCatalog}
          serviceFieldMode={serviceFieldMode}
        />
      </ConfigEntityModal>

      {pathsRule && pathsModalIndex != null && (
        <RulePathsModal
          open
          host={str(pathsRule.host)}
          rule={pathsRule}
          onClose={() => setPathsModalIndex(null)}
          onSave={savePaths}
          serviceCatalog={serviceCatalog}
          serviceFieldMode={serviceFieldMode}
        />
      )}
    </>
  )
})
