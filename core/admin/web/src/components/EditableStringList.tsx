import { useCallback, useState } from 'react'
import { createPortal } from 'react-dom'
import { FormField } from './Form'
import { ConfigEntityModal, EntityRowActions } from './ConfigEntityModal'
import { moveAdjacent } from '../lib/arrayMove'

export type EditableStringListProps = {
  /** Display title; pass with count e.g. `维护域名 (${n})` or set showCount */
  title: string
  titleTooltip?: string
  showCount?: boolean
  items: string[]
  onChange: (items: string[]) => void
  valueLabel: string
  fieldKeyName: string
  emptyHint: string
  placeholder?: string
  addTitle: string
  editTitle: string
  hint?: string
}

function ListTitleTooltip({ label, text }: { label: string; text: string }) {
  const [visible, setVisible] = useState(false)
  const [pos, setPos] = useState({ x: 0, y: 0 })

  const showAt = useCallback((el: HTMLElement) => {
    const rect = el.getBoundingClientRect()
    const popW = 320
    let x = rect.left
    const y = rect.bottom + 8
    if (x + popW > window.innerWidth - 12) {
      x = Math.max(12, window.innerWidth - popW - 12)
    }
    setPos({ x, y })
    setVisible(true)
  }, [])

  const hide = useCallback(() => setVisible(false), [])

  const pop =
    visible &&
    createPortal(
      <div
        className="waf-rule-tooltip-pop waf-rule-tooltip-pop--fixed"
        role="tooltip"
        style={{ left: pos.x, top: pos.y }}
      >
        <span className="waf-rule-tooltip-line">{text}</span>
      </div>,
      document.body,
    )

  return (
    <>
      <span
        className="waf-rule-tooltip editable-string-list-title-tooltip"
        onMouseEnter={(e) => showAt(e.currentTarget)}
        onMouseLeave={hide}
        onFocus={(e) => showAt(e.currentTarget)}
        onBlur={hide}
        tabIndex={0}
      >
        <span className="editable-string-list-title">{label}</span>
      </span>
      {pop}
    </>
  )
}

/** Generic add/edit/delete/reorder list for string values (hosts, IPs, paths, …). */
export function EditableStringList({
  title,
  titleTooltip,
  showCount = false,
  items,
  onChange,
  valueLabel,
  fieldKeyName,
  emptyHint,
  placeholder,
  addTitle,
  editTitle,
  hint,
}: EditableStringListProps) {
  const [modalOpen, setModalOpen] = useState(false)
  const [editIndex, setEditIndex] = useState<number | null>(null)
  const [draft, setDraft] = useState('')

  const displayTitle = showCount ? `${title} (${items.length})` : title

  const openAdd = () => {
    setEditIndex(null)
    setDraft('')
    setModalOpen(true)
  }

  const openEdit = (index: number) => {
    setEditIndex(index)
    setDraft(items[index] ?? '')
    setModalOpen(true)
  }

  const save = () => {
    const value = draft.trim()
    if (!value) return
    const next = [...items]
    if (editIndex == null) next.push(value)
    else next[editIndex] = value
    onChange(next)
    setModalOpen(false)
  }

  const remove = (index: number) => {
    const value = items[index] ?? ''
    if (!window.confirm(`删除 ${value || `#${index + 1}`}？`)) return
    onChange(items.filter((_, i) => i !== index))
  }

  const moveItem = (index: number, delta: -1 | 1) => {
    onChange(moveAdjacent(items, index, delta))
  }

  return (
    <>
      <section className="editable-string-list">
        <div className="editable-string-list-toolbar">
          {titleTooltip ? (
            <ListTitleTooltip label={displayTitle} text={titleTooltip} />
          ) : (
            <span className="editable-string-list-title">{displayTitle}</span>
          )}
          <button type="button" className="btn btn-ghost editable-string-list-add" onClick={openAdd}>
            + 添加
          </button>
        </div>

        {items.length === 0 ? (
          <p className="editable-string-list-empty">{emptyHint}</p>
        ) : (
          <div className="editable-string-list-panel">
            <div className="editable-string-list-row editable-string-list-row--head">
              <span className="editable-string-list-value">{valueLabel}</span>
              <span className="editable-string-list-actions-head">操作</span>
            </div>
            {items.map((item, i) => (
              <div key={`${item}-${i}`} className="editable-string-list-row">
                <code className="editable-string-list-value">{item}</code>
                <div className="editable-string-list-actions">
                  <EntityRowActions
                    onEdit={() => openEdit(i)}
                    onDelete={() => remove(i)}
                    onMoveUp={() => moveItem(i, -1)}
                    onMoveDown={() => moveItem(i, 1)}
                    disableMoveUp={i === 0}
                    disableMoveDown={i === items.length - 1}
                  />
                </div>
              </div>
            ))}
          </div>
        )}
      </section>

      <ConfigEntityModal
        open={modalOpen}
        title={editIndex == null ? addTitle : editTitle}
        onClose={() => setModalOpen(false)}
        onSave={save}
        disableSave={!draft.trim()}
      >
        <FormField
          label={valueLabel}
          keyName={fieldKeyName}
          hint={hint}
          placeholder={placeholder}
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
        />
      </ConfigEntityModal>
    </>
  )
}
