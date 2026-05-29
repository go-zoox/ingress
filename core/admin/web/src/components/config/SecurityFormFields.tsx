import type { ReactNode } from 'react'
import {
  FormCheckbox,
  FormField,
  FormSection,
  FormSelectField,
} from '../Form'
import type { SecurityFormSlice } from '../../lib/configEntities'
import {
  applySecurityProfileSwitch,
  effectiveCSP,
  effectiveCORSEnabled,
  effectiveFrame,
  effectiveReferrerPolicy,
  profileDefaults,
  SECURITY_PROFILE_DEFAULTS,
  type ActiveSecurityProfile,
} from '../../lib/securityProfiles'

function SecurityFeatureGroup({
  title,
  hint,
  enabled,
  onEnabledChange,
  children,
}: {
  title: string
  hint?: string
  enabled: boolean
  onEnabledChange: (enabled: boolean) => void
  children: ReactNode
}) {
  return (
    <div className={`security-feature-group${enabled ? ' is-expanded' : ''}`}>
      <div className="security-feature-group-head">
        <FormCheckbox label={title} checked={enabled} onChange={onEnabledChange} />
        {hint && !enabled ? <p className="form-hint security-feature-group-hint">{hint}</p> : null}
      </div>
      {enabled ? <div className="security-feature-group-body">{children}</div> : null}
    </div>
  )
}

function PresetEffectiveValue({ label, value }: { label: string; value: string }) {
  if (!value) return null
  return (
    <div className="security-preset-effective">
      <span className="security-preset-effective-label">{label}</span>
      <code className="security-preset-effective-value">{value}</code>
    </div>
  )
}

function ProfilePreview({ profile }: { profile: keyof typeof SECURITY_PROFILE_DEFAULTS }) {
  const def = SECURITY_PROFILE_DEFAULTS[profile]
  const others = (Object.keys(SECURITY_PROFILE_DEFAULTS) as ActiveSecurityProfile[]).filter((p) => p !== profile)

  return (
    <div className="security-profile-preview" role="status">
      <p className="security-profile-preview-title">当前预设生效项</p>
      <ul className="security-profile-preview-list">
        <li><strong>iframe：</strong>{def.frameLabel}</li>
        <li><strong>CSP：</strong><code>{def.csp}</code></li>
        <li>
          <strong>CORS：</strong>
          {def.corsEnabled ? '开启（须填写允许的来源）' : '关闭'}
        </li>
        <li><strong>HSTS：</strong>HTTPS 请求时下发</li>
        <li><strong>nosniff / Referrer：</strong>开启 · {def.referrerPolicy}</li>
      </ul>
      <p className="form-hint security-profile-preview-diff">
        与 {others.map((p) => SECURITY_PROFILE_DEFAULTS[p].label).join('、')} 的主要差异：
        {profile === 'strict' && ' CSP 允许同源资源；无 CORS'}
        {profile === 'api' && ' CSP 禁止默认加载任何资源；默认开启 CORS'}
        {profile === 'embeddable' && ' iframe 允许同源嵌入；CSP 仅限制 frame-ancestors'}
      </p>
    </div>
  )
}

