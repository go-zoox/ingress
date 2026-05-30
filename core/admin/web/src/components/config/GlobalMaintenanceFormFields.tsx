import { FormField, FormGrid, FormItem, FormSection } from '../Form'
import { MaintenanceHostListEditor } from './MaintenanceHostListEditor'
import type { GlobalMaintenanceForm } from '../../lib/maintenance'

const HOST_TOOLTIP =
  '精确域名、*.wildcard.example.com，或 Go 正则。每个域名可单独设置维护时间；留空时间表示该域名始终处于维护（匹配时）。'

export function GlobalMaintenanceFormFields({
  form,
  onChange,
  idPrefix = '',
}: {
  form: GlobalMaintenanceForm
  onChange: (next: GlobalMaintenanceForm) => void
  idPrefix?: string
}) {
  const patch = (fn: (next: GlobalMaintenanceForm) => void) => {
    const next = { ...form }
    fn(next)
    onChange(next)
  }

  return (
    <FormGrid columns={1}>
      <FormSection title="全局维护域名 maintenance.hosts">
        <MaintenanceHostListEditor
          title="维护域名"
          showCount
          titleTooltip={HOST_TOOLTIP}
          items={form.maintenance_host_entries}
          fieldKeyPrefix={`${idPrefix}maintenance.hosts[]`}
          emptyHint="未登记域名；留空表示不启用全局维护"
          addTitle="添加维护域名"
          editTitle="编辑维护域名"
          onChange={(items) => patch((n) => { n.maintenance_host_entries = items })}
        />
      </FormSection>

      <FormSection title="默认响应">
        <FormField
          label="状态 API 路径 status_path"
          keyName={`${idPrefix}maintenance.status_path`}
          hint="留空使用默认 /_/ingress/status；JSON 维护状态探测端点"
          placeholder="/_/ingress/status"
          value={form.maintenance_status_path}
          onChange={(e) => patch((n) => { n.maintenance_status_path = e.target.value })}
        />
        <FormField
          label="Retry-After（秒）"
          keyName={`${idPrefix}maintenance.retry_after`}
          type="number"
          hint="可选；写入 Retry-After 响应头"
          value={form.maintenance_retry_after}
          onChange={(e) => patch((n) => { n.maintenance_retry_after = Number(e.target.value) })}
        />
        <FormField
          label="标题 title"
          keyName={`${idPrefix}maintenance.title`}
          hint="规则级未覆盖时使用"
          value={form.maintenance_title}
          onChange={(e) => patch((n) => { n.maintenance_title = e.target.value })}
        />
        <FormField
          label="说明 subtitle"
          keyName={`${idPrefix}maintenance.subtitle`}
          value={form.maintenance_subtitle}
          onChange={(e) => patch((n) => { n.maintenance_subtitle = e.target.value })}
        />
        <FormItem label="维护响应头 response_header" hint="留空使用默认 X-Ingress-Maintenance: true">
          <div className="form-list-row">
            <FormField
              label="Header 名"
              keyName={`${idPrefix}maintenance.response_header.name`}
              placeholder="X-Ingress-Maintenance"
              value={form.maintenance_response_header_name}
              onChange={(e) => patch((n) => { n.maintenance_response_header_name = e.target.value })}
            />
            <FormField
              label="Header 值"
              keyName={`${idPrefix}maintenance.response_header.value`}
              placeholder="true"
              value={form.maintenance_response_header_value}
              onChange={(e) => patch((n) => { n.maintenance_response_header_value = e.target.value })}
            />
          </div>
        </FormItem>
      </FormSection>

      <FormSection title="Bypass（全局默认）">
        <p className="form-hint">与规则级 bypass 合并（OR）；任一匹配即放行。</p>
        <FormField
          label="Bypass 路径 bypass.paths"
          keyName={`${idPrefix}maintenance.bypass.paths`}
          hint="逗号分隔；支持精确路径或前缀*（如 /healthz, /internal/*）"
          value={form.maintenance_bypass_paths}
          onChange={(e) => patch((n) => { n.maintenance_bypass_paths = e.target.value })}
        />
        <FormField
          label="Bypass IP bypass.allow_ips"
          keyName={`${idPrefix}maintenance.bypass.allow_ips`}
          hint="逗号分隔 IP 或 CIDR"
          value={form.maintenance_bypass_allow_ips}
          onChange={(e) => patch((n) => { n.maintenance_bypass_allow_ips = e.target.value })}
        />
        <FormItem label="Bypass 请求头 bypass.header">
          <div className="form-list-row">
            <FormField
              label="Header 名"
              keyName={`${idPrefix}maintenance.bypass.header.name`}
              value={form.maintenance_bypass_header_name}
              onChange={(e) => patch((n) => { n.maintenance_bypass_header_name = e.target.value })}
            />
            <FormField
              label="Header 值"
              keyName={`${idPrefix}maintenance.bypass.header.value`}
              value={form.maintenance_bypass_header_value}
              onChange={(e) => patch((n) => { n.maintenance_bypass_header_value = e.target.value })}
            />
          </div>
        </FormItem>
      </FormSection>
    </FormGrid>
  )
}
