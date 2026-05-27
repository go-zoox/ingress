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
  patchCustomRuleEnabled,
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
import {
  WAF_BUILTIN_RULES,
  defaultBuiltinEnabled,
  isBuiltinRuleEnabled,
  patchBuiltinRuleEnabled,
  type BuiltinWAFRule,
} from '../../lib/wafBuiltinRules'
import { Drawer } from '../Drawer'
import { EllipsisTooltip } from '../EllipsisTooltip'
import { obj, str } from '../../lib/ingressModuleForms'

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
        label="启用规则"
        checked={form.enabled}
        onChange={(v) => patch((n) => { n.enabled = v })}
      />
      <FormCheckbox
        label="仅审计不拦截（log_only）"
        checked={form.log_only}
        onChange={(v) => patch((n) => { n.log_only = v })}
      />
    </FormGrid>
  )
}

function RuleEnabledToggle({
  enabled,
  onChange,
  title,
}: {
  enabled: boolean
  onChange: (enabled: boolean) => void
  title?: string
}) {
  return (
    <label className="waf-rule-toggle" title={title}>
      <input
        type="checkbox"
        checked={enabled}
        onChange={(e) => onChange(e.target.checked)}
      />
      <span>{enabled ? '开' : '关'}</span>
    </label>
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

function BuiltinRuleDetailDrawer({
  rule,
  doc,
  open,
  onClose,
}: {
  rule: BuiltinWAFRule | null
  doc: Record<string, unknown>
  open: boolean
  onClose: () => void
}) {
  if (!rule) return null

  const enabled = isBuiltinRuleEnabled(doc, rule.id)
  const override = Object.prototype.hasOwnProperty.call(obj(obj(doc.waf).builtin_rules), rule.id)
  const disableAllBuiltin = !defaultBuiltinEnabled(doc)

  return (
    <Drawer
      open={open}
      title="内置规则详情"
      onClose={onClose}
      width={480}
      footer={
        <button type="button" className="btn btn-ghost" onClick={onClose}>
          关闭
        </button>
      }
    >
      <dl className="route-detail-dl">
        <dt>ID</dt>
        <dd><code>{rule.id}</code></dd>
        <dt>名称</dt>
        <dd>{rule.name}</dd>
        <dt>类型</dt>
        <dd>{rule.type}</dd>
        <dt>启用</dt>
        <dd>
          {enabled ? '开' : '关'}
          {override ? (
            <span className="badge badge-audit" style={{ marginLeft: 8 }}>
              已单独覆盖
            </span>
          ) : disableAllBuiltin ? (
            <span className="form-hint" style={{ display: 'block', marginTop: 4 }}>
              跟随全局「禁用内置规则」
            </span>
          ) : (
            <span className="form-hint" style={{ display: 'block', marginTop: 4 }}>
              跟随全局内置规则开关
            </span>
          )}
        </dd>
        <dt>检测目标</dt>
        <dd>{rule.targets.join(', ')}</dd>
        <dt>模式</dt>
        <dd>
          <code className="waf-rule-pattern">{rule.pattern}</code>
        </dd>
        {rule.description ? (
          <>
            <dt>说明</dt>
            <dd>{rule.description}</dd>
          </>
        ) : null}
      </dl>
    </Drawer>
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
  const disableAllBuiltin = !defaultBuiltinEnabled(doc)
  const [modalOpen, setModalOpen] = useState(false)
  const [editIndex, setEditIndex] = useState<number | null>(null)
  const [draft, setDraft] = useState<WAFRuleForm>(emptyWAFRuleForm())
  const [detailRule, setDetailRule] = useState<BuiltinWAFRule | null>(null)

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

  const enabledBuiltinCount = WAF_BUILTIN_RULES.filter((r) => isBuiltinRuleEnabled(doc, r.id)).length

  return (
    <>
      <FormSection title={`内置规则 (${enabledBuiltinCount}/${WAF_BUILTIN_RULES.length} 启用)`}>
        <div className="table-scroll">
        <table className="data config-waf-rules-table config-waf-rules-table--compact">
          <thead>
            <tr>
              <th className="col-enable">启用</th>
              <th className="col-name">名称</th>
              <th className="col-desc">说明</th>
              <th className="col-actions">操作</th>
            </tr>
          </thead>
          <tbody>
            {WAF_BUILTIN_RULES.map((rule) => {
              const enabled = isBuiltinRuleEnabled(doc, rule.id)
              const override = Object.prototype.hasOwnProperty.call(obj(obj(doc.waf).builtin_rules), rule.id)
              return (
                <tr key={rule.id} className={enabled ? undefined : 'row-muted'}>
                  <td className="col-enable">
                    <RuleEnabledToggle
                      enabled={enabled}
                      onChange={(v) => onChange(patchBuiltinRuleEnabled(doc, rule.id, v))}
                      title={
                        override
                          ? '已单独覆盖；清除覆盖后跟随「禁用内置规则」'
                          : disableAllBuiltin
                            ? '全局已禁用内置规则；单独开启此条'
                            : '跟随全局内置规则开关'
                      }
                    />
                  </td>
                  <td className="col-name">
                    <EllipsisTooltip text={rule.name} />
                  </td>
                  <td className="col-desc">
                    <EllipsisTooltip text={rule.description ?? ''} />
                  </td>
                  <td className="col-actions">
                    <button
                      type="button"
                      className="action-link"
                      onClick={() => setDetailRule(rule)}
                    >
                      查看详情
                    </button>
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
        </div>
        <p className="form-hint">
          默认跟随上方「禁用内置规则」：未禁用时全部内置规则启用；禁用后全部关闭，可在此单独开启。
          覆盖写入 <code>waf.builtin_rules</code>；同 id 的自定义规则仍可覆盖 pattern/targets。
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
        <div className="table-scroll">
        <table className="data config-waf-rules-table">
          <thead>
            <tr>
              <th>启用</th>
              <th>ID</th>
              <th>类型</th>
              <th className="col-pattern">Pattern</th>
              <th className="col-targets">Targets</th>
              <th className="col-actions">操作</th>
            </tr>
          </thead>
          <tbody>
            {rules.length === 0 ? (
              <tr>
                <td colSpan={6} className="empty-hint">
                  无自定义规则
                </td>
              </tr>
            ) : (
              rules.map((rule, i) => {
                const enabled = rule.enabled !== false
                return (
                  <tr key={`${str(rule.id)}-${i}`} className={enabled ? undefined : 'row-muted'}>
                    <td>
                      <RuleEnabledToggle
                        enabled={enabled}
                        onChange={(v) => onChange(patchCustomRuleEnabled(doc, i, v))}
                      />
                    </td>
                    <td><code>{str(rule.id)}</code></td>
                    <td>{str(rule.type, 'regex')}</td>
                    <td className="col-pattern"><code className="path-cell">{str(rule.pattern)}</code></td>
                    <td>{arrTargets(rule)}</td>
                    <td>
                      <EntityRowActions onEdit={() => openEdit(i)} onDelete={() => remove(i)} />
                    </td>
                  </tr>
                )
              })
            )}
          </tbody>
        </table>
        </div>
        <p className="form-hint">
          v1 不扫描 body；<code>regex</code> 使用 Go regexp，<code>contains</code> 为子串匹配。
          关闭规则写入 <code>enabled: false</code>，不会删除规则定义。
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

      <BuiltinRuleDetailDrawer
        rule={detailRule}
        doc={doc}
        open={detailRule != null}
        onClose={() => setDetailRule(null)}
      />
    </>
  )
}

function arrTargets(rule: Record<string, unknown>): string {
  const targets = Array.isArray(rule.targets) ? rule.targets : []
  return targets.map((t) => str(t)).join(', ') || '—'
}