export function SecurityFormFields({
  form,
  onChange,
  title = '安全 security',
  embedded = false,
}: {
  form: SecurityFormSlice
  onChange: (next: SecurityFormSlice) => void
  title?: string
  embedded?: boolean
  idPrefix?: string
}) {
  const patch = (fn: (next: SecurityFormSlice) => void) => {
    const next = { ...form }
    fn(next)
    onChange(next)
  }

  const profile = form.security_profile || 'off'
  const def = profileDefaults(profile)
  const hstsMode = form.security_hsts || 'auto'
  const frameMode = form.security_frame || 'inherit'
  const hstsEnabled = hstsMode !== 'off'
  const frameEnabled = frameMode !== 'off'
  const referrerCustom = form.security_referrer_policy.trim() !== ''
  const cspCustom = form.security_csp.trim() !== ''
  const frameEffective = effectiveFrame(form)
  const frameEffectiveLabel =
    frameEffective === 'deny'
      ? '禁止嵌入 (DENY)'
      : frameEffective === 'sameorigin'
        ? '同源可嵌入 (SAMEORIGIN)'
        : '关闭'
  const corsEnabled = effectiveCORSEnabled(form)
  const presetReferrer = effectiveReferrerPolicy(form)
  const presetCsp = effectiveCSP(form)

  return (
    <FormSection title={embedded ? undefined : title}>
      <div className="security-form-profile">
        <FormSelectField
          label="安全预设"
          keyName="security.profile"
          value={profile}
          onChange={(e) =>
            onChange(
              applySecurityProfileSwitch(
                form,
                e.target.value as SecurityFormSlice['security_profile'],
              ),
            )
          }
        >
          <option value="off">关闭</option>
          <option value="strict">通用 Web 站点 (strict)</option>
          <option value="api">API 接口 (api)</option>
          <option value="embeddable">可被 iframe 嵌入 (embeddable)</option>
        </FormSelectField>
        {def ? <p className="form-hint">{def.summary}</p> : null}
      </div>

      {isActiveProfile(profile) ? <ProfilePreview profile={profile} /> : null}

      {profile !== 'off' ? (
        <div className="security-form-groups">
          <FormSection title="传输安全">
            <SecurityFeatureGroup
              title="强制 HTTPS（HSTS）"
              hint="启用后浏览器仅通过 HTTPS 访问站点"
              enabled={hstsEnabled}
              onEnabledChange={(v) =>
                patch((n) => {
                  n.security_hsts = v ? 'auto' : 'off'
                })
              }
            >
              <FormSelectField
                label="下发策略"
                keyName="security.hsts"
                value={hstsMode === 'off' ? 'auto' : hstsMode}
                onChange={(e) =>
                  patch((n) => {
                    n.security_hsts = e.target.value as SecurityFormSlice['security_hsts']
                  })
                }
              >
                <option value="auto">仅 HTTPS 连接时下发（三个预设默认）</option>
                <option value="on">始终下发</option>
              </FormSelectField>
            </SecurityFeatureGroup>
          </FormSection>

          <FormSection title="嵌入与隔离">
            <SecurityFeatureGroup
              title="iframe 嵌入策略"
              hint={`当前生效：${frameEffectiveLabel}`}
              enabled={frameEnabled}
              onEnabledChange={(v) =>
                patch((n) => {
                  n.security_frame = v ? 'inherit' : 'off'
                })
              }
            >
              <FormSelectField
                label="策略"
                keyName="security.frame"
                value={frameMode === 'off' ? 'inherit' : frameMode}
                onChange={(e) =>
                  patch((n) => {
                    n.security_frame = e.target.value as SecurityFormSlice['security_frame']
                  })
                }
              >
                <option value="inherit">
                  跟随预设（{def?.frameLabel ?? frameEffectiveLabel}）
                </option>
                <option value="deny">禁止嵌入 (DENY)</option>
                <option value="sameorigin">同源可嵌入 (SAMEORIGIN)</option>
              </FormSelectField>
              {frameMode === 'inherit' && def ? (
                <PresetEffectiveValue label="预设生效" value={def.frameLabel} />
              ) : null}
            </SecurityFeatureGroup>
          </FormSection>

          <FormSection title="内容安全">
            <div className="security-feature-group is-expanded security-feature-group--plain">
              <FormCheckbox
                label="MIME 嗅探防护"
                checked={form.security_content_type_options !== false}
                onChange={(v) => patch((n) => { n.security_content_type_options = v })}
              />
              <p className="form-hint security-feature-group-hint security-feature-group-hint--inline">
                禁止浏览器猜测响应类型（nosniff）；三个预设默认开启
              </p>
            </div>
            <SecurityFeatureGroup
              title="来源页策略 (Referrer-Policy)"
              hint={referrerCustom ? undefined : `预设：${presetReferrer}`}
              enabled={referrerCustom || Boolean(presetReferrer)}
              onEnabledChange={(v) =>
                patch((n) => {
                  if (v && !n.security_referrer_policy.trim()) {
                    n.security_referrer_policy = presetReferrer || 'strict-origin-when-cross-origin'
                  } else if (!v) {
                    n.security_referrer_policy = ''
                  }
                })
              }
            >
              {!referrerCustom && presetReferrer ? (
                <PresetEffectiveValue label="预设生效" value={presetReferrer} />
              ) : null}
              <FormField
                label="自定义策略值"
                keyName="security.referrer_policy"
                hint="留空则使用预设；填 off 表示关闭"
                value={form.security_referrer_policy}
                onChange={(e) => patch((n) => { n.security_referrer_policy = e.target.value })}
              />
            </SecurityFeatureGroup>
            <SecurityFeatureGroup
              title="CSP 策略"
              hint={cspCustom ? undefined : '各预设 CSP 不同，见上方说明'}
              enabled={cspCustom || Boolean(presetCsp)}
              onEnabledChange={(v) =>
                patch((n) => {
                  if (v && !n.security_csp.trim()) {
                    n.security_csp = presetCsp || "default-src 'self'"
                  } else if (!v) {
                    n.security_csp = ''
                  }
                })
              }
            >
              {!cspCustom && presetCsp ? (
                <PresetEffectiveValue label="预设生效" value={presetCsp} />
              ) : null}
              <FormField
                label="自定义 CSP"
                keyName="security.csp"
                hint="留空则使用预设；填 off 表示关闭"
                value={form.security_csp}
                onChange={(e) => patch((n) => { n.security_csp = e.target.value })}
              />
            </SecurityFeatureGroup>
          </FormSection>

          <FormSection title="跨域访问">
            <SecurityFeatureGroup
              title="跨域资源共享（CORS）"
              hint={
                profile === 'api'
                  ? 'api 预设默认开启；需填写允许的来源'
                  : 'strict / embeddable 预设默认关闭'
              }
              enabled={corsEnabled}
              onEnabledChange={(v) => patch((n) => { n.security_cors_enabled = v })}
            >
              {profile !== 'api' && !form.security_cors_enabled && !form.security_cors_origins.trim() ? (
                <p className="form-hint">当前预设不启用 CORS；勾选后可手动开启。</p>
              ) : null}
              <label className="form-label" htmlFor="security-cors-origins">
                允许的来源
                <span className="form-hint-inline">每行一个 Origin，如 https://app.example.com</span>
              </label>
              <textarea
                id="security-cors-origins"
                className="code config-module-text form-control"
                rows={3}
                spellCheck={false}
                value={form.security_cors_origins}
                onChange={(e) => patch((n) => { n.security_cors_origins = e.target.value })}
              />
              <FormField
                label="允许的 HTTP 方法"
                keyName="security.cors.methods"
                hint="逗号分隔；留空为 GET, POST, PUT, PATCH, DELETE, OPTIONS"
                value={form.security_cors_methods}
                onChange={(e) => patch((n) => { n.security_cors_methods = e.target.value })}
              />
              <FormField
                label="允许的请求头"
                keyName="security.cors.headers"
                hint="逗号分隔；留空为 Authorization, Content-Type 等常用头"
                value={form.security_cors_headers}
                onChange={(e) => patch((n) => { n.security_cors_headers = e.target.value })}
              />
              <FormCheckbox
                label="允许携带 Cookie / 凭证"
                checked={form.security_cors_credentials}
                onChange={(v) => patch((n) => { n.security_cors_credentials = v })}
              />
              <FormField
                label="预检结果缓存（秒）"
                keyName="security.cors.max_age"
                type="number"
                value={form.security_cors_max_age || ''}
                onChange={(e) => patch((n) => { n.security_cors_max_age = Number(e.target.value) })}
              />
            </SecurityFeatureGroup>
          </FormSection>
        </div>
      ) : null}

      {!embedded ? (
        <p className="form-hint">
          全局基线在配置模块「安全」中编辑；Host / Path 级在此或规则编辑器中覆盖。预检 OPTIONS 由 Ingress 直接响应。
        </p>
      ) : null}
    </FormSection>
  )
}

function isActiveProfile(profile: string): profile is keyof typeof SECURITY_PROFILE_DEFAULTS {
  return profile in SECURITY_PROFILE_DEFAULTS
}
