import {
  FormCheckbox,
  FormField,
  FormSection,
  FormSelectField,
} from '../Form'
import type { BackendForm } from '../../lib/configEntities'

export function BackendCacheFormFields<T extends BackendForm>({
  form,
  onChange,
  idPrefix = '',
}: {
  form: T
  onChange: (next: T) => void
  idPrefix?: string
}) {
  const patch = (fn: (next: T) => void) => {
    const next = { ...form }
    fn(next)
    onChange(next)
  }

  return (
    <FormSection title="HTTP 响应缓存 backend.cache">
      <FormCheckbox
        label="启用 backend.cache"
        checked={form.cache_enabled}
        onChange={(v) => patch((n) => { n.cache_enabled = v })}
      />
      {form.cache_enabled && (
        <>
          <FormField
            label="TTL（秒）"
            keyName={`${idPrefix}cache.ttl`}
            hint="默认 300；origin 未给出更短 max-age 时使用"
            type="number"
            value={form.cache_ttl}
            onChange={(e) => patch((n) => { n.cache_ttl = Number(e.target.value) })}
          />
          <FormField
            label="最大 body（字节）"
            keyName={`${idPrefix}cache.max_body_bytes`}
            hint="默认 2097152（2MiB）；超出则不存储"
            type="number"
            value={form.cache_max_body_bytes}
            onChange={(e) => patch((n) => { n.cache_max_body_bytes = Number(e.target.value) })}
          />
          <FormSelectField
            label="缓存键 hash"
            keyName={`${idPrefix}cache.key_hash`}
            value={form.cache_key_hash}
            onChange={(e) => patch((n) => { n.cache_key_hash = e.target.value })}
          >
            <option value="md5">md5</option>
            <option value="sha256">sha256</option>
          </FormSelectField>
          <FormCheckbox
            label="skip_vary（忽略 Vary，单变体缓存）"
            checked={form.cache_skip_vary}
            onChange={(v) => patch((n) => { n.cache_skip_vary = v })}
          />
          <FormCheckbox
            label="skip_when_set_cookie（响应含 Set-Cookie 时不缓存）"
            checked={form.cache_skip_when_set_cookie}
            onChange={(v) => patch((n) => { n.cache_skip_when_set_cookie = v })}
          />
          <FormCheckbox
            label="ignore_response_private（允许缓存 private 响应）"
            checked={form.cache_ignore_response_private}
            onChange={(v) => patch((n) => { n.cache_ignore_response_private = v })}
          />
          <FormCheckbox
            label="honor_pragma_no_cache（Pragma: no-cache 触发 bypass）"
            checked={form.cache_honor_pragma_no_cache}
            onChange={(v) => patch((n) => { n.cache_honor_pragma_no_cache = v })}
          />
          <FormField
            label="key_headers"
            keyName={`${idPrefix}cache.key_headers`}
            hint="逗号分隔；留空使用默认 Authorization, Cookie, Accept-Encoding"
            value={form.cache_key_headers}
            onChange={(e) => patch((n) => { n.cache_key_headers = e.target.value })}
          />
          <FormField
            label="methods"
            keyName={`${idPrefix}cache.methods`}
            hint="逗号分隔；默认 GET, HEAD"
            value={form.cache_methods}
            onChange={(e) => patch((n) => { n.cache_methods = e.target.value })}
          />
          <p className="form-hint">
            适用于 service / handler / redirect backend；需顶层 <code>cache</code> 引擎（Redis/内存）。
            命中时访问日志附加 <code>cache_hit=1</code>。
          </p>
        </>
      )}
    </FormSection>
  )
}
