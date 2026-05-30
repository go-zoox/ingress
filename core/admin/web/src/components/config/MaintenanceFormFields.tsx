import {
  FormCheckbox,
  FormField,
  FormGrid,
  FormItem,
  FormSection,
  FormSelectField,
} from '../Form'
import type { BackendForm } from '../../lib/configEntities'
import type { MaintenanceScope } from '../../lib/maintenance'
import { MaintenanceHostListEditor } from './MaintenanceHostListEditor'

export function MaintenanceFormFields<T extends BackendForm>({
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

  const scope = (form.maintenance_scope || 'all') as MaintenanceScope

  return (
    <FormSection title={embedded ? undefined : '维护模式 service.maintenance'}>
      <FormGrid columns={1}>
        <FormCheckbox
          label="启用规则级维护（503，跳过 auth）"
          checked={form.maintenance_enabled}
          onChange={(v) => patch((n) => { n.maintenance_enabled = v })}
        />
        {form.maintenance_enabled ? (
          <>
            <FormSelectField
              label="作用范围 scope"
              keyName={`${idPrefix}service.maintenance.scope`}
              hint="all：该规则匹配的所有 Host；listed：仅下方列出的 Host（须属于本规则）"
              value={scope}
              onChange={(e) =>
                patch((n) => {
                  n.maintenance_scope = e.target.value as MaintenanceScope
                })
              }
            >
              <option value="all">all — 规则下全部 Host</option>
              <option value="listed">listed — 仅列出的 Host</option>
            </FormSelectField>
            {scope === 'listed' ? (
              <MaintenanceHostListEditor
                title="维护 Host"
                showCount
                titleTooltip="须能被本规则的 host 模式匹配；每个 Host 可单独设置维护时间"
                items={form.maintenance_host_entries}
                fieldKeyPrefix={`${idPrefix}service.maintenance.hosts[]`}
                emptyHint="scope=listed 时至少添加一个 Host"
                addTitle="添加维护 Host"
                editTitle="编辑维护 Host"
                onChange={(items) => patch((n) => { n.maintenance_host_entries = items })}
              />
            ) : null}
            <FormField
              label="Retry-After（秒）"
              keyName={`${idPrefix}service.maintenance.retry_after`}
              type="number"
              hint="可选；覆盖全局默认值"
              value={form.maintenance_retry_after}
              onChange={(e) => patch((n) => { n.maintenance_retry_after = Number(e.target.value) })}
            />
            <FormField
              label="标题 title"
              keyName={`${idPrefix}service.maintenance.title`}
              hint="可选；覆盖全局 / 内置 503 标题"
              value={form.maintenance_title}
              onChange={(e) => patch((n) => { n.maintenance_title = e.target.value })}
            />
            <FormField
              label="说明 subtitle"
              keyName={`${idPrefix}service.maintenance.subtitle`}
              value={form.maintenance_subtitle}
              onChange={(e) => patch((n) => { n.maintenance_subtitle = e.target.value })}
            />
            <FormField
              label="Bypass 路径 bypass.paths"
              keyName={`${idPrefix}service.maintenance.bypass.paths`}
              hint="逗号分隔；与全局 bypass 合并（OR）"
              value={form.maintenance_bypass_paths}
              onChange={(e) => patch((n) => { n.maintenance_bypass_paths = e.target.value })}
            />
            <FormField
              label="Bypass IP bypass.allow_ips"
              keyName={`${idPrefix}service.maintenance.bypass.allow_ips`}
              hint="逗号分隔 IP 或 CIDR"
              value={form.maintenance_bypass_allow_ips}
              onChange={(e) => patch((n) => { n.maintenance_bypass_allow_ips = e.target.value })}
            />
            <FormItem label="Bypass 请求头 bypass.header">
              <div className="form-list-row">
                <FormField
                  label="Header 名"
                  keyName={`${idPrefix}service.maintenance.bypass.header.name`}
                  value={form.maintenance_bypass_header_name}
                  onChange={(e) => patch((n) => { n.maintenance_bypass_header_name = e.target.value })}
                />
                <FormField
                  label="Header 值"
                  keyName={`${idPrefix}service.maintenance.bypass.header.value`}
                  value={form.maintenance_bypass_header_value}
                  onChange={(e) => patch((n) => { n.maintenance_bypass_header_value = e.target.value })}
                />
              </div>
            </FormItem>
          </>
        ) : null}
      </FormGrid>
    </FormSection>
  )
}
