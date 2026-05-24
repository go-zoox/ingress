import { useRef, useState } from 'react'
import {
  FormCheckbox,
  FormField,
  FormGrid,
  FormSection,
  FormSelectField,
} from '../Form'
import {
  ConfigEntityModal,
  EntityRowActions,
  EntityTableToolbar,
} from '../ConfigEntityModal'
import {
  emptyWAFRuleForm,
  formTargets,
  patchWafAllow,
  patchWafDeny,
  patchWafRules,
  wafAllowList,
  wafDenyList,
  wafRuleSaveDisabled,
  wafRuleToForm,
  wafRuleToRow,
  wafRulesFromDoc,
  type WAFRuleForm,
} from '../../lib/wafEntities'
import { WAF_BUILTIN_RULES } from '../../lib/wafBuiltinRules'
import { str } from '../../lib/ingressModuleForms'

function WAFRuleFormFields({
  form,
  onChange,
}: {
  form: WAFRuleForm
  onChange: (next: WAFRuleForm) => void
}) {
  const patch = (fn: (next: WAFRuleForm) => void) => {
    const next = { ...form }
    fn(next)
    onChange(next)
  }

  return (
    <FormGrid columns={1}>
      <FormField
        label="规则 ID"
        keyName="waf.rules[].id"
        hint="必填；与内置或全局规则同 id 时可覆盖"
        value={form.id}
        onChange={(e) => patch((n) => { n.id = e.target.value })}
      />
      <FormField
        label="名称 name"
        keyName="waf.rules[].name"
        value={form.name}
        onChange={(e) => patch((n) => { n.name = e.target.value })}
      />
      <FormSelectField
        label="匹配类型 type"
        keyName="waf.rules[].type"
        value={form.type}
        onChange={(e) => patch((n) => { n.type = e.target.value as WAFRuleForm['type'] })}
      >
        <option value="regex">regex（Go regexp）</option>
        <option value="contains">contains（子串）</option>
      </FormSelectField>
      <FormField
        label="模式 pattern"
        keyName="waf.rules[].pattern"
        value={form.pattern}
        onChange={(e) => patch((n) => { n.pattern = e.target.value })}
      />
      <FormSection title="匹配目标 targets">
        <FormCheckbox
          label="path"
          checked={form.target_path}
          onChange={(v) => patch((n) => { n.target_path = v })}
        />
        <FormCheckbox
          label="query"
          checked={form.target_query}
          onChange={(v) => patch((n) => { n.target_query = v })}
        />
        <FormCheckbox
          label="uri"
          checked={form.target_uri}
          onChange={(v) => patch((n) => { n.target_uri = v })}
        />
        <FormCheckbox
          label="headers（全部请求头）"
          checked={form.target_headers}
          onChange={(v) => patch((n) => { n.target_headers = v })}
        />
        <FormField
          label="额外目标"
          keyName="waf.rules[].targets.extra"
          hint="逗号分隔，如 header:User-Agent, header:Authorization"
          value={form.target_extra}
          onChange={(e) => patch((n) => { n.target_extra = e.target.value })}
        />
        {formTargets(form).length === 0 && (
          <p className="form-hint form-hint-warn">至少选择一个 target</p>
        )}
      </FormSection>
      <FormCheckbox
        label="仅审计不拦截（log_only）"
        checked={form.log_only}
        onChange={(v) => patch((n) => { n.log_only = v })}
      />
    </FormGrid>
  )
}

function IPListEditor({
  title,
  items,
  onChange,
}: {
  title: string
  items: string[]
  onChange: (items: string[]) => void
}) {
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const text = items.join('\n')

  const trySave = () => {
    const el = textareaRef.current
    if (!el) return
    const lines = el.value
      .split('\n')
      .map((s) => s.trim())
      .filter(Boolean)
    onChange(lines)
  }

  return (
    <FormSection title={title}>
      <p className="form-hint">每行一个 CIDR 或 IP，如 203.0.113.0/24 或 192.168.1.1</p>
      <textarea
        ref={textareaRef}
        className="code"
        rows={items.length < 3 ? 3 : Math.min(items.length + 1, 8)}
        spellCheck={false}
        defaultValue={text}
        onBlur={trySave}
      />
      <p className="form-hint" style={{ marginTop: 4 }}>
        {items.length > 0 ? `${items.length} 条记录 · 修改后失焦保存` : '暂未配置'}
      </p>
    </FormSection>
  )
}

