import {
  FormCheckbox,
  FormField,
  FormGrid,
  FormSection,
  FormSelectField,
} from '../Form'
import {
  fallbackToForm,
  formToFallback,
  type FallbackForm,
} from '../../lib/configEntities'

export function FallbackEditor({
  doc,
  onChange,
}: {
  doc: Record<string, unknown>
  onChange: (doc: Record<string, unknown>) => void
}) {
  const form = fallbackToForm(doc)

  const patch = (fn: (next: FallbackForm) => void) => {
    const next = { ...form }
    fn(next)
    onChange(formToFallback(next))
  }

  return (
    <FormGrid columns={1}>
      <p className="form-hint form-item--full">
        全局 fallback：当 host 未匹配任何 rules 时使用（对应 @@fallback）。
      </p>
      <FormSelectField
        label="Backend 类型"
        keyName="fallback.type"
        value={form.backend_type}
        onChange={(e) => patch((n) => { n.backend_type = e.target.value as FallbackForm['backend_type'] })}
      >
        <option value="service">service（上游代理）</option>
        <option value="redirect">redirect（重定向）</option>
      </FormSelectField>

      {form.backend_type === 'service' ? (
        <FormSection title="上游服务">
          <FormField
            label="服务名"
            keyName="service.name"
            value={form.service_name}
            onChange={(e) => patch((n) => { n.service_name = e.target.value })}
          />
          <FormField
            label="端口"
            keyName="service.port"
            type="number"
            value={form.service_port}
            onChange={(e) => patch((n) => { n.service_port = Number(e.target.value) })}
          />
          <FormSelectField
            label="协议"
            keyName="service.protocol"
            value={form.service_protocol}
            onChange={(e) => patch((n) => { n.service_protocol = e.target.value })}
          >
            <option value="http">http</option>
            <option value="https">https</option>
          </FormSelectField>
        </FormSection>
      ) : (
        <FormSection title="重定向">
          <FormField
            label="重定向 URL"
            keyName="redirect.url"
            value={form.redirect_url}
            onChange={(e) => patch((n) => { n.redirect_url = e.target.value })}
          />
          <FormCheckbox
            label="永久重定向 (301/308)"
            checked={form.redirect_permanent}
            onChange={(v) => patch((n) => { n.redirect_permanent = v })}
          />
        </FormSection>
      )}
    </FormGrid>
  )
}
