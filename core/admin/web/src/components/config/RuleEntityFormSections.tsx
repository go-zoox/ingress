import { FormGrid, FormField, FormSelectField } from '../Form'
import { AuthFormFields } from './AuthFormFields'
import { BackendCacheFormFields } from './BackendCacheFormFields'
import { BackendCoreFormFields } from './BackendCoreFormFields'
import { BackendTypeTabs } from './BackendTypeTabs'
import { EntityFormLayout, EntityFormUnavailable, type EntityFormSection } from './EntityFormLayout'
import { HealthCheckFormFields } from './HealthCheckFormFields'
import { RateLimitFormFields } from './RateLimitFormFields'
import { RouteSecurityFormFields } from './RouteSecurityFormFields'
import { MaintenanceFormFields } from './MaintenanceFormFields'
import { ServiceRequestFormFields } from './ServiceRequestFormFields'
import type { BackendForm, PathForm, RuleForm } from '../../lib/configEntities'
import { securityLayerBadge, serviceRequestConfigured } from '../../lib/configEntities'
import type { ServiceForm } from '../../lib/services'

export type RuleEntitySectionId =
  | 'basic'
  | 'backend'
  | 'paths'
  | 'health'
  | 'auth'
  | 'maintenance'
  | 'upstream'
  | 'cache'
  | 'security'
  | 'rate_limit'

const SECTION_COPY: Record<RuleEntitySectionId, { label: string; description: string }> = {
  basic: {
    label: '基础信息',
    description: 'Host 匹配与规则优先级（列表顺序即匹配顺序）。',
  },
  backend: {
    label: 'Backend',
    description: 'Host 级默认 upstream / handler / redirect 配置。',
  },
  paths: {
    label: 'Path 规则',
    description: '按 path 前缀覆盖 Host 级 backend；保存 Host 后在列表「Paths」中编辑。',
  },
  health: {
    label: '健康检查',
    description: '上游 service 健康探测（仅 service backend）。',
  },
  auth: {
    label: '认证授权',
    description: '上游 service 访问认证（仅 service backend）。',
  },
  maintenance: {
    label: '维护模式',
    description: 'Host 级上游维护；开启后整站返回 503 并跳过 auth。',
  },
  upstream: {
    label: '上游转发',
    description: 'request / response 改写（Host、路径、头、超时等；仅 service backend）。',
  },
  cache: {
    label: '响应缓存',
    description: 'HTTP 响应缓存 backend.cache（service / handler / redirect）。',
  },
  security: {
    label: '安全',
    description: '覆盖全局 security: 的响应头与 CORS（Host 或 Path 级）。',
  },
  rate_limit: {
    label: '路由限流',
    description: 'rules[].rate_limit 按 Host 规则限流。',
  },
}

function serviceOnly(form: BackendForm): boolean {
  return form.backend_type === 'service'
}

function ruleSections(form: BackendForm & { paths?: PathForm[] }, includePaths: boolean): EntityFormSection[] {
  const ids: RuleEntitySectionId[] = includePaths
    ? ['basic', 'backend', 'paths', 'health', 'auth', 'maintenance', 'upstream', 'cache', 'security', 'rate_limit']
    : ['basic', 'backend', 'health', 'auth', 'maintenance', 'upstream', 'cache', 'security']

  return ids.map((id) => {
    const copy = SECTION_COPY[id]
    const section: EntityFormSection = {
      id,
      label: copy.label,
      description: copy.description,
    }
    if (id === 'paths' && includePaths) {
      const count = form.paths?.length ?? 0
      section.badge = count > 0 ? count : undefined
    }
    if (id === 'cache' && form.cache_enabled) section.badge = '开'
    if (id === 'auth' && form.auth_type) section.badge = form.auth_type
    if (id === 'maintenance' && form.maintenance_enabled) {
      section.badge = form.maintenance_scope === 'listed' ? '部分' : '开'
    }
    if (id === 'health' && form.health_check_enable) section.badge = '开'
    if (id === 'upstream' && serviceRequestConfigured(form)) section.badge = '已配'
    if (id === 'security' && 'security_override' in form) {
      section.badge = securityLayerBadge(form as RuleForm)
    }
    if (id === 'rate_limit' && 'rate_limit_requests' in form) {
      const rl = form as RuleForm
      if (rl.rate_limit_enabled === true || rl.rate_limit_requests > 0) {
        section.badge = '开'
      }
    }
    if ((id === 'health' || id === 'auth' || id === 'upstream' || id === 'maintenance') && !serviceOnly(form)) {
      section.disabled = true
    }
    return section
  })
}

