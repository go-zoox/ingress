import {
  FormCheckbox,
  FormField,
  FormSection,
  FormSelectField,
} from '../Form'
import type { RateLimitFormSlice } from '../../lib/configEntities'

export function RateLimitFormFields<T extends RateLimitFormSlice>({
  form,
  onChange,
  idPrefix = '',
  title = '限流 rate_limit',
  embedded = false,
}: {
  form: T
  onChange: (next: T) => void
  idPrefix?: string
  title?: string
  embedded?: boolean
}) {
  const patch = (fn: (next: T) => void) => {
    const next = { ...form }
    fn(next)
    onChange(next)
  }

  const configured = form.rate_limit_requests > 0 || form.rate_limit_enabled === true

  return (
    <FormSection title={embedded ? undefined : title}>
      <FormCheckbox
        label="启用限流"
        checked={form.rate_limit_enabled !== false && (configured || form.rate_limit_enabled === true)}
        onChange={(v) => patch((n) => {
          n.rate_limit_enabled = v ? true : false
          if (v && n.rate_limit_requests <= 0) {
            n.rate_limit_requests = 100
            n.rate_limit_period = 60
          }
        })}
      />
      {(configured || form.rate_limit_enabled === true) && form.rate_limit_enabled !== false && (
        <>
          <FormField
            label="请求上限 requests"
            keyName={`${idPrefix}rate_limit.requests`}
            type="number"
            hint="窗口内允许的最大请求数"
            value={form.rate_limit_requests || ''}
            onChange={(e) => patch((n) => { n.rate_limit_requests = Number(e.target.value) })}
          />
          <FormField
            label="窗口 period（秒）"
            keyName={`${idPrefix}rate_limit.period`}
            type="number"
            value={form.rate_limit_period || ''}
            onChange={(e) => patch((n) => { n.rate_limit_period = Number(e.target.value) })}
          />
          <FormSelectField
            label="计数维度 key"
            keyName={`${idPrefix}rate_limit.key`}
            value={form.rate_limit_key || 'ip'}
            onChange={(e) => patch((n) => { n.rate_limit_key = e.target.value as RateLimitFormSlice['rate_limit_key'] })}
          >
            <option value="ip">ip（客户端 IP）</option>
            <option value="global">global（全网关）</option>
            <option value="route">route（按路由）</option>
            <option value="header">header（按请求头值）</option>
          </FormSelectField>
          {form.rate_limit_key === 'header' && (
            <FormField
              label="Header 名称"
              keyName={`${idPrefix}rate_limit.header`}
              hint="如 X-API-Key、Authorization"
              value={form.rate_limit_header}
              onChange={(e) => patch((n) => { n.rate_limit_header = e.target.value })}
            />
          )}
          {form.rate_limit_key === 'ip' && (
            <>
              <FormCheckbox
                label="trust_proxy（从 X-Forwarded-For 取 IP）"
                checked={form.rate_limit_trust_proxy}
                onChange={(v) => patch((n) => { n.rate_limit_trust_proxy = v })}
              />
              {form.rate_limit_trust_proxy && (
                <FormField
                  label="xff_index"
                  keyName={`${idPrefix}rate_limit.xff_index`}
                  type="number"
                  hint="0=最左；负数从右计数"
                  value={form.rate_limit_xff_index}
                  onChange={(e) => patch((n) => { n.rate_limit_xff_index = Number(e.target.value) })}
                />
              )}
            </>
          )}
          <p className="form-hint">
            超出限制返回 429，响应含 Retry-After。配置 Redis 缓存时，限流计数器可共享 Redis。
          </p>
        </>
      )}
    </FormSection>
  )
}
