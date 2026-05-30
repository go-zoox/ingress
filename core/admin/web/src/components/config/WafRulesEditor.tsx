import { useEffect, useRef, useState } from 'react'
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
} from '../ConfigEntityModal'
import {
  emptyWAFRuleForm,
  formTargets,
  patchCustomRuleEnabled,
  patchCustomRuleAction,
  patchBuiltinRuleAllowHosts,
  customActionOverridden,
  wafCustomRuleName,
  wafCustomRuleDescription,
  patchWafAllow,
  patchWafAllowHosts,
  patchWafDeny,
  patchWafRules,
  wafAllowList,
  wafAllowHostsList,
  wafDenyList,
  wafRuleSaveDisabled,
  wafRuleAllowHosts,
  wafRuleToForm,
  wafRuleToRow,
  wafRulesFromDoc,
  WAF_ACTION_OPTIONS,
  type WAFAction,
  type WAFRuleForm,
  wafActionLabel,
} from '../../lib/wafEntities'
import {
  WAF_BUILTIN_RULES,
  builtinActionOverridden,
  builtinRuleAction,
  defaultBuiltinEnabled,
  isBuiltinRuleEnabled,
  patchBuiltinRuleAction,
  patchBuiltinRuleEnabled,
  type BuiltinWAFRule,
} from '../../lib/wafBuiltinRules'
import { actionFromRow, wafInheritEffectiveLabel } from '../../lib/wafAction'
import { Drawer } from '../Drawer'
import { EllipsisTooltip } from '../EllipsisTooltip'
import { arr, obj, str, bool } from '../../lib/ingressModuleForms'
import { moveAdjacent } from '../../lib/arrayMove'
import { EditableStringList } from '../EditableStringList'

function WafActionTag({
  action,
  globalLogOnly,
  disabled,
  overridden,
}: {
  action: WAFAction
  globalLogOnly: boolean
  disabled?: boolean
  overridden?: boolean
}) {
  const label = wafActionLabel(action)
  const badgeClass =
    action === 'block' ? 'badge-block'
      : action === 'pass' ? 'badge-exact'
        : 'badge-audit'
  const title =
    action === 'inherit'
      ? `继承全局，当前生效：${wafInheritEffectiveLabel(globalLogOnly)}`
      : overridden
        ? `已单独覆盖：${label}`
        : label

  return (
    <span
      className={`badge waf-action-tag ${badgeClass}${disabled ? ' waf-action-tag--muted' : ''}`}
      title={title}
    >
      {label}
    </span>
  )
}

function RuleActionSelect({
  value,
  onChange,
  disabled,
  title,
  className = 'waf-action-select',
}: {
  value: WAFAction
  onChange: (action: WAFAction) => void
  disabled?: boolean
  title?: string
  className?: string
}) {
  return (
    <select
      className={className}
      value={value}
      title={title}
      disabled={disabled}
      onChange={(e) => onChange(e.target.value as WAFAction)}
    >
      {WAF_ACTION_OPTIONS.map((o) => (
        <option key={o.value} value={o.value} title={o.hint}>
          {o.label}
        </option>
      ))}
    </select>
  )
}

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
      <FormSelectField
        label="命中后处置 action"
        keyName="waf.rules[].action"
        hint="继承（默认）跟随全局；拦截/仅记录/通过为显式覆盖"
        value={form.action}
        onChange={(e) => patch((n) => { n.action = e.target.value as WAFAction })}
      >
        {WAF_ACTION_OPTIONS.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label} — {o.hint}
          </option>
        ))}
      </FormSelectField>
      <EditableStringList
        title="域名白名单"
        showCount
        titleTooltip="精确域名、*.wildcard.example.com，或 Go 正则（含 ( ) [ ] ^ $ 等时自动识别）"
        items={form.allow_hosts}
        valueLabel="Host 模式"
        fieldKeyName="waf.rules[].allow_hosts[]"
        emptyHint="未配置；匹配的 Host 将跳过本规则（其他规则仍评估）"
        placeholder="admin.internal 或 *.cdn.example.com"
        addTitle="添加域名白名单"
        editTitle="编辑域名白名单"
        onChange={(items) => patch((n) => { n.allow_hosts = items })}
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

