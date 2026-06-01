import { useState } from 'react'
import { FormField, FormGrid } from '../Form'
import { DateTimeRangeField, formatRFC3339Display } from '../DateTimeRangeField'
import { ConfigEntityModal, EntityRowActions } from '../ConfigEntityModal'
import { moveAdjacent } from '../../lib/arrayMove'
import { isDateTimeRangeOrdered } from '../../lib/datetimeRange'
import type { MaintenanceHostFormEntry } from '../../lib/maintenance'

export function emptyMaintenanceHostEntry(): MaintenanceHostFormEntry {
  return { host: '', window_start: '', window_end: '' }
}

export function MaintenanceHostListEditor({
  title,
  titleTooltip,
  showCount,
  items,
  onChange,
  fieldKeyPrefix,
  emptyHint,
  addTitle,
  editTitle,
}: {
  title: string
  titleTooltip?: string
  showCount?: boolean
  items: MaintenanceHostFormEntry[]
  onChange: (items: MaintenanceHostFormEntry[]) => void
  fieldKeyPrefix: string
  emptyHint: string
  addTitle: string
  editTitle: string
}) {
  const displayTitle = showCount ? `${title} (${items.length})` : title
  const [modalOpen, setModalOpen] = useState(false)
  const [editIndex, setEditIndex] = useState<number | null>(null)
  const [draft, setDraft] = useState<MaintenanceHostFormEntry>(emptyMaintenanceHostEntry())

  const openAdd = () => {
    setEditIndex(null)
    setDraft(emptyMaintenanceHostEntry())
    setModalOpen(true)
  }

  const openEdit = (index: number) => {
    setEditIndex(index)
    setDraft({ ...items[index] })
    setModalOpen(true)
  }

  const save = () => {
    const host = draft.host.trim()
    if (!host) return
    if (!draft.window_start.trim() || !draft.window_end.trim()) return
    if (!isDateTimeRangeOrdered(draft.window_start, draft.window_end)) return
    const next = [...items]
    const row: MaintenanceHostFormEntry = {
      host,
      window_start: draft.window_start.trim(),
      window_end: draft.window_end.trim(),
    }
    if (editIndex == null) next.push(row)
    else next[editIndex] = row
    onChange(next)
    setModalOpen(false)
  }

  const remove = (index: number) => {
    const value = items[index]?.host ?? ''
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
            <span className="editable-string-list-title" title={titleTooltip}>{displayTitle}</span>
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
          <div className="editable-string-list-panel maintenance-host-list-panel">
            <div className="editable-string-list-row editable-string-list-row--head maintenance-host-list-head">
              <span>Host 模式</span>
              <span>开始</span>
              <span>结束</span>
              <span className="editable-string-list-actions-head">操作</span>
            </div>
            {items.map((item, i) => (
              <div key={`${item.host}-${i}`} className="editable-string-list-row maintenance-host-list-row">
                <code className="editable-string-list-value">{item.host}</code>
                <span className="maintenance-host-window-cell">{formatRFC3339Display(item.window_start)}</span>
                <span className="maintenance-host-window-cell">{formatRFC3339Display(item.window_end)}</span>
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
        disableSave={
          !draft.host.trim() ||
          !draft.window_start.trim() ||
          !draft.window_end.trim() ||
          !isDateTimeRangeOrdered(draft.window_start, draft.window_end)
        }
      >
        <FormGrid columns={1}>
          <FormField
            label="Host 模式"
            keyName={`${fieldKeyPrefix}.host`}
            placeholder="app.example.com 或 staging-*.example.com"
            value={draft.host}
            onChange={(e) => setDraft((d) => ({ ...d, host: e.target.value }))}
          />
          <DateTimeRangeField
            label="维护时间 window"
            hint="必填；维护开始与结束时间（RFC3339）"
            start={draft.window_start}
            end={draft.window_end}
            showDisplayHint
            onChange={({ start, end }) => setDraft((d) => ({ ...d, window_start: start, window_end: end }))}
          />
        </FormGrid>
      </ConfigEntityModal>
    </>
  )
}
