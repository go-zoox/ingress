import { FormCheckbox, FormField, FormGrid, FormSelectField, FormTextareaField } from '../Form'
import type { RuleForm } from '../../lib/configEntities'

export function ConnectProxyFormFields({
  form,
  onChange,
  idPrefix = '',
}: {
  form: RuleForm
  onChange: (next: RuleForm) => void
  idPrefix?: string
}) {
  const patch = (fn: (next: RuleForm) => void) => {
    const next = { ...form }
    fn(next)
    onChange(next)
  }

  return (
    <FormGrid columns={1}>
      <FormCheckbox
        label="启用正向代理（CONNECT + HTTP absolute-form）"
        checked={form.connect_proxy_enabled}
        onChange={(v) => patch((n) => { n.connect_proxy_enabled = v })}
      />
      <p className="form-hint">
        CONNECT 用于 HTTPS 目标，absolute-form 用于 HTTP 目标。两者都按 TLS SNI 匹配本规则；
        普通 HTTPS 请求仍使用上方 Backend。
      </p>

      {form.connect_proxy_enabled ? (
        <>
          <FormField
            label="代理上游服务名"
            keyName={`${idPrefix}connect_proxy.upstream.name`}
            value={form.connect_proxy_upstream_name}
            placeholder="留空则回退到当前 Backend service"
            onChange={(e) => patch((n) => { n.connect_proxy_upstream_name = e.target.value })}
            hint="独立上游，避免普通 HTTPS 请求也被转到代理端口"
          />
          <FormGrid columns={2}>
            <FormField
              label="代理上游端口"
              type="number"
              value={form.connect_proxy_upstream_port}
              onChange={(e) => patch((n) => { n.connect_proxy_upstream_port = Number(e.target.value) })}
            />
            <FormSelectField
              label="代理上游协议"
              value={form.connect_proxy_upstream_protocol}
              onChange={(e) => patch((n) => { n.connect_proxy_upstream_protocol = e.target.value })}
            >
              <option value="http">http</option>
              <option value="https">https</option>
            </FormSelectField>
          </FormGrid>

          <FormTextareaField
            label="目标白名单（每行一个正则，整串匹配）"
            value={form.connect_proxy_allowed_targets}
            rows={4}
            mono
            placeholder={'^.+:443$\n^.+:80$'}
            onChange={(e) => patch((n) => { n.connect_proxy_allowed_targets = e.target.value })}
            hint="匹配 host:port；HTTP 目标未写端口时按 :80 归一化。空列表表示不限制目标。"
          />

          <p className="form-hint">
            认证由下游代理校验 <code>Proxy-Authorization</code>，407 原样回给客户端；
            本路由的限流继承「路由限流」分区与全局 rate_limit。
          </p>

          <FormGrid columns={2}>
            <FormField
              label="连接超时（秒）"
              type="number"
              value={form.connect_proxy_connect_timeout}
              onChange={(e) => patch((n) => { n.connect_proxy_connect_timeout = Number(e.target.value) })}
            />
            <FormField
              label="上游响应超时（秒）"
              type="number"
              value={form.connect_proxy_response_timeout}
              onChange={(e) => patch((n) => { n.connect_proxy_response_timeout = Number(e.target.value) })}
            />
            <FormField
              label="空闲断开（秒）"
              type="number"
              value={form.connect_proxy_idle_timeout}
              onChange={(e) => patch((n) => { n.connect_proxy_idle_timeout = Number(e.target.value) })}
            />
            <FormField
              label="最大连接数 / 实例"
              type="number"
              value={form.connect_proxy_max_connections}
              onChange={(e) => patch((n) => { n.connect_proxy_max_connections = Number(e.target.value) })}
              hint="0 表示不限制"
            />
          </FormGrid>
        </>
      ) : null}
    </FormGrid>
  )
}
