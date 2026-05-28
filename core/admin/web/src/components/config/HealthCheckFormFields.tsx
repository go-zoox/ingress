import {
  FormCheckbox,
  FormField,
  FormSection,
  FormSelectField,
} from '../Form'
import type { BackendForm } from '../../lib/configEntities'

export function HealthCheckFormFields<T extends BackendForm>({
  form,
  onChange,
  idPrefix = '',
  embedded = false,
}: {
  form: T
  onChange: (next: T) => void
  idPrefix?: string
  embedded?: boolean
}) {
  const patch = (fn: (next: T) => void) => {
    const next = { ...form }
    fn(next)
    onChange(next)
  }

  return (
    <FormSection title={embedded ? undefined : '健康检查 backend.service.healthcheck'}>
      <FormCheckbox
        label="启用健康检查 healthcheck.enable"
        checked={form.health_check_enable}
        onChange={(v) => patch((n) => { n.health_check_enable = v })}
      />

      {form.health_check_enable && (
        <>
          <FormSelectField
            label="请求方法 healthcheck.method"
            keyName={`${idPrefix}service.healthcheck.method`}
            hint="默认 GET"
            value={form.health_check_method || 'GET'}
            onChange={(e) => patch((n) => { n.health_check_method = e.target.value })}
          >
            <option value="GET">GET</option>
            <option value="POST">POST</option>
            <option value="PUT">PUT</option>
            <option value="HEAD">HEAD</option>
          </FormSelectField>

          <FormField
            label="检查路径 healthcheck.path"
            keyName={`${idPrefix}service.healthcheck.path`}
            hint="默认 /health"
            value={form.health_check_path}
            placeholder="/health"
            onChange={(e) => patch((n) => { n.health_check_path = e.target.value })}
          />

          <FormField
            label="期望状态码 healthcheck.status"
            keyName={`${idPrefix}service.healthcheck.status`}
            hint="逗号分隔，默认 200"
            value={form.health_check_status}
            placeholder="200"
            onChange={(e) => patch((n) => { n.health_check_status = e.target.value })}
          />

          <FormCheckbox
            label="始终标记为健康 healthcheck.ok"
            checked={form.health_check_ok}
            onChange={(v) => patch((n) => { n.health_check_ok = v })}
          />
          {form.health_check_ok && (
            <p className="form-hint">
              启用后不执行实际检查，始终返回健康状态
            </p>
          )}
        </>
      )}
    </FormSection>
  )
}
