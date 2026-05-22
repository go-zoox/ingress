import {
  FormField,
  FormSection,
  FormSelectField,
} from '../Form'
import {
  DEFAULT_HANDLER_SCRIPT,
  HANDLER_SCRIPT_PLACEHOLDER,
  type BackendForm,
  type HandlerEngine,
  type HandlerType,
} from '../../lib/configEntities'

export function HandlerFormFields<T extends BackendForm>({
  form,
  onChange,
}: {
  form: T
  onChange: (next: T) => void
}) {
  const patch = (fn: (next: T) => void) => {
    const next = { ...form }
    fn(next)
    onChange(next)
  }

  return (
    <>
      <FormSelectField
        label="Handler 类型"
        keyName="handler.type"
        value={form.handler_type}
        onChange={(e) => patch((n) => { n.handler_type = e.target.value as HandlerType })}
      >
        <option value="static_response">static_response（固定响应）</option>
        <option value="file_server">file_server（静态文件）</option>
        <option value="templates">templates（Go 模板）</option>
        <option value="script">script（脚本）</option>
      </FormSelectField>

      {form.handler_type === 'static_response' && (
        <FormSection title="static_response">
          <FormField
            label="状态码"
            keyName="handler.status_code"
            type="number"
            value={form.handler_status_code}
            onChange={(e) => patch((n) => { n.handler_status_code = Number(e.target.value) })}
          />
          <FormField
            label="Content-Type"
            keyName="handler.headers.Content-Type"
            hint="写入 handler.headers"
            value={form.handler_content_type}
            onChange={(e) => patch((n) => { n.handler_content_type = e.target.value })}
          />
          <FormField
            label="响应 Body"
            keyName="handler.body"
            value={form.handler_body}
            onChange={(e) => patch((n) => { n.handler_body = e.target.value })}
          />
        </FormSection>
      )}

      {form.handler_type === 'file_server' && (
        <FormSection title="file_server">
          <FormField
            label="根目录 root_dir"
            keyName="handler.root_dir"
            hint="相对进程工作目录，如 ./static"
            value={form.handler_root_dir}
            onChange={(e) => patch((n) => { n.handler_root_dir = e.target.value })}
          />
          <FormField
            label="索引文件 index_file"
            keyName="handler.index_file"
            value={form.handler_index_file}
            onChange={(e) => patch((n) => { n.handler_index_file = e.target.value })}
          />
        </FormSection>
      )}

      {form.handler_type === 'templates' && (
        <FormSection title="templates">
          <FormField
            label="模板目录 root_dir"
            keyName="handler.root_dir"
            hint="相对进程工作目录，如 ./templates"
            value={form.handler_root_dir}
            onChange={(e) => patch((n) => { n.handler_root_dir = e.target.value })}
          />
          <p className="form-hint">
            模板数据：<code>{'{{.Path}}'}</code>、<code>{'{{.Method}}'}</code>
          </p>
        </FormSection>
      )}

      {form.handler_type === 'script' && (
        <FormSection title="script">
          <FormSelectField
            label="脚本引擎 engine"
            keyName="handler.engine"
            value={form.handler_engine}
            onChange={(e) =>
              patch((n) => {
                const engine = e.target.value as HandlerEngine
                const prevDefault = DEFAULT_HANDLER_SCRIPT[n.handler_engine]
                n.handler_engine = engine
                if (!n.handler_script.trim() || n.handler_script === prevDefault) {
                  n.handler_script = DEFAULT_HANDLER_SCRIPT[engine]
                }
              })
            }
          >
            <option value="javascript">javascript（goja，ctx.status / ctx.fetch）</option>
            <option value="go">go（yaegi，ctx.SetHeader / ctx.String / ctx.Fetch）</option>
          </FormSelectField>
          {form.handler_engine === 'javascript' && (
            <>
              <FormField
                label="初始状态码"
                keyName="handler.status_code"
                hint="脚本可通过 ctx.status 修改"
                type="number"
                value={form.handler_status_code}
                onChange={(e) => patch((n) => { n.handler_status_code = Number(e.target.value) })}
              />
              <FormField
                label="初始 Content-Type"
                keyName="handler.headers.Content-Type"
                value={form.handler_content_type}
                onChange={(e) => patch((n) => { n.handler_content_type = e.target.value })}
              />
            </>
          )}
          <label className="form-item form-item--full">
            <span className="form-label">
              <span className="form-label-text">脚本 script</span>
              <code className="form-key">handler.script</code>
            </span>
            <span className="form-control-wrap">
              <textarea
                className="code config-module-text form-control"
                spellCheck={false}
                rows={10}
                placeholder={HANDLER_SCRIPT_PLACEHOLDER[form.handler_engine]}
                value={form.handler_script}
                onChange={(e) => patch((n) => { n.handler_script = e.target.value })}
              />
            </span>
          </label>
        </FormSection>
      )}
    </>
  )
}