function CustomRuleRowActions({
  onView,
  onEdit,
  onDelete,
  onMoveUp,
  onMoveDown,
  disableMoveUp,
  disableMoveDown,
}: {
  onView: () => void
  onEdit: () => void
  onDelete: () => void
  onMoveUp: () => void
  onMoveDown: () => void
  disableMoveUp: boolean
  disableMoveDown: boolean
}) {
  const [menuOpen, setMenuOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!menuOpen) return
    const close = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false)
      }
    }
    document.addEventListener('mousedown', close)
    return () => document.removeEventListener('mousedown', close)
  }, [menuOpen])

  return (
    <div className="row-actions">
      <button type="button" className="action-link" onClick={onView}>
        查看详情
      </button>
      <div className="action-menu" ref={menuRef}>
        <button
          type="button"
          className="action-link action-advanced"
          aria-expanded={menuOpen}
          onClick={() => setMenuOpen((v) => !v)}
        >
          高级
        </button>
        {menuOpen && (
          <div className="action-menu-panel" role="menu">
            <button
              type="button"
              role="menuitem"
              className="action-menu-item"
              onClick={() => {
                setMenuOpen(false)
                onEdit()
              }}
            >
              编辑
            </button>
            <button
              type="button"
              role="menuitem"
              className="action-menu-item"
              disabled={disableMoveUp}
              onClick={() => {
                if (disableMoveUp) return
                setMenuOpen(false)
                onMoveUp()
              }}
            >
              上移
            </button>
            <button
              type="button"
              role="menuitem"
              className="action-menu-item"
              disabled={disableMoveDown}
              onClick={() => {
                if (disableMoveDown) return
                setMenuOpen(false)
                onMoveDown()
              }}
            >
              下移
            </button>
            <div className="action-menu-sep" aria-hidden />
            <button
              type="button"
              role="menuitem"
              className="action-menu-item action-danger"
              onClick={() => {
                setMenuOpen(false)
                onDelete()
              }}
            >
              删除
            </button>
          </div>
        )}
      </div>
    </div>
  )
}

