import { useState } from 'react'
import {
  FormField,
  FormGrid,
  FormSelectField,
} from '../Form'
import {
  ConfigEntityModal,
  EntityTableToolbar,
} from '../ConfigEntityModal'
import { BackendFormGrid, backendFormWide } from './BackendFormFields'
import { RateLimitFormFields } from './RateLimitFormFields'
import { RulePathsModal } from './RulePathsModal'
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

function RuleFormFields({
  form,
  onChange,
}: {
  form: RuleForm
  onChange: (next: RuleForm) => void
}) {
  const patch = (fn: (next: RuleForm) => void) => {
    const next = { ...form }
    fn(next)
    onChange(next)
  }

  return (
    <FormGrid columns={1}>
      <FormField
        label="Host"
        keyName="host"
        value={form.host}
        onChange={(e) => patch((n) => { n.host = e.target.value })}
      />
      <FormSelectField
        label="Host 类型"
        keyName="host_type"
        value={form.host_type}
        onChange={(e) => patch((n) => { n.host_type = e.target.value })}
      >
        <option value="auto">auto（自动推断）</option>
        <option value="exact">exact</option>
        <option value="wildcard">wildcard</option>
        <option value="regex">regex</option>
      </FormSelectField>

      <p className="form-section-label">Host 级 Backend</p>
      <BackendFormGrid<RuleForm> form={form} onChange={onChange} />

      <RateLimitFormFields<RuleForm>
        form={form}
        onChange={onChange}
        title="路由限流 rules[].rate_limit"
      />

      {form.paths.length > 0 && (
        <p className="form-hint">
          已配置 {form.paths.length} 条 path；保存 Host 后可在列表中点击「Paths」继续编辑。
        </p>
      )}
    </FormGrid>
  )
}

export function RulesEditor({
  doc,
  onChange,
}: {
  doc: Record<string, unknown>
  onChange: (doc: Record<string, unknown>) => void
}) {
  const rules = rulesFromDoc(doc)
  const [modalOpen, setModalOpen] = useState(false)
  const [pathsModalIndex, setPathsModalIndex] = useState<number | null>(null)
  const [editIndex, setEditIndex] = useState<number | null>(null)
  const [draft, setDraft] = useState<RuleForm>(emptyRuleForm())

  const patchRules = (rows: Record<string, unknown>[]) => {
    onChange({ rules: rows })
  }

  const openAdd = () => {
    setEditIndex(null)
    setDraft(emptyRuleForm())
    setModalOpen(true)
  }

  const openEdit = (index: number) => {
    setEditIndex(index)
    setDraft(ruleToForm(rules[index]))
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

  const savePaths = (nextRule: Record<string, unknown>) => {
    if (pathsModalIndex == null) return
    const next = [...rules]
    next[pathsModalIndex] = nextRule
    patchRules(next)
  }

  const pathsRule = pathsModalIndex == null ? null : rules[pathsModalIndex]

  return (
    <>
      <EntityTableToolbar label="rules" onAdd={openAdd} />
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
                    <code>{str(rule.host)}</code>
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
                    <div className="row-actions">
                      <button type="button" className="action-link" onClick={() => openEdit(i)}>
                        编辑
                      </button>
                      <button type="button" className="action-link" onClick={() => openPaths(i)}>
                        Paths
                      </button>
                      <button type="button" className="action-link action-danger" onClick={() => remove(i)}>
                        删除
                      </button>
                    </div>
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
        wide={backendFormWide(draft)}
        onClose={() => setModalOpen(false)}
        onSave={save}
        disableSave={ruleSaveDisabled(draft)}
      >
        <RuleFormFields form={draft} onChange={setDraft} />
      </ConfigEntityModal>

      {pathsRule && pathsModalIndex != null && (
        <RulePathsModal
          open
          host={str(pathsRule.host)}
          rule={pathsRule}
          onClose={() => setPathsModalIndex(null)}
          onSave={savePaths}
        />
      )}
    </>
  )
}