export function WafRulesEditor({
  doc,
  onChange,
}: {
  doc: Record<string, unknown>
  onChange: (doc: Record<string, unknown>) => void
}) {
  const rules = wafRulesFromDoc(doc)
  const denyList = wafDenyList(doc)
  const allowList = wafAllowList(doc)
  const [modalOpen, setModalOpen] = useState(false)
  const [editIndex, setEditIndex] = useState<number | null>(null)
  const [draft, setDraft] = useState<WAFRuleForm>(emptyWAFRuleForm())

  const patchRules = (rows: Record<string, unknown>[]) => {
    onChange(patchWafRules(doc, rows))
  }

  const openAdd = () => {
    setEditIndex(null)
    setDraft(emptyWAFRuleForm())
    setModalOpen(true)
  }

  const openEdit = (index: number) => {
    setEditIndex(index)
    setDraft(wafRuleToForm(rules[index]))
    setModalOpen(true)
  }

  const save = () => {
    if (wafRuleSaveDisabled(draft)) return
    const row = wafRuleToRow(draft)
    const next = [...rules]
    if (editIndex == null) next.push(row)
    else next[editIndex] = row
    patchRules(next)
    setModalOpen(false)
  }

  const remove = (index: number) => {
    const id = str(rules[index]?.id)
    if (!window.confirm(`删除 WAF 规则 ${id || `#${index + 1}`}？`)) return
    patchRules(rules.filter((_, i) => i !== index))
  }

  return (
    <>
      <FormSection title={`内置规则 (${WAF_BUILTIN_RULES.length})`}>
        <table className="data config-waf-rules-table">
          <thead>
            <tr>
              <th>ID</th>
              <th>名称</th>
              <th>类型</th>
              <th>Pattern</th>
              <th>Targets</th>
            </tr>
          </thead>
          <tbody>
            {WAF_BUILTIN_RULES.map((rule) => (
              <tr key={rule.id}>
                <td><code>{rule.id}</code></td>
                <td>{rule.name}</td>
                <td>{rule.type}</td>
                <td><code className="path-cell">{rule.pattern}</code></td>
                <td>{rule.targets.join(', ')}</td>
              </tr>
            ))}
          </tbody>
        </table>
        <p className="form-hint">
          内置规则为只读，全局启用/禁用通过上方的「禁用内置规则」控制。同 id 的自定义规则可覆盖内置规则。
        </p>
      </FormSection>

      <IPListEditor
        title={`IP 黑名单 deny (${denyList.length})`}
        items={denyList}
        onChange={(items) => onChange(patchWafDeny(doc, items))}
      />

      <IPListEditor
        title={`IP 白名单 allow (${allowList.length})`}
        items={allowList}
        onChange={(items) => onChange(patchWafAllow(doc, items))}
      />

      <FormSection title={`自定义规则 (${rules.length})`}>
        <EntityTableToolbar label="waf.rules" onAdd={openAdd} />
        <table className="data config-waf-rules-table">
          <thead>
            <tr>
              <th>ID</th>
              <th>类型</th>
              <th>Pattern</th>
              <th>Targets</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {rules.length === 0 ? (
              <tr>
                <td colSpan={5} className="empty-hint">
                  无自定义规则；内置规则默认启用（除非 disable_builtin）
                </td>
              </tr>
            ) : (
              rules.map((rule, i) => (
                <tr key={`${str(rule.id)}-${i}`}>
                  <td><code>{str(rule.id)}</code></td>
                  <td>{str(rule.type, 'regex')}</td>
                  <td><code className="path-cell">{str(rule.pattern)}</code></td>
                  <td>{arrTargets(rule)}</td>
                  <td>
                    <EntityRowActions onEdit={() => openEdit(i)} onDelete={() => remove(i)} />
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
        <p className="form-hint">
          v1 不扫描 body；<code>regex</code> 使用 Go regexp，<code>contains</code> 为子串匹配。
          同 id 的规则可覆盖内置或全局定义。
        </p>
      </FormSection>

      <ConfigEntityModal
        open={modalOpen}
        title={editIndex == null ? '添加 WAF 规则' : '编辑 WAF 规则'}
        wide
        onClose={() => setModalOpen(false)}
        onSave={save}
        disableSave={wafRuleSaveDisabled(draft)}
      >
        <WAFRuleFormFields form={draft} onChange={setDraft} />
      </ConfigEntityModal>
    </>
  )
}

function arrTargets(rule: Record<string, unknown>): string {
  const targets = Array.isArray(rule.targets) ? rule.targets : []
  return targets.map((t) => str(t)).join(', ') || '—'
}
