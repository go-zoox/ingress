import { useState } from 'react'
import { CollapsibleFormSection, FormCheckbox, FormField, FormGrid, FormSection } from '../Form'
import { EditableStringList } from '../EditableStringList'
import {
  buildIPAllowlistYAML,
  evaluateAllowlist,
  ipAllowlistEntriesValid,
  resolveClientIP,
  type IPAllowlistYAMLInput,
} from '../../lib/ipAllowlist'
import type { RuleForm } from '../../lib/configEntities'

const INTRANET_PRESETS = ['10.0.0.0/8', '172.16.0.0/12', '192.168.0.0/16', '127.0.0.1/32']

type TrialResult = {
  ip: string
  source: string
  allowed: boolean
  reason: string
}

/**
 * Per-route IP allowlist editor. Persists to rules[].waf.allow / deny; the
 * backend WAF IP phase enforces it and returns 403 on block.
 */
export function IPAllowlistFormFields({
  form,
  onChange,
  trustedProxies,
}: {
  form: RuleForm
  onChange: (next: RuleForm) => void
  trustedProxies: string[]
}) {
  const [advancedOpen, setAdvancedOpen] = useState(false)
  const [testRemote, setTestRemote] = useState('10.0.0.5')
  const [testXff, setTestXff] = useState('203.0.113.7, 10.0.0.2')
  const [trial, setTrial] = useState<TrialResult | null>(null)

  const allow = form.waf_ip_allow
  const deny = form.waf_ip_deny
  const patch = (next: Partial<RuleForm>) => onChange({ ...form, ...next })

  const empty = form.waf_ip_enabled && allow.length === 0 && deny.length === 0
  const invalid = !ipAllowlistEntriesValid(allow) || !ipAllowlistEntriesValid(deny)
  const yamlInput: IPAllowlistYAMLInput = {
    host: form.host,
    enabled: form.waf_ip_enabled,
    allow,
    deny,
    signatures: form.waf_ip_signatures,
  }

  const addPreset = (cidr: string) => {
    if (allow.includes(cidr)) return
    patch({ waf_ip_allow: [...allow, cidr] })
  }

  const runTrial = () => {
    const resolved = resolveClientIP(testRemote, testXff, trustedProxies)
    const decision = evaluateAllowlist(
      form.waf_ip_enabled ? allow : [],
      form.waf_ip_enabled ? deny : [],
      resolved.ip,
    )
    setTrial({ ...resolved, ...decision })
  }

  const trustLabel =
    trustedProxies.length > 0
      ? `受信代理优先 · ${trustedProxies.length} 个网段（XFF 仅受信来源采信）`
      : '仅 RemoteAddr（不采信 X-Forwarded-For）'

  return (
    <FormGrid columns={1}>
      <FormCheckbox
        label="启用 IP 白名单（rules[].waf.allow，由 WAF IP 阶段执行）"
        checked={form.waf_ip_enabled}
        onChange={(v) => patch({ waf_ip_enabled: v })}
      />
      <p className="form-hint">
        非空时仅名单内的来源 IP 可访问该域名；留空表示不限制（默认）。deny 优先于 allow。命中拦截返回 403。
      </p>

      {form.waf_ip_enabled && (
        <>
          {empty && (
            <p className="form-hint form-hint-warn">
              白名单已启用但列表为空：保存会被禁用。请至少添加一个允许的 IP / CIDR，或关闭白名单。
            </p>
          )}
          {invalid && (
            <p className="form-hint form-hint-warn">
              存在非法 IP / CIDR。示例：10.0.0.0/8、203.0.113.7/32、fc00::/7。
            </p>
          )}

          <EditableStringList
            title="允许的来源 IP / CIDR"
            showCount
            items={allow}
            valueLabel="IP / CIDR"
            fieldKeyName="rules[].waf.allow[]"
            emptyHint="暂无允许名单"
            placeholder="10.0.0.0/8"
            addTitle="添加允许的 IP / CIDR"
            editTitle="编辑允许的 IP / CIDR"
            hint="支持单个 IP 与 CIDR（IPv4 / IPv6）。非空即白名单生效。"
            onChange={(items) => patch({ waf_ip_allow: items })}
          />
          <div className="ip-allowlist-presets">
            <span>快速添加内网网段：</span>
            {INTRANET_PRESETS.map((cidr) => (
              <button
                type="button"
                key={cidr}
                className="btn btn-ghost btn-sm"
                onClick={() => addPreset(cidr)}
              >
                {cidr}
              </button>
            ))}
          </div>

          <CollapsibleFormSection
            title="高级：拒绝列表与 WAF 攻击签名"
            open={advancedOpen}
            onOpenChange={setAdvancedOpen}
          >
            <EditableStringList
              title="拒绝的来源 IP / CIDR（deny，优先于 allow）"
              showCount
              items={deny}
              valueLabel="IP / CIDR"
              fieldKeyName="rules[].waf.deny[]"
              emptyHint="暂无拒绝名单（一般留空）"
              placeholder="203.0.113.66/32"
              addTitle="添加拒绝的 IP / CIDR"
              editTitle="编辑拒绝的 IP / CIDR"
              hint="用于临时封禁：即使命中 allow 也会被拒绝。"
              onChange={(items) => patch({ waf_ip_deny: items })}
            />
            <FormCheckbox
              label="同时启用 WAF 攻击签名（内置规则）"
              checked={form.waf_ip_signatures}
              onChange={(v) => patch({ waf_ip_signatures: v })}
            />
            <p className="form-hint">
              关闭时写入 <code>disable_builtin: true</code>，该域名只做 IP 准入，避免内置签名误伤白名单用户；开启后该域名同时进行攻击检测。
            </p>
          </CollapsibleFormSection>

          <FormSection title="客户端 IP 判定与拦截响应">
            <p className="form-hint">
              客户端 IP 判定继承全局 WAF：{trustLabel}。可在「WAF」模块配置 <code>trusted_proxies</code>。
            </p>
            <p className="form-hint">
              拦截响应固定 403，复用 <code>error_pages</code> 的 403 页面；访问日志标记 <code>waf_block=1</code>，WAF 事件记录 <code>rule=ip allow</code>。
            </p>
          </FormSection>

          <FormSection title="拦截试算">
            <p className="form-hint">用一组来源地址验证当前白名单（与后端算法一致：受信代理取 XFF 右起第一个非受信地址）。</p>
            <FormField
              label="RemoteAddr（TCP 来源）"
              value={testRemote}
              onChange={(e) => setTestRemote(e.target.value)}
            />
            <FormField
              label="X-Forwarded-For"
              value={testXff}
              onChange={(e) => setTestXff(e.target.value)}
            />
            <div className="ip-allowlist-presets">
              <button type="button" className="btn btn-sm btn-primary" onClick={runTrial}>
                试算
              </button>
            </div>
            {trial && (
              <div className={`ip-allowlist-result ${trial.allowed ? 'ip-allowlist-result--ok' : 'ip-allowlist-result--deny'}`}>
                <div className="ip-allowlist-result-title">
                  {trial.allowed ? '✓ 放行' : '✕ 拒绝 403'}
                  <span className={`badge ${trial.allowed ? 'badge-exact' : 'badge-block'}`}>
                    {trial.allowed ? '200' : '403'}
                  </span>
                </div>
                <div>判定 IP：<code>{trial.ip}</code></div>
                <div>取址来源：{trial.source}</div>
                <div>判定依据：{trial.reason}</div>
              </div>
            )}
          </FormSection>

          <FormSection title="将写入 ingress.yaml">
            <pre className="ip-allowlist-yaml">{buildIPAllowlistYAML(yamlInput)}</pre>
          </FormSection>
        </>
      )}
    </FormGrid>
  )
}
