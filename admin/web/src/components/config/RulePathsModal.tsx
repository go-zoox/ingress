import { useEffect, useState } from 'react'
import {
  FormField,
  FormGrid,
} from '../Form'
import {
  ConfigEntityModal,
  EntityRowActions,
  EntityTableToolbar,
} from '../ConfigEntityModal'
import { BackendFormGrid, backendFormWide } from './BackendFormFields'
import {
  applyPathsToRule,
  emptyPathForm,
  formToPath,
  pathSaveDisabled,
  pathSummary,
  pathsFromRule,
  type PathForm,
} from '../../lib/configEntities'

function PathFormFields({
  form,
  onChange,
}: {
  form: PathForm
  onChange: (next: PathForm) => void
}) {
  const patch = (fn: (next: PathForm) => void) => {
    const next = { ...form }
    fn(next)
    onChange(next)
  }

  return (
    <FormGrid columns={1}>
      <FormField
        label="Path 前缀"
        keyName="path"
        hint="如 /api、/v2；匹配最长前缀"
        value={form.path}
        onChange={(e) => patch((n) => { n.path = e.target.value })}
      />
      <BackendFormGrid<PathForm>
        form={form}
        onChange={onChange}
        idPrefix="paths[]."
        variant="path"
      />
    </FormGrid>
  )
}

export function RulePathsModal({
  open,
  host,
  rule,
  onClose,
  onSave,
}: {
  open: boolean
  host: string
  rule: Record<string, unknown>
  onClose: () => void
  onSave: (nextRule: Record<string, unknown>) => void
}) {
  const [paths, setPaths] = useState<PathForm[]>([])
  const [pathModalOpen, setPathModalOpen] = useState(false)
  const [editIndex, setEditIndex] = useState<number | null>(null)
  const [draft, setDraft] = useState<PathForm>(emptyPathForm())

  useEffect(() => {
    if (open) setPaths(pathsFromRule(rule))
  }, [open, rule])

  if (!open) return null

  const openAdd = () => {
    setEditIndex(null)
    setDraft(emptyPathForm())
    setPathModalOpen(true)
  }

  const openEdit = (index: number) => {
    setEditIndex(index)
    setDraft({ ...paths[index] })
    setPathModalOpen(true)
  }

  const savePath = () => {
    if (pathSaveDisabled(draft)) return
    const next = [...paths]
    if (editIndex == null) next.push({ ...draft })
    else next[editIndex] = { ...draft }
    setPaths(next)
    setPathModalOpen(false)
  }

  const removePath = (index: number) => {
    const label = paths[index]?.path || `#${index + 1}`
    if (!window.confirm(`删除 path ${label}？`)) return
    setPaths(paths.filter((_, i) => i !== index))
  }

  const saveAll = () => {
    onSave(applyPathsToRule(rule, paths))
    onClose()
  }

  const origPaths = pathsFromRule(rule)

  return (
    <>
      <div className="modal-overlay open" onClick={(e) => e.target === e.currentTarget && onClose()}>
        <div className="modal config-entity-modal config-entity-modal--wide config-paths-modal" role="dialog">
          <header>
            <h2>Path 配置</h2>
            <p className="config-paths-host">
              Host: <code>{host || '—'}</code>
            </p>
          </header>
          <div className="content">
            <EntityTableToolbar label="rules[].paths" onAdd={openAdd} />
            <table className="data config-paths-table">
              <thead>
                <tr>
                  <th>#</th>
                  <th>Path</th>
                  <th>Backend</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                {paths.length === 0 ? (
                  <tr>
                    <td colSpan={4} className="empty-hint">
                      无 path 规则，点击「添加」；未配置时使用 Host 级 backend
                    </td>
                  </tr>
                ) : (
                  paths.map((row, i) => (
                    <tr key={`${row.path}-${i}`}>
                      <td>{i + 1}</td>
                      <td><code>{row.path}</code></td>
                      <td>{pathSummary(formToPath(row, origPaths[i]))}</td>
                      <td>
                        <EntityRowActions onEdit={() => openEdit(i)} onDelete={() => removePath(i)} />
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
          <footer>
            <button type="button" className="btn" onClick={onClose}>
              取消
            </button>
            <button type="button" className="btn btn-primary" onClick={saveAll}>
              保存 Paths
            </button>
          </footer>
        </div>
      </div>

      <ConfigEntityModal
        open={pathModalOpen}
        title={editIndex == null ? '添加 Path' : '编辑 Path'}
        wide={backendFormWide(draft)}
        onClose={() => setPathModalOpen(false)}
        onSave={savePath}
        disableSave={pathSaveDisabled(draft)}
      >
        <PathFormFields form={draft} onChange={setDraft} />
      </ConfigEntityModal>
    </>
  )
}