function RulePathsSummary({ count }: { count: number }) {
  if (count === 0) {
    return (
      <p className="form-hint">
        尚未配置 path 规则。未配置时，该 Host 下所有请求使用 Host 级 backend。
        保存后在规则列表点击「Paths」添加 / 编辑。
      </p>
    )
  }
  return (
    <p className="form-hint">
      已配置 <strong>{count}</strong> 条 path 规则。保存 Host 后在列表中点击「Paths」继续编辑顺序与 backend 覆盖。
    </p>
  )
}

type RuleEntityFormSectionsProps =
  | {
      variant: 'rule'
      form: RuleForm
      onChange: (next: RuleForm) => void
      activeSection: RuleEntitySectionId
      onSectionChange: (id: RuleEntitySectionId) => void
      idPrefix?: string
      serviceCatalog?: ServiceForm[]
      serviceFieldMode?: 'manual' | 'catalog-select'
    }
  | {
      variant: 'path'
      form: PathForm
      onChange: (next: PathForm) => void
      activeSection: RuleEntitySectionId
      onSectionChange: (id: RuleEntitySectionId) => void
      idPrefix?: string
      serviceCatalog?: ServiceForm[]
      serviceFieldMode?: 'manual' | 'catalog-select'
    }

function SharedBackendSections<T extends BackendForm>({
  form,
  onChange,
  activeSection,
  idPrefix,
  variant,
  serviceCatalog,
  serviceFieldMode = 'manual',
}: {
  form: T
  onChange: (next: T) => void
  activeSection: RuleEntitySectionId
  idPrefix: string
  variant: 'rule' | 'path'
  serviceCatalog?: ServiceForm[]
  serviceFieldMode?: 'manual' | 'catalog-select'
}) {
  switch (activeSection) {
    case 'backend':
      return (
        <FormGrid columns={1}>
          <BackendCoreFormFields
            form={form}
            onChange={onChange}
            idPrefix={idPrefix}
            variant={variant === 'path' ? 'path' : 'host'}
            serviceCatalog={serviceCatalog}
            serviceFieldMode={serviceFieldMode}
          />
        </FormGrid>
      )
    case 'health':
      if (!serviceOnly(form)) {
        return (
          <EntityFormUnavailable
            title="健康检查仅适用于 service backend。"
            detail="请将 Backend 类型设为 service，或切换到 Backend 分区修改类型。"
          />
        )
      }
      return (
        <FormGrid columns={1}>
          <HealthCheckFormFields form={form} onChange={onChange} idPrefix={idPrefix} embedded />
        </FormGrid>
      )
    case 'auth':
      if (!serviceOnly(form)) {
        return (
          <EntityFormUnavailable
            title="认证授权仅适用于 service backend。"
            detail="handler / redirect backend 不支持 backend.service.auth。"
          />
        )
      }
      return (
        <FormGrid columns={1}>
          <AuthFormFields form={form} onChange={onChange} idPrefix={idPrefix} embedded />
        </FormGrid>
      )
    case 'maintenance':
      if (variant !== 'rule') {
        return (
          <EntityFormUnavailable
            title="维护模式仅适用于 Host 级 service backend。"
            detail="Path 级 backend 不支持 service.maintenance。"
          />
        )
      }
      if (!serviceOnly(form)) {
        return (
          <EntityFormUnavailable
            title="维护模式仅适用于 service backend。"
            detail="handler / redirect backend 不支持 service.maintenance。"
          />
        )
      }
      return (
        <FormGrid columns={1}>
          <MaintenanceFormFields form={form} onChange={onChange} idPrefix={idPrefix} embedded />
        </FormGrid>
      )
    case 'upstream':
      if (!serviceOnly(form)) {
        return (
          <EntityFormUnavailable
            title="上游转发仅适用于 service backend。"
            detail="handler / redirect backend 不支持 backend.service.request / response。"
          />
        )
      }
      return (
        <FormGrid columns={1}>
          <ServiceRequestFormFields
            form={form}
            onChange={onChange}
            idPrefix={idPrefix}
            embedded
            showStripPrefixHint={variant === 'path'}
          />
        </FormGrid>
      )
    case 'cache':
      return (
        <FormGrid columns={1}>
          <BackendCacheFormFields form={form} onChange={onChange} idPrefix={idPrefix} embedded />
        </FormGrid>
      )
    default:
      return null
  }
}

