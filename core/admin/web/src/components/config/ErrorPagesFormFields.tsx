import { FormField, FormSection, FormSelectField } from '../Form'
import {
  ERROR_PAGE_STATUS_CODES,
  type ErrorPageCode,
  type ErrorPageEntryForm,
  type ErrorPageMode,
  errorPageEntryConfigured,
  errorPagesFromDoc,
  patchErrorPagesOnDoc,
} from '../../lib/errorPages'

type Props = {
  doc: Record<string, unknown>
  onChange: (doc: Record<string, unknown>) => void
}

export function ErrorPagesFormFields({ doc, onChange }: Props) {
  const entries = errorPagesFromDoc(doc)
  const configuredCount = ERROR_PAGE_STATUS_CODES.filter(({ code }) =>
    errorPageEntryConfigured(entries[code]),
  ).length

  const patchEntries = (fn: (next: Record<ErrorPageCode, ErrorPageEntryForm>) => void) => {
    const next = { ...entries }
    for (const { code } of ERROR_PAGE_STATUS_CODES) {
      next[code] = { ...entries[code] }
    }
    fn(next)
    const docNext = { ...doc }
    patchErrorPagesOnDoc(docNext, next)
    onChange(docNext)
  }

  const patchEntry = (code: ErrorPageCode, fn: (entry: ErrorPageEntryForm) => void) => {
    patchEntries((next) => {
      fn(next[code])
    })
  }

  return (
    <FormSection title="错误页面 error_pages">
      <p className="form-hint">
        支持 401 / 403 / 404 / 500 / 502 / 503 / 504。未单独配置的状态码使用 ingress 内置 HTML 模板。
        {configuredCount > 0 ? ` 已自定义 ${configuredCount} 个。` : null}
      </p>
      <div className="error-pages-list">
        {ERROR_PAGE_STATUS_CODES.map(({ code, label, hint }) => {
          const entry = entries[code]
          const active = errorPageEntryConfigured(entry)
          return (
            <details
              key={code}
              className={`error-page-item${active ? ' error-page-item--active' : ''}`}
              {...(active ? { open: true } : {})}
            >
              <summary className="error-page-summary">
                <span className="error-page-code">{code}</span>
                <span className="error-page-label">{label}</span>
                {active ? <span className="error-page-badge">{modeLabel(entry.mode)}</span> : null}
              </summary>
              <div className="error-page-body">
                <p className="form-hint">{hint}</p>
                <FormSelectField
                  label="页面类型"
                  value={entry.mode}
                  onChange={(e) => {
                    const mode = e.target.value as ErrorPageMode
                    patchEntry(code, (row) => {
                      row.mode = mode
                      if (mode === 'default') {
                        row.title = ''
                        row.subtitle = ''
                        row.file = ''
                        row.body = ''
                      }
                    })
                  }}
                >
                  <option value="default">内置（默认文案）</option>
                  <option value="builtin">内置（自定义标题/说明）</option>
                  <option value="file">自定义 HTML 文件</option>
                  <option value="inline">内联 HTML</option>
                </FormSelectField>

                {entry.mode === 'builtin' ? (
                  <>
                    <FormField
                      label="标题 title"
                      hint="留空则使用内置默认标题"
                      value={entry.title}
                      onChange={(e) => patchEntry(code, (row) => { row.title = e.target.value })}
                    />
                    <FormField
                      label="说明 subtitle"
                      hint="留空则使用内置默认说明"
                      full
                      value={entry.subtitle}
                      onChange={(e) => patchEntry(code, (row) => { row.subtitle = e.target.value })}
                    />
                  </>
                ) : null}

                {entry.mode === 'file' ? (
                  <FormField
                    label="HTML 文件路径"
                    hint="相对 ingress.yaml 所在目录，或绝对路径"
                    full
                    value={entry.file}
                    onChange={(e) => patchEntry(code, (row) => { row.file = e.target.value })}
                  />
                ) : null}

                {entry.mode === 'inline' ? (
                  <label className="form-item form-item--full">
                    <span className="form-label">内联 HTML body</span>
                    <textarea
                      className="code config-module-text form-control error-page-inline-body"
                      rows={8}
                      spellCheck={false}
                      value={entry.body}
                      onChange={(e) => patchEntry(code, (row) => { row.body = e.target.value })}
                    />
                  </label>
                ) : null}
              </div>
            </details>
          )
        })}
      </div>
    </FormSection>
  )
}

function modeLabel(mode: ErrorPageMode) {
  switch (mode) {
    case 'builtin':
      return '内置'
    case 'file':
      return '文件'
    case 'inline':
      return '内联'
    default:
      return ''
  }
}
