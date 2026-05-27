import {
  FormCheckbox,
  FormField,
  FormGrid,
  FormSelectField,
} from '../Form'
import { AuthFormFields } from './AuthFormFields'
import { BackendCacheFormFields } from './BackendCacheFormFields'
import { HandlerFormFields } from './HandlerFormFields'
import { HealthCheckFormFields } from './HealthCheckFormFields'
import type { BackendForm } from '../../lib/configEntities'

export function BackendFormFields<T extends BackendForm>({
  form,
  onChange,
  idPrefix = '',
  variant = 'host',
}: {
  form: T
  onChange: (next: T) => void
  idPrefix?: string
  /** path-level backends may use strip_prefix on service */
  variant?: 'host' | 'path'
}) {
  const patch = (fn: (next: T) => void) => {
    const next = { ...form }
    fn(next)
    onChange(next)
  }

  return (
    <>
      <FormSelectField
        label="Backend 类型"
        keyName={`${idPrefix}backend.type`}
        value={form.backend_type}
        onChange={(e) => patch((n) => { n.backend_type = e.target.value as BackendForm['backend_type'] })}
      >
        <option value="service">service（上游代理）</option>
        <option value="redirect">redirect（重定向）</option>
        <option value="handler">handler（直接响应）</option>
      </FormSelectField>

      {form.backend_type === 'service' && (
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

      <BackendCacheFormFields form={form} onChange={onChange} idPrefix={idPrefix} />

      {form.backend_type === 'service' && (
        <AuthFormFields form={form} onChange={onChange} idPrefix={idPrefix} />
      )}

      {form.backend_type === 'service' && (
        <HealthCheckFormFields form={form} onChange={onChange} idPrefix={idPrefix} />
      )}
    </>
  )
}

export function BackendFormGrid<T extends BackendForm>({
  form,
  onChange,
  idPrefix = '',
  variant = 'host',
}: {
  form: T
  onChange: (next: T) => void
  idPrefix?: string
  variant?: 'host' | 'path'
}) {
  return (
    <FormGrid columns={1}>
      <BackendFormFields form={form} onChange={onChange} idPrefix={idPrefix} variant={variant} />
    </FormGrid>
  )
}

export function backendFormWide(form: BackendForm): boolean {
  return form.backend_type === 'handler' || form.cache_enabled || form.auth_type !== '' || form.health_check_enable
}
