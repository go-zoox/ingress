import {
  FormField,
  FormInput,
  FormItem,
  FormSection,
  FormSelectField,
} from '../Form'
import type { BackendForm, StringKVForm } from '../../lib/configEntities'

function KVListFields({
  label,
  hint,
  rows,
  onChange,
}: {
  label: string
  hint?: string
  rows: StringKVForm[]
  onChange: (rows: StringKVForm[]) => void
}) {
  const updateRow = (idx: number, field: 'key' | 'value', value: string) => {
    const next = rows.map((row, i) => (i === idx ? { ...row, [field]: value } : row))
    onChange(next)
  }

  const removeRow = (idx: number) => {
    const next = rows.filter((_, i) => i !== idx)
    onChange(next.length > 0 ? next : [{ key: '', value: '' }])
  }

  return (
    <FormItem label={label} hint={hint}>
      {rows.length > 0 && (
        <div className="form-list-rows">
          {rows.map((row, idx) => (
            <div key={idx} className="form-list-row">
              <FormInput
                placeholder="键名"
                value={row.key}
                aria-label={`${label} key ${idx + 1}`}
                onChange={(e) => updateRow(idx, 'key', e.target.value)}
              />
              <FormInput
                placeholder="值"
                value={row.value}
                aria-label={`${label} value ${idx + 1}`}
                onChange={(e) => updateRow(idx, 'value', e.target.value)}
              />
              <button
                type="button"
                className="btn btn-sm"
                aria-label={`删除 ${label} ${idx + 1}`}
                onClick={() => removeRow(idx)}
              >
                ✕
              </button>
            </div>
          ))}
        </div>
      )}
      <button
        type="button"
        className="btn btn-sm"
        style={{ marginTop: rows.length > 0 ? '0.5rem' : 0 }}
        onClick={() => onChange([...rows, { key: '', value: '' }])}
      >
        + 添加
      </button>
    </FormItem>
  )
}

export function ServiceRequestFormFields<T extends BackendForm>({
  form,
  onChange,
  idPrefix = '',
  embedded = false,
  showStripPrefixHint = false,
}: {
  form: T
  onChange: (next: T) => void
  idPrefix?: string
  embedded?: boolean
  /** Path-level editor: warn when strip_prefix conflicts with path rewrites. */
  showStripPrefixHint?: boolean
}) {
  const patch = (fn: (next: T) => void) => {
    const next = { ...form }
    fn(next)
    onChange(next)
  }

  const pathRewrites = form.service_request_path_rewrites
  const stripPrefixConflict =
    showStripPrefixHint &&
    form.service_strip_prefix &&
    pathRewrites.some((r) => r.trim())

  const updatePathRewrite = (idx: number, value: string) => {
    patch((n) => {
      const next = [...n.service_request_path_rewrites]
      next[idx] = value
      n.service_request_path_rewrites = next
    })
  }

  const removePathRewrite = (idx: number) => {
    patch((n) => {
      n.service_request_path_rewrites = n.service_request_path_rewrites.filter((_, i) => i !== idx)
    })
  }

  return (
    <>
      <FormSection title={embedded ? undefined : '上游请求 service.request'}>
        <FormSelectField
          label="Host 改写 request.host.rewrite"
          keyName={`${idPrefix}service.request.host.rewrite`}
          hint="默认跟随 service.mode；external 通常等价于改写为上游 Host"
          value={form.service_request_host_rewrite}
          onChange={(e) => patch((n) => {
            n.service_request_host_rewrite = e.target.value as BackendForm['service_request_host_rewrite']
          })}
        >
          <option value="default">默认（跟随 mode）</option>
          <option value="true">强制改写 Host</option>
          <option value="false">不改写 Host</option>
        </FormSelectField>

        <FormItem
          label="路径改写 request.path.rewrites"
          hint="格式 ^/prefix/(.*):/$1；与 strip_prefix 不可同时配置"
        >
          {stripPrefixConflict && (
            <p className="form-hint form-hint--warn" style={{ marginBottom: '0.5rem' }}>
              已启用 strip_prefix，请关闭 strip_prefix 或清空路径改写规则后再保存。
            </p>
          )}
          {pathRewrites.length > 0 && (
            <div className="form-list-rows">
              {pathRewrites.map((rule, idx) => (
                <div key={idx} className="form-list-row">
                  <FormInput
                    placeholder="^/api/v1/(.*):/api/v2/$1"
                    value={rule}
                    aria-label={`path rewrite ${idx + 1}`}
                    onChange={(e) => updatePathRewrite(idx, e.target.value)}
                  />
                  <button
                    type="button"
                    className="btn btn-sm"
                    aria-label={`删除路径改写 ${idx + 1}`}
                    onClick={() => removePathRewrite(idx)}
                  >
                    ✕
                  </button>
                </div>
              ))}
            </div>
          )}
          <button
            type="button"
            className="btn btn-sm"
            style={{ marginTop: pathRewrites.length > 0 ? '0.5rem' : 0 }}
            onClick={() => patch((n) => {
              n.service_request_path_rewrites = [...n.service_request_path_rewrites, '']
            })}
          >
            + 添加路径规则
          </button>
        </FormItem>

        <KVListFields
          label="请求头 request.headers"
          hint="转发到上游时附加或覆盖的请求头"
          rows={form.service_request_headers}
          onChange={(rows) => patch((n) => { n.service_request_headers = rows })}
        />

        <KVListFields
          label="查询参数 request.query"
          hint="转发到上游时附加或覆盖的 query 参数"
          rows={form.service_request_query}
          onChange={(rows) => patch((n) => { n.service_request_query = rows })}
        />

        <FormField
          label="转发延迟 request.delay（毫秒）"
          keyName={`${idPrefix}service.request.delay`}
          type="number"
          hint="发送上游请求前的延迟；0 表示不设置"
          value={form.service_request_delay || ''}
          onChange={(e) => patch((n) => {
            n.service_request_delay = Number(e.target.value) || 0
          })}
        />

        <FormField
          label="上游超时 request.timeout（秒）"
          keyName={`${idPrefix}service.request.timeout`}
          type="number"
          hint="上游请求超时；0 表示使用框架默认"
          value={form.service_request_timeout || ''}
          onChange={(e) => patch((n) => {
            n.service_request_timeout = Number(e.target.value) || 0
          })}
        />
      </FormSection>

      <FormSection title={embedded ? undefined : '上游响应 service.response'}>
        <KVListFields
          label="响应头 response.headers"
          hint="由 ingress 写入客户端响应的额外头（与安全模块独立）"
          rows={form.service_response_headers}
          onChange={(rows) => patch((n) => { n.service_response_headers = rows })}
        />
      </FormSection>
    </>
  )
}
