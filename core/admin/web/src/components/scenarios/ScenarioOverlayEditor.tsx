import {
  CollapsibleFormSection,
  FormCheckbox,
  FormField,
  FormGrid,
  FormSection,
  FormSelectField,
} from '../Form'
import { BackendCacheFormFields } from '../config/BackendCacheFormFields'
import { GlobalMaintenanceFormFields } from '../config/GlobalMaintenanceFormFields'
import { RateLimitFormFields } from '../config/RateLimitFormFields'
import { SecurityFormFields } from '../config/SecurityFormFields'
import { WafRulesEditor } from '../config/WafRulesEditor'
import {
  bool,
  obj,
  setBool,
} from '../../lib/ingressModuleForms'
import {
  type ScenarioCacheOverlayForm,
  type ScenarioItemForm,
  type ScenarioOverlaySections,
  emptyScenarioRulePatch,
  maintenanceFormFromOverlay,
  patchMaintenanceOverlay,
  patchSecurityOverlay,
  securityFormFromOverlay,
} from '../../lib/scenarios'
import {
  WAF_GLOBAL_MODE_OPTIONS,
  type WAFGlobalMode,
  wafGlobalModeFromLogOnly,
} from '../../lib/wafAction'

function ScenarioCacheOverlayFields({
  form,
  onChange,
}: {
  form: ScenarioCacheOverlayForm
  onChange: (next: ScenarioCacheOverlayForm) => void
}) {
  const patch = (fn: (n: ScenarioCacheOverlayForm) => void) => {
    const next = { ...form }
    fn(next)
    onChange(next)
  }
  return (
    <FormGrid>
      <FormField
        label="TTL（秒）"
        type="number"
        value={form.ttl}
        onChange={(e) => patch((n) => { n.ttl = Number(e.target.value) })}
      />
      <FormField
        label="Redis 主机"
        value={form.host}
        onChange={(e) => patch((n) => { n.host = e.target.value })}
      />
      <FormField
        label="Redis 端口"
        type="number"
        value={form.port}
        onChange={(e) => patch((n) => { n.port = Number(e.target.value) })}
      />
      <FormField
        label="键前缀"
        value={form.prefix}
        onChange={(e) => patch((n) => { n.prefix = e.target.value })}
      />
      <FormField
        label="用户名"
        value={form.username}
        onChange={(e) => patch((n) => { n.username = e.target.value })}
      />
      <FormField
        label="密码"
        type="password"
        value={form.password}
        onChange={(e) => patch((n) => { n.password = e.target.value })}
      />
      <FormField
        label="DB"
        type="number"
        value={form.db}
        onChange={(e) => patch((n) => { n.db = Number(e.target.value) })}
      />
    </FormGrid>
  )
}