export function RuleEntityFormSections(props: RuleEntityFormSectionsProps) {
  const { form, onChange, activeSection, onSectionChange, variant, idPrefix = '', serviceCatalog, serviceFieldMode = 'manual' } = props
  const sections = ruleSections(form, variant === 'rule')

  const renderSection = () => {
    if (variant === 'rule') {
      const ruleForm = form
      const setRuleForm = onChange
      switch (activeSection) {
        case 'basic':
          return (
            <FormGrid columns={1}>
              <FormField
                label="Host"
                keyName="host"
                value={ruleForm.host}
                onChange={(e) => setRuleForm({ ...ruleForm, host: e.target.value })}
              />
              <FormSelectField
                label="Host 类型"
                keyName="host_type"
                value={ruleForm.host_type}
                onChange={(e) => setRuleForm({ ...ruleForm, host_type: e.target.value })}
              >
                <option value="auto">auto（自动推断）</option>
                <option value="exact">exact</option>
                <option value="wildcard">wildcard</option>
                <option value="regex">regex</option>
              </FormSelectField>
            </FormGrid>
          )
        case 'paths':
          return <RulePathsSummary count={ruleForm.paths.length} />
        case 'rate_limit':
          return (
            <FormGrid columns={1}>
              <RateLimitFormFields
                form={ruleForm}
                onChange={setRuleForm}
                title="路由限流 rules[].rate_limit"
                embedded
              />
            </FormGrid>
          )
        case 'security':
          return (
            <FormGrid columns={1}>
              <RouteSecurityFormFields
                form={ruleForm}
                onChange={setRuleForm}
                layer="rule"
                embedded
              />
            </FormGrid>
          )
        default:
          return (
            <SharedBackendSections
              form={ruleForm}
              onChange={setRuleForm}
              activeSection={activeSection}
              idPrefix={idPrefix}
              variant={variant}
              serviceCatalog={serviceCatalog}
              serviceFieldMode={serviceFieldMode}
            />
          )
      }
    }

    const pathForm = form
    const setPathForm = onChange
    switch (activeSection) {
      case 'basic':
        return (
          <FormGrid columns={1}>
            <FormField
              label="Path 前缀"
              keyName={`${idPrefix}path`}
              hint="如 /api、/v2；匹配最长前缀"
              value={pathForm.path}
              onChange={(e) => setPathForm({ ...pathForm, path: e.target.value })}
            />
          </FormGrid>
        )
      case 'security':
        return (
          <FormGrid columns={1}>
            <RouteSecurityFormFields
              form={pathForm}
              onChange={setPathForm}
              layer="path"
              embedded
            />
          </FormGrid>
        )
      default:
        return (
          <SharedBackendSections
            form={pathForm}
            onChange={setPathForm}
            activeSection={activeSection}
            idPrefix={idPrefix}
            variant={variant}
            serviceCatalog={serviceCatalog}
            serviceFieldMode={serviceFieldMode}
          />
        )
    }
  }

  const handleSectionChange = (id: string) => {
    onSectionChange(id as RuleEntitySectionId)
  }

  const backendSubhead =
    activeSection === 'backend' ? (
      variant === 'rule' ? (
        <BackendTypeTabs
          value={form.backend_type}
          onChange={(backend_type) =>
            (onChange as (next: RuleForm) => void)({ ...(form as RuleForm), backend_type })
          }
        />
      ) : (
        <BackendTypeTabs
          value={form.backend_type}
          onChange={(backend_type) =>
            (onChange as (next: PathForm) => void)({ ...(form as PathForm), backend_type })
          }
        />
      )
    ) : undefined

  return (
    <EntityFormLayout
      sections={sections}
      activeSection={activeSection}
      onSectionChange={handleSectionChange}
      subhead={backendSubhead}
    >
      {renderSection()}
    </EntityFormLayout>
  )
}

export function defaultRuleEntitySection(_variant: 'rule' | 'path'): RuleEntitySectionId {
  return 'basic'
}
