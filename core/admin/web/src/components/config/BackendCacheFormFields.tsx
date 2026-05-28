import {
  FormCheckbox,
  FormField,
  FormInput,
  FormItem,
  FormMultiSelectField,
  FormSection,
  FormSelectField,
} from '../Form'
import type { BackendForm, CachePathRuleForm } from '../../lib/configEntities'
import { emptyCachePathRule } from '../../lib/configEntities'

const CACHE_BACKEND_METHOD_OPTIONS = ['GET', 'HEAD'] as const
const CACHE_PATH_METHOD_OPTIONS = ['GET', 'HEAD', 'POST'] as const

function pathRuleHasPost(methods: string[]) {
  return methods.map((m) => m.toUpperCase()).includes('POST')
}

function CachePathKeyJsonFields({
  idx,
  idPrefix,
  row,
  updatePathRule,
}: {
  idx: number
  idPrefix: string
  row: CachePathRuleForm
  updatePathRule: (idx: number, fn: (row: CachePathRuleForm) => void) => void
}) {
  const paths = row.key_json.length > 0 ? row.key_json : ['']

  return (
    <FormItem
      label="key_json"
      keyName={`${idPrefix}cache.paths[${idx}].key_json`}
      hint="JSON body 点分路径，如 product.id；配置后缓存键为 httpcache:v2"
    >
      <div className="form-list-rows">
        {paths.map((path, pathIdx) => (
          <div key={pathIdx} className="form-list-row">
            <FormInput
              placeholder="product.id"
              value={path}
              aria-label={`key_json ${pathIdx + 1}`}
              onChange={(e) => {
                updatePathRule(idx, (r) => {
                  const next = [...(r.key_json.length > 0 ? r.key_json : [''])]
                  next[pathIdx] = e.target.value
                  r.key_json = next
                })
              }}
            />
            {paths.length > 1 && (
              <button
                type="button"
                className="btn btn-sm"
                aria-label="删除字段"
                onClick={() => {
                  updatePathRule(idx, (r) => {
                    r.key_json = r.key_json.filter((_, i) => i !== pathIdx)
                  })
                }}
              >
                ✕
              </button>
            )}
          </div>
        ))}
      </div>
      <button
        type="button"
        className="btn btn-sm"
        style={{ marginTop: '0.5rem' }}
        onClick={() => {
          updatePathRule(idx, (r) => {
            r.key_json = [...(r.key_json.length > 0 ? r.key_json : ['']), '']
          })
        }}
      >
        + 添加 JSON 字段
      </button>
    </FormItem>
  )
}

