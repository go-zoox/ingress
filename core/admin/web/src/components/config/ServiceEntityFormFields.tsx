import {
  FormField,
  FormGrid,
  FormSection,
  FormSelectField,
} from '../Form'
import { HealthCheckFormFields } from './HealthCheckFormFields'
import type { BackendForm } from '../../lib/configEntities'
import type { ServiceForm } from '../../lib/services'

export function ServiceEntityFormFields({
  form,
  onChange,
}: {
  form: ServiceForm
  onChange: (next: ServiceForm) => void
}) {
  const patch = (fn: (next: ServiceForm) => void) => {
    const next = { ...form }
    fn(next)
    onChange(next)
  }

  return (
    <FormGrid columns={1}>
      <FormSection title="上游 Service">
        <FormField
          label="服务名 name"
          keyName="service.name"
          hint="唯一标识；路由 backend.service.name 引用此名称"
          value={form.service_name}
          onChange={(e) => patch((n) => { n.service_name = e.target.value })}
        />
        <FormField
          label="端口 port"
          keyName="service.port"
          type="number"
          value={form.service_port}
          onChange={(e) => patch((n) => { n.service_port = Number(e.target.value) })}
        />
        <FormSelectField
          label="协议 protocol"
          keyName="service.protocol"
          value={form.service_protocol}
          onChange={(e) => patch((n) => { n.service_protocol = e.target.value })}
        >
          <option value="http">http</option>
          <option value="https">https</option>
        </FormSelectField>
        <FormSelectField
          label="Host 模式 mode"
          keyName="service.mode"
          value={form.service_mode}
          onChange={(e) => patch((n) => { n.service_mode = e.target.value })}
        >
          <option value="">默认（internal）</option>
          <option value="internal">internal（保留客户端 Host）</option>
          <option value="external">external（Host 对齐上游名）</option>
        </FormSelectField>
        <FormField
          label="备注 note"
          keyName="service.note"
          hint="仅 Admin 目录展示，不写入路由 backend"
          value={form.note}
          onChange={(e) => patch((n) => { n.note = e.target.value })}
        />
      </FormSection>
      <HealthCheckFormFields
        form={form as unknown as BackendForm}
        onChange={(next) =>
          onChange({
            ...form,
            health_check_enable: next.health_check_enable,
            health_check_method: next.health_check_method,
            health_check_path: next.health_check_path,
            health_check_status: next.health_check_status,
            health_check_ok: next.health_check_ok,
          })
        }
        idPrefix="catalog-"
        embedded
      />
    </FormGrid>
  )
}
