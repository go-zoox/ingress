import { forwardRef, useImperativeHandle, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  ConfigEntityModal,
  EntityRowActions,
  EntityTableToolbar,
} from '../ConfigEntityModal'
import { ServiceEntityFormFields } from './ServiceEntityFormFields'
import {
  emptyServiceForm,
  formToService,
  serviceSaveDisabled,
  serviceSummary,
  serviceToForm,
  servicesFromDoc,
  type ServiceForm,
} from '../../lib/services'
import { str } from '../../lib/ingressModuleForms'

export type ServicesEditorHandle = {
  openAdd: () => void
}

export const ServicesEditor = forwardRef<
  ServicesEditorHandle,
  {
    doc: Record<string, unknown>
    onChange: (doc: Record<string, unknown>) => void
    usageForName?: (name: string) => number
    onOpenDetail?: (name: string) => void
    /** 服务页隐藏 services 标签，添加按钮由外部工具栏提供 */
    hideTableChrome?: boolean
  }
>(function ServicesEditor(
  { doc, onChange, usageForName, onOpenDetail, hideTableChrome = false },
  ref,
) {
  const services = servicesFromDoc(doc)
  const [modalOpen, setModalOpen] = useState(false)
  const [editIndex, setEditIndex] = useState<number | null>(null)
  const [draft, setDraft] = useState<ServiceForm>(emptyServiceForm())

  const patchServices = (rows: Record<string, unknown>[]) => {
    onChange({ services: rows })
  }

  const openAdd = () => {
    setEditIndex(null)
    setDraft(emptyServiceForm())
    setModalOpen(true)
  }

  useImperativeHandle(ref, () => ({ openAdd }), [])

  const openEdit = (index: number) => {
    setEditIndex(index)
    setDraft(serviceToForm(services[index]))
    setModalOpen(true)
  }

  const save = () => {
    if (!draft.service_name.trim()) return
    const row = formToService(draft, editIndex == null ? undefined : services[editIndex])
    const next = [...services]
    if (editIndex == null) next.push(row)
    else next[editIndex] = row
    patchServices(next)
    setModalOpen(false)
  }

  const remove = (index: number) => {
    const name = str(services[index]?.name)
    const usage = usageForName?.(name) ?? 0
    const extra = usage > 0 ? `（${usage} 条路由仍引用此服务名）` : ''
    if (!window.confirm(`删除服务 ${name || `#${index + 1}`}？${extra}`)) return
    patchServices(services.filter((_, i) => i !== index))
  }

  return (
    <>
      {!hideTableChrome ? (
        <>
          <EntityTableToolbar label="services" onAdd={openAdd} />
          <p className="form-hint">
            可复用的上游 Service 目录；在{' '}
            <Link to="/routes">路由</Link> 编辑 backend 时可从目录选择并填充字段。
          </p>
        </>
      ) : null}
      <table className="data config-rules-table">
        <thead>
          <tr>
            <th>#</th>
            <th>名称</th>
            <th>目标</th>
            <th>健康检查</th>
            <th>备注</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          {services.length === 0 ? (
            <tr>
              <td colSpan={6} className="empty-hint">
                无服务，点击「添加」
              </td>
            </tr>
          ) : (
            services.map((row, i) => {
              const hc = row.healthcheck as Record<string, unknown> | undefined
              const hcOn = hc && typeof hc === 'object' && hc.enable === true
              const name = str(row.name)
              return (
                <tr key={`${name}-${i}`}>
                  <td>{i + 1}</td>
                  <td>
                    {onOpenDetail ? (
                      <button
                        type="button"
                        className="action-link config-host-detail-link"
                        title="查看服务详情"
                        onClick={() => onOpenDetail(name)}
                      >
                        <code>{name}</code>
                      </button>
                    ) : (
                      <code>{name}</code>
                    )}
                  </td>
                  <td>{serviceSummary(row)}</td>
                  <td>{hcOn ? '启用' : '—'}</td>
                  <td>{str(row.note) || '—'}</td>
                  <td>
                    <EntityRowActions
                      onEdit={() => openEdit(i)}
                      onDelete={() => remove(i)}
                      disableMoveUp
                      disableMoveDown
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
        title={editIndex == null ? '添加上游服务' : '编辑上游服务'}
        onClose={() => setModalOpen(false)}
        onSave={save}
        disableSave={serviceSaveDisabled(draft)}
      >
        <ServiceEntityFormFields form={draft} onChange={setDraft} />
      </ConfigEntityModal>
    </>
  )
})