function ScenarioWafOverlayFields({
  waf,
  onChange,
}: {
  waf: Record<string, unknown>
  onChange: (next: Record<string, unknown>) => void
}) {
  const block = { ...waf }
  const patchWaf = (fn: (n: Record<string, unknown>) => void) => {
    const next = { ...block }
    fn(next)
    onChange(next)
  }
  return (
    <FormGrid columns={1}>
      <FormCheckbox
        label="启用 WAF overlay"
        checked={bool(block.enabled)}
        onChange={(v) => patchWaf((n) => setBool(n, 'enabled', v))}
      />
      <FormSelectField
        label="全局处置"
        value={wafGlobalModeFromLogOnly(bool(block.log_only))}
        onChange={(e) => {
          const mode = e.target.value as WAFGlobalMode
          patchWaf((n) => {
            if (mode === 'audit') n.log_only = true
            else delete n.log_only
          })
        }}
      >
        {WAF_GLOBAL_MODE_OPTIONS.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </FormSelectField>
      <WafRulesEditor
        doc={{ waf: block }}
        onChange={(d) => onChange({ ...obj(d.waf) })}
      />
    </FormGrid>
  )
}

type Props = {
  form: ScenarioItemForm
  onChange: (next: ScenarioItemForm) => void
  hostOptions: string[]
}

export function ScenarioOverlayEditor({ form, onChange, hostOptions }: Props) {
  const setSection = (key: keyof ScenarioOverlaySections, enabled: boolean) => {
    const next = { ...form, sections: { ...form.sections, [key]: enabled } }
    if (key === 'rules' && enabled && next.rule_patches.length === 0) {
      next.rule_patches = [emptyScenarioRulePatch(hostOptions[0] ?? '')]
    }
    if (key === 'waf' && enabled && Object.keys(next.waf).length === 0) {
      next.waf = { enabled: false }
    }
    onChange(next)
  }

  const patch = (fn: (n: ScenarioItemForm) => void) => {
    const next = { ...form }
    fn(next)
    onChange(next)
  }

  return (
    <div className="scenario-overlay-editor">
      <FormSection title="Overlay 差异配置">
        <p className="form-hint">
          勾选要在此场景相对基线覆盖的模块。基线路由写在根 <code>rules</code>，此处仅写差异。
        </p>
      </FormSection>

      <CollapsibleFormSection
        title="全局 cache"
        open={form.sections.cache}
        onOpenChange={(open) => setSection('cache', open)}
      >
        <ScenarioCacheOverlayFields
          form={form.cache}
          onChange={(cache) => patch((n) => { n.cache = cache })}
        />
      </CollapsibleFormSection>

      <CollapsibleFormSection
        title="全局限流 rate_limit"
        open={form.sections.rate_limit}
        onOpenChange={(open) => setSection('rate_limit', open)}
      >
        <RateLimitFormFields
          form={form.rate_limit}
          onChange={(rate_limit) => patch((n) => { n.rate_limit = rate_limit })}
          title="rate_limit overlay"
        />
      </CollapsibleFormSection>

      <CollapsibleFormSection
        title="路由 rules（按 Host 覆盖 backend）"
        open={form.sections.rules}
        onOpenChange={(open) => setSection('rules', open)}
      >
        {form.rule_patches.map((rp, idx) => (
          <div key={idx} className="form-section scenario-rule-patch">
            <div className="scenario-rule-patch-head">
              <strong>Host 规则 #{idx + 1}</strong>
              <button
                type="button"
                className="btn btn-sm"
                onClick={() => patch((n) => {
                  n.rule_patches = n.rule_patches.filter((_, i) => i !== idx)
                })}
              >
                删除
              </button>
            </div>
            {hostOptions.length > 0 ? (
              <FormSelectField
                label="Host"
                value={rp.host}
                onChange={(e) => patch((n) => {
                  n.rule_patches[idx] = { ...n.rule_patches[idx], host: e.target.value }
                })}
              >
                <option value="">选择 Host</option>
                {hostOptions.map((h) => (
                  <option key={h} value={h}>{h}</option>
                ))}
              </FormSelectField>
            ) : (
              <FormField
                label="Host"
                hint="须与基线 rules[].host 完全一致"
                value={rp.host}
                onChange={(e) => patch((n) => {
                  n.rule_patches[idx] = { ...n.rule_patches[idx], host: e.target.value }
                })}
              />
            )}
            <BackendCacheFormFields
              form={rp.backend}
              embedded
              idPrefix={`scenario-${idx}-`}
              onChange={(backend) => patch((n) => {
                n.rule_patches[idx] = { ...n.rule_patches[idx], backend }
              })}
            />
          </div>
        ))}
        <button
          type="button"
          className="btn btn-sm"
          onClick={() => patch((n) => {
            n.rule_patches = [...n.rule_patches, emptyScenarioRulePatch(hostOptions[0] ?? '')]
          })}
        >
          + 添加 Host 覆盖
        </button>
      </CollapsibleFormSection>

      <CollapsibleFormSection
        title="WAF"
        open={form.sections.waf}
        onOpenChange={(open) => setSection('waf', open)}
      >
        <ScenarioWafOverlayFields
          waf={form.waf}
          onChange={(waf) => patch((n) => { n.waf = waf })}
        />
      </CollapsibleFormSection>

      <CollapsibleFormSection
        title="维护 maintenance"
        open={form.sections.maintenance}
        onOpenChange={(open) => setSection('maintenance', open)}
      >
        <GlobalMaintenanceFormFields
          form={maintenanceFormFromOverlay(form.maintenance)}
          onChange={(maint) => patch((n) => {
            n.maintenance = patchMaintenanceOverlay(maint)
          })}
        />
      </CollapsibleFormSection>

      <CollapsibleFormSection
        title="安全 security"
        open={form.sections.security}
        onOpenChange={(open) => setSection('security', open)}
      >
        <SecurityFormFields
          form={securityFormFromOverlay(form.security)}
          onChange={(sec) => patch((n) => {
            n.security = patchSecurityOverlay(sec)
          })}
          title="security overlay"
        />
      </CollapsibleFormSection>
    </div>
  )
}
