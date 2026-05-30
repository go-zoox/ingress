import {
  FormCheckbox,
  FormField,
  FormSelectField,
} from '../Form'
import { HandlerFormFields } from './HandlerFormFields'
import type { BackendForm } from '../../lib/configEntities'
import { applyServiceToBackend, type ServiceForm } from '../../lib/services'

export function BackendCoreFormFields<T extends BackendForm>({
  form,
  onChange,
  idPrefix = '',
  variant = 'host',
  serviceCatalog,
  serviceFieldMode = 'manual',
}: {
  form: T
  onChange: (next: T) => void
  idPrefix?: string
  variant?: 'host' | 'path'
  serviceCatalog?: ServiceForm[]
  /** catalog-select: 仅选服务目录（路由页）；manual: 完整字段（配置中心） */
  serviceFieldMode?: 'manual' | 'catalog-select'
}) {
  const patch = (fn: (next: T) => void) => {
    const next = { ...form }
    fn(next)
    onChange(next)
  }

  const catalog = serviceCatalog?.filter((s) => s.service_name.trim()) ?? []
  const catalogOnly = serviceFieldMode === 'catalog-select' && catalog.length > 0
  const catalogValue =
    catalog.some((s) => s.service_name === form.service_name) ? form.service_name : ''

  const servicePicker = catalog.length > 0 ? (
    <FormSelectField
      label={catalogOnly ? '上游服务' : '从服务目录选择'}
      keyName={`${idPrefix}service.catalog`}
      hint={
        catalogOnly
          ? '在「服务」页维护 name / port / protocol；此处仅选择引用'
          : '选用后填充下方字段；仍可在本路由内微调'
      }
      value={catalogValue}
      onChange={(e) => {
        const name = e.target.value
        if (!name) return
        const svc = catalog.find((s) => s.service_name === name)
        if (svc) onChange(applyServiceToBackend(svc, form) as T)
      }}
    >
      {!catalogOnly ? <option value="">— 手动填写 —</option> : null}
      {catalogOnly && !catalogValue ? <option value="">— 请选择 —</option> : null}
      {catalog.map((s) => (
        <option key={s.service_name} value={s.service_name}>
          {s.service_name} ({s.service_protocol || 'http'}:{s.service_port})
        </option>
      ))}
    </FormSelectField>
  ) : serviceFieldMode === 'catalog-select' ? (
    <p className="form-hint">
      暂无服务目录，请先在 <a href="/services">服务</a> 页添加上游 Service。
    </p>
  ) : null

  return (
    <>
      {form.backend_type === 'service' && (
        <>
          {servicePicker}
          {!catalogOnly ? (
            <>
              <FormField
                label="上游服务名"
                keyName={`${idPrefix}service.name`}
                value={form.service_name}
                onChange={(e) => patch((n) => { n.service_name = e.target.value })}
              />
              <FormField
                label="上游端口"
                keyName={`${idPrefix}service.port`}
                type="number"
                value={form.service_port}
                onChange={(e) => patch((n) => { n.service_port = Number(e.target.value) })}
              />
              <FormSelectField
                label="协议"
                keyName={`${idPrefix}service.protocol`}
                value={form.service_protocol}
                onChange={(e) => patch((n) => { n.service_protocol = e.target.value })}
              >
                <option value="http">http</option>
                <option value="https">https</option>
              </FormSelectField>
              <FormSelectField
                label="Host 模式 service.mode"
                keyName={`${idPrefix}service.mode`}
                value={form.service_mode}
                onChange={(e) => patch((n) => { n.service_mode = e.target.value })}
              >
                <option value="">默认（internal）</option>
                <option value="internal">internal（保留客户端 Host）</option>
                <option value="external">external（Host 对齐上游名）</option>
              </FormSelectField>
            </>
          ) : null}
          {variant === 'path' && (
            <FormCheckbox
              label="strip_prefix（去掉 path 前缀再转发）"
              checked={form.service_strip_prefix}
              onChange={(v) => patch((n) => { n.service_strip_prefix = v })}
            />
          )}
        </>
      )}

      {form.backend_type === 'redirect' && (
        <>
          <FormField
            label="重定向 URL"
            keyName={`${idPrefix}redirect.url`}
            value={form.redirect_url}
            onChange={(e) => patch((n) => { n.redirect_url = e.target.value })}
          />
          <FormCheckbox
            label="永久重定向 (301/308)（默认 302/307）"
            checked={form.redirect_permanent}
            onChange={(v) => patch((n) => { n.redirect_permanent = v })}
          />
        </>
      )}

      {form.backend_type === 'handler' && (
        <HandlerFormFields form={form} onChange={onChange} />
      )}
    </>
  )
}
