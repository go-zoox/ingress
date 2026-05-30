import { useEffect, useState } from 'react'
import {
  CollapsibleFormSection,
  FormField,
  FormGrid,
  FormItem,
  FormTextareaField,
} from '../Form'
import { MaintenanceHostListEditor } from './MaintenanceHostListEditor'
import type { GlobalMaintenanceForm } from '../../lib/maintenance'
import { globalMaintenanceSectionOpen } from '../../lib/maintenance'

const HOST_TOOLTIP =
  '精确域名、*.wildcard.example.com，或 Go 正则。每个域名可单独设置维护时间；留空时间表示该域名始终处于维护（匹配时）。'

type SectionKey = 'hosts' | 'response' | 'statusApi' | 'bypass'

type SectionOpenState = Record<SectionKey, boolean>

function emptySectionOpen(): SectionOpenState {
  return { hosts: false, response: false, statusApi: false, bypass: false }
}

export function GlobalMaintenanceFormFields({
  form,
  onChange,
  idPrefix = '',
  formRevision = '',
}: {
  form: GlobalMaintenanceForm
  onChange: (next: GlobalMaintenanceForm) => void
  idPrefix?: string
  /** Changes when config is loaded/saved from server — re-syncs section switches. */
  formRevision?: string
}) {
  const [sectionsOpen, setSectionsOpen] = useState<SectionOpenState>(emptySectionOpen)

  useEffect(() => {
    setSectionsOpen(globalMaintenanceSectionOpen(form))
  }, [formRevision])

  const setSectionOpen = (key: SectionKey, open: boolean) => {
    setSectionsOpen((prev) => ({ ...prev, [key]: open }))
  }

  const patch = (fn: (next: GlobalMaintenanceForm) => void) => {
    const next = { ...form }
    fn(next)
    onChange(next)
  }

  return (
    <FormGrid columns={1}>
      <CollapsibleFormSection
        title="全局维护域名 maintenance.hosts"
        open={sectionsOpen.hosts}
        onOpenChange={(open) => setSectionOpen('hosts', open)}
      >
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
      </CollapsibleFormSection>

      <CollapsibleFormSection
        title="默认响应"
        open={sectionsOpen.response}
        onOpenChange={(open) => setSectionOpen('response', open)}
      >
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
        <FormItem
          label="维护响应头 response_header"
          hint="作用于所有维护 503（业务请求与维护中的状态 API）；留空使用默认 X-Ingress-Maintenance: true"
        >
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
      </CollapsibleFormSection>

      <CollapsibleFormSection
        title="状态 API"
        open={sectionsOpen.statusApi}
        onOpenChange={(open) => setSectionOpen('statusApi', open)}
      >
        <p className="form-hint">
          仅配置维护状态探测端点（路径与 JSON 响应体）；与上方维护响应头无关。
        </p>
        <FormField
          label="路径 status_path"
          keyName={`${idPrefix}maintenance.status_path`}
          hint="留空使用默认 /_/ingress/status"
          placeholder="/_/ingress/status"
          value={form.maintenance_status_path}
          onChange={(e) => patch((n) => { n.maintenance_status_path = e.target.value })}
        />
        <FormField
          label="Content-Type status_response.content_type"
          keyName={`${idPrefix}maintenance.status_response.content_type`}
          hint="留空使用 application/json; charset=utf-8"
          placeholder="application/json; charset=utf-8"
          value={form.maintenance_status_response_content_type}
          onChange={(e) => patch((n) => { n.maintenance_status_response_content_type = e.target.value })}
        />
        <p className="form-hint">
          响应体 JSON 模板占位符：${'{'}host{'}'}、${'{'}title{'}'}、${'{'}subtitle{'}'}、${'{'}retry_after{'}'}（裸数字）、
          ${'{'}maintenance_header_name{'}'}、${'{'}maintenance_header_value{'}'}、${'{'}status{'}'}（ok | maintenance）。字符串写在引号内。
        </p>
        <FormTextareaField
          label="正常 ok status_response.ok"
          keyName={`${idPrefix}maintenance.status_response.ok`}
          hint={'留空使用内置 {"status":"ok"}'}
          full
          mono
          rows={4}
          spellCheck={false}
          placeholder='{"ready":true,"host":"${host}"}'
          value={form.maintenance_status_response_ok}
          onChange={(e) => patch((n) => { n.maintenance_status_response_ok = e.target.value })}
        />
        <FormTextareaField
          label="维护 maintenance status_response.maintenance"
          keyName={`${idPrefix}maintenance.status_response.maintenance`}
          hint={'留空使用内置 {"status":"maintenance","title":"…","subtitle":"…","retry_after":300,"maintenance_header_name":"…","maintenance_header_value":"…"}'}
          full
          mono
          rows={5}
          spellCheck={false}
          placeholder='{"status":"maintenance","title":"${title}","retry_after":${retry_after}}'
          value={form.maintenance_status_response_maintenance}
          onChange={(e) => patch((n) => { n.maintenance_status_response_maintenance = e.target.value })}
        />
      </CollapsibleFormSection>

      <CollapsibleFormSection
        title="Bypass（全局默认）"
        open={sectionsOpen.bypass}
        onOpenChange={(open) => setSectionOpen('bypass', open)}
      >
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
      </CollapsibleFormSection>
    </FormGrid>
  )
}
