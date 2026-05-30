import { Drawer } from '../Drawer'
import { FormField, FormGrid } from '../Form'
import { ScenarioOverlayEditor } from './ScenarioOverlayEditor'
import { scenarioIdFromLabel, type ScenarioItemForm } from '../../lib/scenarios'

type Props = {
  open: boolean
  form: ScenarioItemForm | null
  isNew: boolean
  existingIds: string[]
  hostOptions: string[]
  onChange: (next: ScenarioItemForm) => void
  onClose: () => void
  onSave: () => void
  saving?: boolean
}

export function ScenarioItemDrawer({
  open,
  form,
  isNew,
  existingIds,
  hostOptions,
  onChange,
  onClose,
  onSave,
  saving,
}: Props) {
  if (!form) return null

  const patch = (fn: (n: ScenarioItemForm) => void) => {
    const next = { ...form }
    fn(next)
    onChange(next)
  }

  const idPreview = isNew ? scenarioIdFromLabel(form.label, existingIds) : form.id

  return (
    <Drawer
      open={open}
      title={isNew ? '新建场景' : `编辑场景 · ${form.label || form.id}`}
      width={760}
      onClose={onClose}
      footer={
        <>
          <button type="button" className="btn btn-ghost" onClick={onClose}>
            取消
          </button>
          <button type="button" className="btn btn-primary" disabled={saving} onClick={onSave}>
            确定
          </button>
        </>
      }
    >
      <FormGrid columns={1}>
        <FormField
          label="显示名称"
          hint="保存后自动生成场景 ID（英文名称会转为小写连字符，如 live → live；中文等会分配 scenario-N）"
          value={form.label}
          onChange={(e) => patch((n) => { n.label = e.target.value })}
        />
        <p className="form-hint">
          场景 ID：<code>{idPreview || '—'}</code>
          {isNew ? '（自动生成）' : '（不可修改）'}
        </p>
        <FormField
          label="说明"
          value={form.description}
          onChange={(e) => patch((n) => { n.description = e.target.value })}
        />
      </FormGrid>
      <ScenarioOverlayEditor form={form} onChange={onChange} hostOptions={hostOptions} />
    </Drawer>
  )
}
