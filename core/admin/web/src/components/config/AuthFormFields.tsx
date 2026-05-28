import {
  FormCheckbox,
  FormField,
  FormSection,
  FormSelectField,
} from '../Form'
import type { BackendForm } from '../../lib/configEntities'

const OAUTH2_PROVIDERS = [
  { value: '', label: '选择 Provider...' },
  { value: 'github', label: 'GitHub' },
  { value: 'gitlab', label: 'GitLab' },
  { value: 'google', label: 'Google' },
  { value: 'microsoft', label: 'Microsoft' },
  { value: 'feishu', label: '飞书 (Feishu)' },
  { value: 'slack', label: 'Slack' },
  { value: 'kakao', label: 'Kakao' },
  { value: 'doreamon', label: 'Doreamon' },
  { value: 'auth0', label: 'Auth0' },
  { value: 'okta', label: 'Okta' },
]

export function AuthFormFields<T extends BackendForm>({
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

  return (
    <FormSection title={embedded ? undefined : '认证 backend.service.auth'}>
      <FormSelectField
        label="认证类型"
        keyName={`${idPrefix}service.auth.type`}
        value={form.auth_type}
        onChange={(e) => {
          const val = e.target.value as BackendForm['auth_type']
          patch((n) => {
            n.auth_type = val
            // Reset enabled to undefined when switching type
            n.auth_enabled = undefined
            // Reset when switching type
            if (val === 'basic' && n.auth_basic_users.length === 0) {
              n.auth_basic_users = [{ username: '', password: '' }]
            }
          })
        }}
      >
        <option value="">无认证</option>
        <option value="basic">Basic Auth</option>
        <option value="bearer">Bearer Token</option>
        <option value="oauth2">OAuth2</option>
        <option value="jwt">JWT</option>
        <option value="oidc">OIDC</option>
      </FormSelectField>

      {form.auth_type !== '' && (
        <FormCheckbox
          label="启用认证 auth.enabled（取消勾选可临时禁用，保留配置）"
          checked={form.auth_enabled !== false}
          onChange={(v) => patch((n) => { n.auth_enabled = v ? undefined : false })}
        />
      )}

      {form.auth_type === 'basic' && (
        <>
          {form.auth_basic_users.map((user, idx) => (
            <div key={idx} style={{ display: 'flex', gap: '0.5rem', alignItems: 'flex-end' }}>
              <FormField
                label={idx === 0 ? '用户名' : ''}
                keyName={`${idPrefix}service.auth.basic.users[${idx}].username`}
                value={user.username}
                onChange={(e) => {
                  const users = [...form.auth_basic_users]
                  users[idx] = { ...users[idx], username: e.target.value }
                  patch((n) => { n.auth_basic_users = users })
                }}
              />
              <FormField
                label={idx === 0 ? '密码' : ''}
                keyName={`${idPrefix}service.auth.basic.users[${idx}].password`}
                type="password"
                value={user.password}
                onChange={(e) => {
                  const users = [...form.auth_basic_users]
                  users[idx] = { ...users[idx], password: e.target.value }
                  patch((n) => { n.auth_basic_users = users })
                }}
              />
              {form.auth_basic_users.length > 1 && (
                <button type="button" className="btn btn-sm" onClick={() => {
                  const users = form.auth_basic_users.filter((_, i) => i !== idx)
                  patch((n) => { n.auth_basic_users = users })
                }}>✕</button>
              )}
            </div>
          ))}
          <button type="button" className="btn btn-sm" onClick={() => {
            patch((n) => { n.auth_basic_users = [...n.auth_basic_users, { username: '', password: '' }] })
          }}>+ 添加用户</button>
        </>
      )}

      {form.auth_type === 'bearer' && (
        <FormField
          label="Tokens（逗号分隔）"
          keyName={`${idPrefix}service.auth.bearer.tokens`}
          hint="多个 token 用英文逗号分隔"
          value={form.auth_bearer_tokens}
          onChange={(e) => patch((n) => { n.auth_bearer_tokens = e.target.value })}
        />
      )}

      {form.auth_type === 'oauth2' && (
        <>
          <FormSelectField
            label="OAuth2 Provider"
            keyName={`${idPrefix}service.auth.oauth2.provider`}
            value={form.auth_oauth2_provider}
            onChange={(e) => patch((n) => { n.auth_oauth2_provider = e.target.value })}
          >
            {OAUTH2_PROVIDERS.map(p => (
              <option key={p.value} value={p.value}>{p.label}</option>
            ))}
          </FormSelectField>
          <FormField
            label="Client ID"
            keyName={`${idPrefix}service.auth.oauth2.client_id`}
            value={form.auth_oauth2_client_id}
            onChange={(e) => patch((n) => { n.auth_oauth2_client_id = e.target.value })}
          />
          <FormField
            label="Client Secret"
            keyName={`${idPrefix}service.auth.oauth2.client_secret`}
            type="password"
            value={form.auth_oauth2_client_secret}
            onChange={(e) => patch((n) => { n.auth_oauth2_client_secret = e.target.value })}
          />
          <FormField
            label="Redirect URL（可选，留空自动生成）"
            keyName={`${idPrefix}service.auth.oauth2.redirect_url`}
            hint="留空时根据请求 Host 自动生成 /oauth2/callback"
            value={form.auth_oauth2_redirect_url}
            onChange={(e) => patch((n) => { n.auth_oauth2_redirect_url = e.target.value })}
          />
          <FormField
            label="Scopes（可选，逗号分隔）"
            keyName={`${idPrefix}service.auth.oauth2.scopes`}
            hint="留空使用 Provider 默认 scopes"
            value={form.auth_oauth2_scopes}
            onChange={(e) => patch((n) => { n.auth_oauth2_scopes = e.target.value })}
          />

          <FormCheckbox
            label="启用 Connect JWT Headers（向上游注入用户信息）"
            checked={form.auth_oauth2_connect_enabled}
            onChange={(v) => patch((n) => { n.auth_oauth2_connect_enabled = v })}
          />

          {form.auth_oauth2_connect_enabled && (
            <>
              <FormField
                label="JWT Secret"
                keyName={`${idPrefix}service.auth.oauth2.connect.jwt.secret`}
                type="password"
                value={form.auth_oauth2_connect_jwt_secret}
                onChange={(e) => patch((n) => { n.auth_oauth2_connect_jwt_secret = e.target.value })}
              />
              <FormSelectField
                label="JWT Algorithm"
                keyName={`${idPrefix}service.auth.oauth2.connect.jwt.algorithm`}
                value={form.auth_oauth2_connect_jwt_algorithm}
                onChange={(e) => patch((n) => { n.auth_oauth2_connect_jwt_algorithm = e.target.value })}
              >
                <option value="hs256">HS256</option>
                <option value="hs384">HS384</option>
                <option value="hs512">HS512</option>
              </FormSelectField>
              <FormField
                label="JWT Expires In"
                keyName={`${idPrefix}service.auth.oauth2.connect.jwt.expires_in`}
                hint="默认 5m（5分钟）"
                value={form.auth_oauth2_connect_jwt_expires_in}
                onChange={(e) => patch((n) => { n.auth_oauth2_connect_jwt_expires_in = e.target.value })}
              />
            </>
          )}
        </>
      )}

      {form.auth_type === 'jwt' && (
        <>
          <FormField
            label="JWT Secret（HS256/384/512）"
            keyName={`${idPrefix}service.auth.jwt.secret`}
            type="password"
            hint="对称密钥；也可写在 auth.secret（旧写法）"
            value={form.auth_jwt_secret}
            onChange={(e) => patch((n) => { n.auth_jwt_secret = e.target.value })}
          />
          <FormField
            label="Public Key PEM（RS*/ES*，可选）"
            keyName={`${idPrefix}service.auth.jwt.public_key`}
            hint="非对称验签时使用"
            value={form.auth_jwt_public_key}
            onChange={(e) => patch((n) => { n.auth_jwt_public_key = e.target.value })}
          />
          <FormSelectField
            label="Algorithm"
            keyName={`${idPrefix}service.auth.jwt.algorithm`}
            value={form.auth_jwt_algorithm}
            onChange={(e) => patch((n) => { n.auth_jwt_algorithm = e.target.value })}
          >
            <option value="HS256">HS256</option>
            <option value="HS384">HS384</option>
            <option value="HS512">HS512</option>
          </FormSelectField>
          <FormField
            label="Issuer（可选）"
            keyName={`${idPrefix}service.auth.jwt.issuer`}
            value={form.auth_jwt_issuer}
            onChange={(e) => patch((n) => { n.auth_jwt_issuer = e.target.value })}
          />
          <FormField
            label="Audience（可选）"
            keyName={`${idPrefix}service.auth.jwt.audience`}
            value={form.auth_jwt_audience}
            onChange={(e) => patch((n) => { n.auth_jwt_audience = e.target.value })}
          />
          <p className="form-hint">客户端在 Authorization: Bearer &lt;token&gt; 中携带 JWT。</p>
        </>
      )}

      {form.auth_type === 'oidc' && (
        <>
          <p className="form-hint">
            配置 <strong>Provider</strong> 启用浏览器重定向登录；配置 <strong>Issuer</strong> 启用 Bearer Token（JWKS）校验 API。
          </p>
          <FormSelectField
            label="OIDC Provider（会话模式）"
            keyName={`${idPrefix}service.auth.oidc.provider`}
            value={form.auth_oidc_provider}
            onChange={(e) => patch((n) => { n.auth_oidc_provider = e.target.value })}
          >
            {OAUTH2_PROVIDERS.map(p => (
              <option key={p.value || 'none'} value={p.value}>{p.label}</option>
            ))}
          </FormSelectField>
          {form.auth_oidc_provider && (
            <>
              <FormField
                label="Client ID"
                keyName={`${idPrefix}service.auth.oidc.client_id`}
                value={form.auth_oidc_client_id}
                onChange={(e) => patch((n) => { n.auth_oidc_client_id = e.target.value })}
              />
              <FormField
                label="Client Secret"
                keyName={`${idPrefix}service.auth.oidc.client_secret`}
                type="password"
                value={form.auth_oidc_client_secret}
                onChange={(e) => patch((n) => { n.auth_oidc_client_secret = e.target.value })}
              />
              <FormField
                label="Redirect URL（可选）"
                keyName={`${idPrefix}service.auth.oidc.redirect_url`}
                value={form.auth_oidc_redirect_url}
                onChange={(e) => patch((n) => { n.auth_oidc_redirect_url = e.target.value })}
              />
              <FormField
                label="Scopes（可选，逗号分隔）"
                keyName={`${idPrefix}service.auth.oidc.scopes`}
                hint="未填时自动包含 openid"
                value={form.auth_oidc_scopes}
                onChange={(e) => patch((n) => { n.auth_oidc_scopes = e.target.value })}
              />
            </>
          )}
          <FormField
            label="Issuer URL（Bearer 模式）"
            keyName={`${idPrefix}service.auth.oidc.issuer`}
            hint="如 https://accounts.example.com — 通过 OIDC Discovery + JWKS 验签"
            value={form.auth_oidc_issuer}
            onChange={(e) => patch((n) => { n.auth_oidc_issuer = e.target.value })}
          />
          <FormField
            label="Audience（Bearer 模式，可选）"
            keyName={`${idPrefix}service.auth.oidc.audience`}
            value={form.auth_oidc_audience}
            onChange={(e) => patch((n) => { n.auth_oidc_audience = e.target.value })}
          />
        </>
      )}
    </FormSection>
  )
}