export function BackendCacheFormFields<T extends BackendForm>({
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

  const updatePathRule = (idx: number, fn: (row: CachePathRuleForm) => void) => {
    const rules = [...form.cache_path_rules]
    rules[idx] = { ...rules[idx] }
    fn(rules[idx])
    patch((n) => { n.cache_path_rules = rules })
  }

  const backendMethodsSelected = form.cache_methods
    ? form.cache_methods.split(',').map((s) => s.trim().toUpperCase()).filter(Boolean)
    : []

  const setBackendMethods = (methods: string[]) => {
    patch((n) => {
      n.cache_methods = methods.length ? methods.join(', ') : ''
    })
  }

  return (
    <FormSection title={embedded ? undefined : 'HTTP 响应缓存 backend.cache'}>
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
          <FormMultiSelectField
            label="methods（backend 级）"
            keyName={`${idPrefix}cache.methods`}
            hint="留空默认 GET, HEAD；不可选 POST（POST 仅在路径规则中配置）"
            options={CACHE_BACKEND_METHOD_OPTIONS}
            value={backendMethodsSelected}
            onChange={setBackendMethods}
            placeholder="默认 GET, HEAD"
          />

          <FormSection title="路径规则 cache.paths（可选，自上而下先匹配先生效）">
            <FormSelectField
              label="未命中任何规则时 default"
              keyName={`${idPrefix}cache.default`}
              value={form.cache_default}
              onChange={(e) => patch((n) => {
                n.cache_default = e.target.value === 'bypass' ? 'bypass' : 'cache'
              })}
            >
              <option value="cache">cache（缓存）</option>
              <option value="bypass">bypass（跳过缓存）</option>
            </FormSelectField>
            <p className="form-hint">
              留空规则列表时，该 backend 下所有路径在启用缓存时均参与缓存（与旧行为一致）。
              POST + JSON 查询接口请用 <code>default: bypass</code>，勾选 <code>POST</code> 并配置 <code>key_json</code>。
            </p>
            {form.cache_path_rules.map((row, idx) => {
              const showKeyJSON = row.action === 'cache' && pathRuleHasPost(row.methods)
              return (
                <div key={idx} className="form-section" style={{ marginTop: '0.75rem' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: '0.5rem' }}>
                    <strong>规则 #{idx + 1}</strong>
                    <button
                      type="button"
                      className="btn btn-sm"
                      onClick={() => {
                        patch((n) => {
                          n.cache_path_rules = n.cache_path_rules.filter((_, i) => i !== idx)
                        })
                      }}
                    >
                      删除
                    </button>
                  </div>
                  <FormField
                    label="match"
                    keyName={`${idPrefix}cache.paths[${idx}].match`}
                    hint="路径模式，如 /static/ 或 ^/api/v[0-9]+/public/"
                    value={row.match}
                    onChange={(e) => updatePathRule(idx, (r) => { r.match = e.target.value })}
                  />
                  <FormSelectField
                    label="match_type"
                    keyName={`${idPrefix}cache.paths[${idx}].match_type`}
                    value={row.match_type || 'auto'}
                    onChange={(e) => updatePathRule(idx, (r) => {
                      r.match_type = e.target.value as CachePathRuleForm['match_type']
                    })}
                  >
                    <option value="auto">auto</option>
                    <option value="prefix">prefix</option>
                    <option value="exact">exact</option>
                    <option value="regex">regex</option>
                  </FormSelectField>
                  <FormSelectField
                    label="action"
                    keyName={`${idPrefix}cache.paths[${idx}].action`}
                    value={row.action}
                    onChange={(e) => updatePathRule(idx, (r) => {
                      r.action = e.target.value === 'bypass' ? 'bypass' : 'cache'
                    })}
                  >
                    <option value="cache">cache</option>
                    <option value="bypass">bypass</option>
                  </FormSelectField>
                  {row.action === 'cache' && (
                    <>
                      <FormField
                        label="ttl 覆盖（秒，0=继承 backend TTL）"
                        keyName={`${idPrefix}cache.paths[${idx}].ttl`}
                        type="number"
                        value={row.ttl}
                        onChange={(e) => updatePathRule(idx, (r) => { r.ttl = Number(e.target.value) })}
                      />
                      <FormField
                        label="max_body_bytes 覆盖（0=继承）"
                        keyName={`${idPrefix}cache.paths[${idx}].max_body_bytes`}
                        type="number"
                        value={row.max_body_bytes}
                        onChange={(e) => updatePathRule(idx, (r) => { r.max_body_bytes = Number(e.target.value) })}
                      />
                      <FormMultiSelectField
                        label="methods（路径级）"
                        keyName={`${idPrefix}cache.paths[${idx}].methods`}
                        hint="留空继承 backend 级 methods；POST JSON 缓存须勾选 POST"
                        options={CACHE_PATH_METHOD_OPTIONS}
                        value={row.methods}
                        onChange={(methods) => {
                          updatePathRule(idx, (r) => {
                            r.methods = methods
                            if (!pathRuleHasPost(methods)) {
                              r.key_json = []
                            } else if (r.key_json.length === 0) {
                              r.key_json = ['']
                            }
                          })
                        }}
                        placeholder="继承 backend（GET, HEAD）"
                      />
                      {showKeyJSON && (
                        <>
                          <CachePathKeyJsonFields
                            idx={idx}
                            idPrefix={idPrefix}
                            row={row}
                            updatePathRule={updatePathRule}
                          />
                          <FormField
                            label="key_body_max_bytes（读 body 上限，0=默认 65536）"
                            keyName={`${idPrefix}cache.paths[${idx}].key_body_max_bytes`}
                            type="number"
                            value={row.key_body_max_bytes}
                            onChange={(e) => updatePathRule(idx, (r) => { r.key_body_max_bytes = Number(e.target.value) })}
                          />
                        </>
                      )}
                    </>
                  )}
                </div>
              )
            })}
            <button
              type="button"
              className="btn btn-sm"
              onClick={() => patch((n) => {
                n.cache_path_rules = [...n.cache_path_rules, emptyCachePathRule()]
              })}
            >
              + 添加路径规则
            </button>
          </FormSection>

          <p className="form-hint">
            适用于 service / handler / redirect backend；需顶层 <code>cache</code> 引擎（Redis/内存）。
            命中时访问日志附加 <code>cache_hit=1</code>。
          </p>
        </>
      )}
    </FormSection>
  )
}
