import { FormCheckbox, FormSection } from '../Form'
import { SecurityFormFields } from './SecurityFormFields'
import type { SecurityLayerFormSlice } from '../../lib/configEntities'
import { emptySecurityForm } from '../../lib/configEntities'

export function RouteSecurityFormFields<T extends SecurityLayerFormSlice>({
  form,
  onChange,
  layer,
  embedded = false,
}: {
  form: T
  onChange: (next: T) => void
  layer: 'rule' | 'path'
  embedded?: boolean
}) {
  const patch = (fn: (next: T) => void) => {
    const next = { ...form }
    fn(next)
    onChange(next)
  }

  const inheritLabel =
    layer === 'path'
      ? '继承 Host / 全局安全配置（不写入 paths[].security）'
      : '继承全局 security:（不写入 rules[].security）'

  const title =
    layer === 'path'
      ? 'Path 安全 paths[].security'
      : 'Host 安全 rules[].security'

  return (
    <FormSection title={embedded ? undefined : title}>
      <FormCheckbox
        label={inheritLabel}
        checked={!form.security_override}
        onChange={(inherit) =>
          patch((n) => {
            if (inherit) {
              n.security_override = false
              return
            }
            n.security_override = true
            if (!n.security_profile) {
              Object.assign(n, emptySecurityForm())
            }
          })
        }
      />

      {form.security_override ? (
        <SecurityFormFields
          form={form}
          onChange={(next) => onChange({ ...form, ...next, security_override: true })}
          title={layer === 'path' ? 'Path 级安全覆盖' : 'Host 级安全覆盖'}
          embedded
        />
      ) : (
        <p className="form-hint">
          {layer === 'path'
            ? '未覆盖时使用 Host 规则上的 security，再回退到全局 security:。'
            : '未覆盖时使用全局 security: 配置。'}
        </p>
      )}
    </FormSection>
  )
}