function CustomRuleDetailDrawer({
  ruleIndex,
  doc,
  open,
  onClose,
  onDocChange,
  onEdit,
}: {
  ruleIndex: number | null
  doc: Record<string, unknown>
  open: boolean
  onClose: () => void
  onDocChange: (doc: Record<string, unknown>) => void
  onEdit: () => void
}) {
  const rules = wafRulesFromDoc(doc)
  const row = ruleIndex != null ? rules[ruleIndex] : null
  if (!row) return null

  const ruleID = str(row.id)
  const enabled = row.enabled !== false
  const action = actionFromRow(row)
  const actionOverride = customActionOverridden(row)
  const globalLogOnly = bool(obj(doc.waf).log_only)
  const ruleAllowHosts = wafRuleAllowHosts(doc, ruleID)
  const targets = arr<string>(row.targets).join(', ') || '—'

  return (
    <Drawer
      open={open}
      title="自定义规则详情"
      onClose={onClose}
      width={480}
      footer={(
        <>
          <button type="button" className="btn btn-ghost" onClick={onClose}>
            关闭
          </button>
          <button type="button" className="btn btn-primary" onClick={onEdit}>
            编辑
          </button>
        </>
      )}
    >
      <dl className="route-detail-dl">
        <dt>ID</dt>
        <dd><code>{ruleID || '—'}</code></dd>
        <dt>名称</dt>
        <dd>{str(row.name) || '—'}</dd>
        <dt>类型</dt>
        <dd>{str(row.type, 'regex')}</dd>
        <dt>启用</dt>
        <dd>{enabled ? '开' : '关'}</dd>
        <dt>命中后处置</dt>
        <dd>
          <RuleActionSelect
            value={action}
            disabled={!enabled}
            onChange={(act) => {
              if (ruleIndex == null) return
              onDocChange(patchCustomRuleAction(doc, ruleIndex, act))
            }}
          />
          {actionOverride ? (
            <span className="badge badge-audit" style={{ marginLeft: 8 }}>
              已单独覆盖
            </span>
          ) : (
            <span className="form-hint" style={{ display: 'block', marginTop: 4 }}>
              继承全局，当前生效：{wafInheritEffectiveLabel(globalLogOnly)}
            </span>
          )}
        </dd>
        <dt>检测目标</dt>
        <dd>{targets}</dd>
        <dt>模式</dt>
        <dd>
          <code className="waf-rule-pattern">{str(row.pattern)}</code>
        </dd>
        <dt>说明</dt>
        <dd>{wafCustomRuleDescription(row)}</dd>
        <dd className="waf-detail-allow-hosts">
          <EditableStringList
            title="域名白名单"
            showCount
            titleTooltip="精确域名、*.wildcard.example.com，或 Go 正则"
            items={ruleAllowHosts}
            valueLabel="Host 模式"
            fieldKeyName="waf.rules[].allow_hosts[]"
            emptyHint="未配置；匹配的 Host 将跳过本规则"
            placeholder="admin.internal 或 *.cdn.example.com"
            addTitle="添加域名白名单"
            editTitle="编辑域名白名单"
            onChange={(items) => onDocChange(patchBuiltinRuleAllowHosts(doc, ruleID, items))}
          />
        </dd>
      </dl>
    </Drawer>
  )
}

