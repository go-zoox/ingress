import {
  FormCheckbox,
  FormField,
  FormSelectField,
} from '../Form'
import {
  redirectStatusLabel,
  type RedirectDuration,
  type RedirectFormSlice,
} from '../../lib/redirectForm'

type PatchFn = (fn: (next: RedirectFormSlice) => void) => void

export function RedirectFormFields({
  form,
  patch,
  idPrefix = '',
  showURL = true,
  showPreservePath = false,
}: {
  form: RedirectFormSlice
  patch: PatchFn
  idPrefix?: string
  /** When false, only duration / preserve_request (e.g. https.redirect_from_http). */
  showURL?: boolean
  /** paths[].backend.redirect only: hosts always keep the request path, so the toggle is hidden elsewhere. */
  showPreservePath?: boolean
}) {
  const status = redirectStatusLabel(form.redirect_duration, form.redirect_preserve_request)

  return (
    <>
      {showURL && (
        <FormField
          label="重定向 URL"
          keyName={`${idPrefix}redirect.url`}
          value={form.redirect_url}
          onChange={(e) => patch((n) => { n.redirect_url = e.target.value })}
        />
      )}
      {showPreservePath && (
        <>
          <FormCheckbox
            label="保留原请求路径与 query (preserve_path)"
            checked={form.redirect_preserve_path}
            onChange={(v) => patch((n) => { n.redirect_preserve_path = v })}
          />
          <p className="form-hint form-item--full">
            关闭（默认）：Location 就是上面填的 URL 本身，不追加请求的 path/query；
            勾选后：URL 未写路径时沿用请求的 path 与 query（整站迁移语义）。
            URL 里写了路径时始终以该路径为准。
          </p>
        </>
      )}
      <FormSelectField
        label="时效 (duration)"
        keyName={`${idPrefix}redirect.duration`}
        value={form.redirect_duration}
        onChange={(e) => patch((n) => { n.redirect_duration = e.target.value as RedirectDuration })}
      >
        <option value="temporary">temporary — 临时跳转</option>
        <option value="permanent">permanent — 永久跳转</option>
      </FormSelectField>
      <FormCheckbox
        label="保留原请求方法与 body (preserve_request)"
        checked={form.redirect_preserve_request}
        onChange={(v) => patch((n) => { n.redirect_preserve_request = v })}
      />
      <p className="form-hint form-item--full">
        当前状态码：<code>{status}</code>
        {' '}— 未勾选 preserve_request 时浏览器可能对 POST 使用 GET；API/表单场景建议勾选以使用 307/308。
      </p>
    </>
  )
}
