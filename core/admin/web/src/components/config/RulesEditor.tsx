import { forwardRef, useEffect, useImperativeHandle, useRef, useState } from 'react'
import { GripVertical } from 'lucide-react'
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
  cloneRuleRow,
  copyHostValidation,
  copyHostWarning,
  emptyRuleForm,
  formToRule,
  insertCopiedRule,
  ruleIPAllowlistLabel,
  ruleSaveDisabled,
  ruleSummary,
  ruleToForm,
  rulesFromDoc,
  suggestCopyHost,
  type CopyHostSuggestion,
  type HostType,
  type RuleForm,
} from '../../lib/configEntities'
import { trustedProxiesFromDoc } from '../../lib/ipAllowlist'
import { arr, str } from '../../lib/ingressModuleForms'
import { moveToIndex, remapIndexAfterMove } from '../../lib/arrayMove'
import { useListDragReorder } from '../../hooks/useListDragReorder'
import type { ServiceForm } from '../../lib/services'

export type RulesEditorHandle = {
  openAdd: () => void
  /** Open the duplicate-route modal for the rule at `index` (used by ?copy=<ri> deep links). */
  openCopy: (index: number) => void
}

type CopySource = {
  index: number
  host: string
  suggestion: CopyHostSuggestion
}

type CopyNotice = {
  message: string
  snapshot: Record<string, unknown>[]
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
  const trustedProxies = trustedProxiesFromDoc(doc)
  const [modalOpen, setModalOpen] = useState(false)
  const [pathsModalIndex, setPathsModalIndex] = useState<number | null>(null)
  const [editIndex, setEditIndex] = useState<number | null>(null)
  const [draft, setDraft] = useState<RuleForm>(emptyRuleForm())
  const [activeSection, setActiveSection] = useState<RuleEntitySectionId>('basic')
  const [copySource, setCopySource] = useState<CopySource | null>(null)
  const [copyPosition, setCopyPosition] = useState<'after' | 'end'>('after')
  const [copyNotice, setCopyNotice] = useState<CopyNotice | null>(null)
  const [highlightIndex, setHighlightIndex] = useState<number | null>(null)
  const noticeTimerRef = useRef<number | null>(null)

  const patchRules = (rows: Record<string, unknown>[]) => {
    onChange({ rules: rows })
  }

  const resetCopyState = () => {
    setCopySource(null)
    setCopyPosition('after')
  }

  const openAdd = () => {
    resetCopyState()
    setEditIndex(null)
    setDraft(emptyRuleForm())
    setActiveSection(defaultRuleEntitySection('rule'))
    setModalOpen(true)
  }

  const openCopy = (index: number) => {
    const source = rules[index]
    if (!source) return
    const sourceHost = str(source.host)
    const hostType = str(source.host_type, 'auto') as HostType
    const suggestion = suggestCopyHost(
      sourceHost,
      rules.map((rule) => str(rule.host)),
      hostType,
    )
    const form = ruleToForm(source)
    form.host = suggestion.host
    setEditIndex(null)
    setCopyPosition('after')
    setCopySource({ index, host: sourceHost, suggestion })
    setDraft(form)
    setActiveSection(defaultRuleEntitySection('rule'))
    setModalOpen(true)
  }

  useImperativeHandle(ref, () => ({ openAdd, openCopy }))

  const closeModal = () => {
    setModalOpen(false)
    resetCopyState()
  }

  const openEdit = (index: number) => {
    resetCopyState()
    setEditIndex(index)
    setDraft(ruleToForm(rules[index]))
    setActiveSection(defaultRuleEntitySection('rule'))
    setModalOpen(true)
  }

  const openPaths = (index: number) => {
    setPathsModalIndex(index)
  }

  useEffect(() => {
    if (highlightIndex == null) return
    const timer = window.setTimeout(() => setHighlightIndex(null), 3200)
    return () => window.clearTimeout(timer)
  }, [highlightIndex])

  useEffect(
    () => () => {
      if (noticeTimerRef.current != null) window.clearTimeout(noticeTimerRef.current)
    },
    [],
  )

  const showCopyNotice = (
    host: string,
    index: number,
    snapshot: Record<string, unknown>[],
  ) => {
    if (noticeTimerRef.current != null) window.clearTimeout(noticeTimerRef.current)
    setCopyNotice({ message: `已创建副本 ${host}（第 ${index + 1} 条）`, snapshot })
    setHighlightIndex(index)
    noticeTimerRef.current = window.setTimeout(() => setCopyNotice(null), 8000)
  }

  const undoCopy = () => {
    if (!copyNotice) return
    if (noticeTimerRef.current != null) window.clearTimeout(noticeTimerRef.current)
    patchRules(copyNotice.snapshot)
    setCopyNotice(null)
    setHighlightIndex(null)
  }

  const copyHostError = copySource ? copyHostValidation(draft.host, rules) : null
  const copyHostWarningMessage = copySource
    ? copyHostWarning(draft.host, (draft.host_type || 'auto') as HostType)
    : null

  const save = () => {
    if (!draft.host.trim()) return
    if (copySource) {
      if (copyHostError) return
      const source = rules[copySource.index]
      const row = formToRule(draft, source ? cloneRuleRow(source) : undefined)
      const result = insertCopiedRule(rules, copySource.index, copyPosition, row)
      patchRules(result.rules)
      showCopyNotice(str(row.host), result.index, rules)
      setModalOpen(false)
      resetCopyState()
      return
    }
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

  const moveRuleByIndex = (from: number, to: number) => {
    patchRules(moveToIndex(rules, from, to))
    setPathsModalIndex((cur) => remapIndexAfterMove(cur, from, to))
  }

  const moveRule = (index: number, delta: -1 | 1) => {
    const j = index + delta
    if (j < 0 || j >= rules.length) return
    moveRuleByIndex(index, j)
  }

  const { rowClassName, handleProps, rowProps } = useListDragReorder({
    items: rules,
    onMove: moveRuleByIndex,
  })

  const savePaths = (nextRule: Record<string, unknown>) => {
    if (pathsModalIndex == null) return
    const next = [...rules]
    next[pathsModalIndex] = nextRule
    patchRules(next)
  }

  const pathsRule = pathsModalIndex == null ? null : rules[pathsModalIndex]

  const copyRowClass = (index: number, extra?: string) => {
    const parts: string[] = []
    if (extra) parts.push(extra)
    if (highlightIndex === index) parts.push('row-flash')
    return parts.length > 0 ? parts.join(' ') : undefined
  }

  const copySuggestionNote = (() => {
    if (!copySource) return null
    const suggestion = copySource.suggestion
    if (!suggestion.auto) {
      return (
        <p className="copy-route-note">
          {copyHostWarningMessage ?? '正则 Host 不会自动改名；请确认副本 Host'}
        </p>
      )
    }
    if (draft.host.trim() !== suggestion.host) return null
    if (suggestion.collisionAvoided) {
      return (
        <p className="copy-route-hint">
          {suggestion.firstSuggestion} 已被现有规则占用，已自动顺延为{' '}
          <code>{suggestion.host}</code>
        </p>
      )
    }
    return <p className="copy-route-hint">已按原 Host 生成区分名，可直接修改</p>
  })()

  return (
    <>
      {!hideTableChrome ? (
        <>
          <EntityTableToolbar label="rules" onAdd={openAdd} />
          <p className="form-hint">列表顺序即匹配优先级，排在前面的规则优先匹配；可拖拽左侧手柄调整顺序。</p>
        </>
      ) : null}
      <table className="data config-rules-table">
        <thead>
          <tr>
            <th className="col-drag" aria-label="排序" />
            <th>#</th>
            <th>状态</th>
            <th>Host</th>
            <th>类型</th>
            <th>Backend</th>
            <th>IP 白名单</th>
            <th>Paths</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          {rules.length === 0 ? (
            <tr>
              <td colSpan={9} className="empty-hint">
                无路由规则，点击「添加」
              </td>
            </tr>
          ) : (
            rules.map((rule, i) => {
              const pathCount = arr(rule.paths).length
              const enabled = rule.enabled !== false
              return (
                <tr
                  key={`${str(rule.host)}-${i}`}
                  className={rowClassName(i, copyRowClass(i, enabled ? undefined : 'row-muted'))}
                  {...rowProps(i)}
                >
                  <td className="col-drag">
                    <span
                      className="row-drag-handle"
                      title="拖拽排序"
                      aria-label={`拖拽调整规则 ${i + 1} 顺序`}
                      {...handleProps(i)}
                    >
                      <GripVertical size={14} aria-hidden />
                    </span>
                  </td>
                  <td>{i + 1}</td>
                  <td>
                    <span className={`badge ${enabled ? 'badge-exact' : 'badge-block'}`}>
                      {enabled ? '启用' : '禁用'}
                    </span>
                  </td>
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
                    {(() => {
                      const label = ruleIPAllowlistLabel(rule)
                      return label ? (
                        <span className="badge badge-wildcard" title="IP 白名单（WAF IP 阶段）">
                          {label}
                        </span>
                      ) : (
                        <span className="badge badge-audit">未限制</span>
                      )
                    })()}
                  </td>
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
                      leadingMenuItems={[{ label: '复制路由', onClick: () => openCopy(i) }]}
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
        title={copySource ? '复制路由' : editIndex == null ? '添加路由规则' : '编辑路由规则'}
        wide
        onClose={closeModal}
        onSave={save}
        saveLabel={copySource ? '创建副本' : '保存'}
        disableSave={ruleSaveDisabled(draft) || Boolean(copyHostError)}
      >
        {copySource ? (
          <div className="copy-route-banner">
            <div className="copy-route-banner-head">
              <span className="copy-route-badge">副本</span>
              <span className="copy-route-source">
                源规则 #{copySource.index + 1} · <code>{copySource.host}</code>
              </span>
            </div>
            <p className="copy-route-desc">
              除 Host 外全部配置（backend / paths / WAF / 限流 / 缓存 / 安全 / 正向代理）已复制；副本与源规则互相独立。
            </p>
            <div className="copy-route-options">
              <label className="copy-route-option">
                <span>插入位置</span>
                <select
                  className="form-control"
                  value={copyPosition}
                  onChange={(event) => setCopyPosition(event.target.value as 'after' | 'end')}
                >
                  <option value="after">紧随规则 #{copySource.index + 1} 之后（推荐）</option>
                  <option value="end">列表末尾</option>
                </select>
              </label>
              {copySuggestionNote}
            </div>
            {copyHostError ? <p className="copy-route-error">{copyHostError}</p> : null}
          </div>
        ) : null}
        <RuleEntityFormSections
          form={draft}
          onChange={setDraft}
          activeSection={activeSection}
          onSectionChange={setActiveSection}
          variant="rule"
          serviceCatalog={serviceCatalog}
          serviceFieldMode={serviceFieldMode}
          trustedProxies={trustedProxies}
          hostError={copyHostError}
        />
      </ConfigEntityModal>

      {copyNotice ? (
        <div className="copy-undo-bar" role="status">
          <span>{copyNotice.message}</span>
          <span className="copy-undo-hint">未保存 · 发布后才会生效</span>
          <button type="button" className="action-link copy-undo-action" onClick={undoCopy}>
            撤销
          </button>
          <button
            type="button"
            className="copy-undo-close"
            aria-label="关闭提示"
            onClick={() => setCopyNotice(null)}
          >
            ×
          </button>
        </div>
      ) : null}

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