function BuiltinRuleDetailDrawer({
  rule,
  doc,
  open,
  onClose,
  onDocChange,
}: {
  rule: BuiltinWAFRule | null
  doc: Record<string, unknown>
  open: boolean
  onClose: () => void
  onDocChange: (doc: Record<string, unknown>) => void
}) {
  if (!rule) return null

  const enabled = isBuiltinRuleEnabled(doc, rule.id)
  const enableOverride = Object.prototype.hasOwnProperty.call(obj(obj(doc.waf).builtin_rules), rule.id)
  const action = builtinRuleAction(doc, rule.id)
  const actionOverride = builtinActionOverridden(doc, rule.id)
  const ruleAllowHosts = wafRuleAllowHosts(doc, rule.id)
  const disableAllBuiltin = !defaultBuiltinEnabled(doc)
  const globalLogOnly = bool(obj(doc.waf).log_only)

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
          {enableOverride ? (
            <span className="badge badge-audit" style={{ marginLeft: 8 }}>
              启用已单独覆盖
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
        <dt>命中后处置</dt>
        <dd>
          <RuleActionSelect
            value={action}
            disabled={!enabled}
            title={enabled ? undefined : '规则已关闭，开启后可设置处置动作'}
            onChange={(act) => onDocChange(patchBuiltinRuleAction(doc, rule.id, act))}
          />
          {actionOverride ? (
            <span className="badge badge-audit" style={{ marginLeft: 8 }}>
              已单独覆盖
            </span>
          ) : (
            <span className="form-hint" style={{ display: 'block', marginTop: 4 }}>
              继承全局，当前生效：{wafInheritEffectiveLabel(globalLogOnly)}
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
        <dd className="waf-detail-allow-hosts">
          <EditableStringList
            title="域名白名单"
            showCount
            titleTooltip="精确域名、*.wildcard.example.com，或 Go 正则"
            items={ruleAllowHosts}
            valueLabel="Host 模式"
            fieldKeyName="waf.rules[].allow_hosts[]"
            emptyHint="未配置；匹配的 Host 将跳过本规则"
            placeholder="admin.internal 或 *.cdn.example.com"
            addTitle="添加域名白名单"
            editTitle="编辑域名白名单"
            onChange={(items) => onDocChange(patchBuiltinRuleAllowHosts(doc, rule.id, items))}
          />
        </dd>
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
  const allowHostsList = wafAllowHostsList(doc)
  const disableAllBuiltin = !defaultBuiltinEnabled(doc)
  const globalLogOnly = bool(obj(doc.waf).log_only)
  const [modalOpen, setModalOpen] = useState(false)
  const [editIndex, setEditIndex] = useState<number | null>(null)
  const [draft, setDraft] = useState<WAFRuleForm>(emptyWAFRuleForm())
  const [detailRule, setDetailRule] = useState<BuiltinWAFRule | null>(null)
  const [detailCustomIndex, setDetailCustomIndex] = useState<number | null>(null)

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

  const moveRule = (index: number, delta: -1 | 1) => {
    patchRules(moveAdjacent(rules, index, delta))
  }

  const enabledBuiltinCount = WAF_BUILTIN_RULES.filter((r) => isBuiltinRuleEnabled(doc, r.id)).length

  return (
    <>
      <section className="waf-list-editor">
        <div className="waf-list-toolbar">
          <span className="waf-list-toolbar-title">
            内置规则 ({enabledBuiltinCount}/{WAF_BUILTIN_RULES.length} 启用)
          </span>
          <span className="waf-list-toolbar-hint">
            默认跟随「禁用内置规则」；启用/处置覆盖写入 builtin_rules、builtin_rule_actions；详情中可查看说明与域名白名单
          </span>
        </div>
        <table className="data config-waf-rules-table config-waf-rules-table--compact">
          <thead>
            <tr>
              <th className="col-enable">启用</th>
              <th className="col-action">处置</th>
              <th className="col-name">名称</th>
              <th className="col-actions">操作</th>
            </tr>
          </thead>
          <tbody>
            {WAF_BUILTIN_RULES.map((rule) => {
              const enabled = isBuiltinRuleEnabled(doc, rule.id)
              const enableOverride = Object.prototype.hasOwnProperty.call(obj(obj(doc.waf).builtin_rules), rule.id)
              const action = builtinRuleAction(doc, rule.id)
              return (
                <tr key={rule.id} className={enabled ? undefined : 'row-muted'}>
                  <td className="col-enable">
                    <RuleEnabledToggle
                      enabled={enabled}
                      onChange={(v) => onChange(patchBuiltinRuleEnabled(doc, rule.id, v))}
                      title={
                        enableOverride
                          ? '已单独覆盖；清除覆盖后跟随「禁用内置规则」'
                          : disableAllBuiltin
                            ? '全局已禁用内置规则；单独开启此条'
                            : '跟随全局内置规则开关'
                      }
                    />
                  </td>
                  <td className="col-action">
                    <WafActionTag
                      action={action}
                      globalLogOnly={globalLogOnly}
                      disabled={!enabled}
                      overridden={builtinActionOverridden(doc, rule.id)}
                    />
                  </td>
                  <td className="col-name">
                    <EllipsisTooltip text={rule.name} />
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
      </section>

      <EditableStringList
        title="IP 黑名单 deny"
        showCount
        items={denyList}
        valueLabel="IP / CIDR"
        fieldKeyName="waf.deny[]"
        emptyHint="暂无 IP 黑名单"
        placeholder="203.0.113.0/24"
        addTitle="添加 IP 黑名单"
        editTitle="编辑 IP 黑名单"
        hint="CIDR 或单个 IP，如 203.0.113.0/24 或 192.168.1.1"
        onChange={(items) => onChange(patchWafDeny(doc, items))}
      />

      <EditableStringList
        title="IP 白名单 allow"
        showCount
        items={allowList}
        valueLabel="IP / CIDR"
        fieldKeyName="waf.allow[]"
        emptyHint="暂无 IP 白名单"
        placeholder="10.0.0.0/8"
        addTitle="添加 IP 白名单"
        editTitle="编辑 IP 白名单"
        hint="非空时仅允许列表内网段通过 WAF 的 IP 阶段"
        onChange={(items) => onChange(patchWafAllow(doc, items))}
      />

      <EditableStringList
        title="域名白名单 allow_hosts"
        showCount
        items={allowHostsList}
        valueLabel="Host 模式"
        fieldKeyName="waf.allow_hosts[]"
        emptyHint="暂无域名白名单"
        placeholder="*.cdn.example.com"
        addTitle="添加域名白名单"
        editTitle="编辑域名白名单"
        hint="精确域名、*.wildcard.example.com，或 Go 正则（含 ( ) [ ] ^ $ 等时自动识别）"
        onChange={(items) => onChange(patchWafAllowHosts(doc, items))}
      />

      <section className="waf-list-editor">
        <div className="waf-list-toolbar">
          <span className="waf-list-toolbar-title">自定义规则 ({rules.length})</span>
          <span className="waf-list-toolbar-hint">
            按顺序评估，前者优先；regex 为 Go regexp，contains 为子串；v1 不扫描 body。名称优先显示 name，否则显示 id
          </span>
          <button type="button" className="btn btn-ghost waf-list-toolbar-add" onClick={openAdd}>
            + 添加
          </button>
        </div>
        <table className="data config-waf-rules-table config-waf-rules-table--compact">
          <thead>
            <tr>
              <th className="col-enable">启用</th>
              <th className="col-action">处置</th>
              <th className="col-name">名称</th>
              <th className="col-actions">操作</th>
            </tr>
          </thead>
          <tbody>
            {rules.length === 0 ? (
              <tr>
                <td colSpan={4} className="empty-hint">
                  暂无自定义规则
                </td>
              </tr>
            ) : (
              rules.map((rule, i) => {
                const enabled = rule.enabled !== false
                const action = actionFromRow(rule)
                const actionOverride = customActionOverridden(rule)
                return (
                  <tr key={`${str(rule.id)}-${i}`} className={enabled ? undefined : 'row-muted'}>
                    <td className="col-enable">
                      <RuleEnabledToggle
                        enabled={enabled}
                        onChange={(v) => onChange(patchCustomRuleEnabled(doc, i, v))}
                      />
                    </td>
                    <td className="col-action">
                      <WafActionTag
                        action={action}
                        globalLogOnly={globalLogOnly}
                        disabled={!enabled}
                        overridden={actionOverride}
                      />
                    </td>
                    <td className="col-name">
                      <EllipsisTooltip text={wafCustomRuleName(rule)} />
                    </td>
                    <td className="col-actions">
                      <CustomRuleRowActions
                        onView={() => setDetailCustomIndex(i)}
                        onEdit={() => openEdit(i)}
                        onDelete={() => remove(i)}
                        onMoveUp={() => moveRule(i, -1)}
                        onMoveDown={() => moveRule(i, 1)}
                        disableMoveUp={i === 0}
                        disableMoveDown={i === rules.length - 1}
                      />
                    </td>
                  </tr>
                )
              })
            )}
          </tbody>
        </table>
      </section>

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

      <CustomRuleDetailDrawer
        ruleIndex={detailCustomIndex}
        doc={doc}
        open={detailCustomIndex != null}
        onClose={() => setDetailCustomIndex(null)}
        onDocChange={onChange}
        onEdit={() => {
          if (detailCustomIndex == null) return
          const idx = detailCustomIndex
          setDetailCustomIndex(null)
          openEdit(idx)
        }}
      />

      <BuiltinRuleDetailDrawer
        rule={detailRule}
        doc={doc}
        open={detailRule != null}
        onClose={() => setDetailRule(null)}
        onDocChange={onChange}
      />
    </>
  )
}
