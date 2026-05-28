import { FormGrid, FormField, FormSelectField } from '../Form'
import { AuthFormFields } from './AuthFormFields'
import { BackendCacheFormFields } from './BackendCacheFormFields'
import { BackendCoreFormFields } from './BackendCoreFormFields'
import { EntityFormLayout, EntityFormUnavailable, type EntityFormSection } from './EntityFormLayout'
import { HealthCheckFormFields } from './HealthCheckFormFields'
import { RateLimitFormFields } from './RateLimitFormFields'
import type { BackendForm, PathForm, RuleForm } from '../../lib/configEntities'

export type RuleEntitySectionId =
  | 'basic'
  | 'backend'
  | 'paths'
  | 'health'
  | 'auth'
  | 'cache'
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
  cache: {
    label: '响应缓存',
    description: 'HTTP 响应缓存 backend.cache（service / handler / redirect）。',
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
    ? ['basic', 'backend', 'paths', 'health', 'auth', 'cache', 'rate_limit']
    : ['basic', 'backend', 'health', 'auth', 'cache']

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
    if (id === 'health' && form.health_check_enable) section.badge = '开'
    if (id === 'rate_limit' && 'rate_limit_requests' in form) {
      const rl = form as RuleForm
      if (rl.rate_limit_enabled === true || rl.rate_limit_requests > 0) {
        section.badge = '开'
      }
    }
    if ((id === 'health' || id === 'auth') && !serviceOnly(form)) {
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
    }
  | {
      variant: 'path'
      form: PathForm
      onChange: (next: PathForm) => void
      activeSection: RuleEntitySectionId
      onSectionChange: (id: RuleEntitySectionId) => void
      idPrefix?: string
    }

function SharedBackendSections<T extends BackendForm>({
  form,
  onChange,
  activeSection,
  idPrefix,
  variant,
}: {
  form: T
  onChange: (next: T) => void
  activeSection: RuleEntitySectionId
  idPrefix: string
  variant: 'rule' | 'path'
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
  const { form, onChange, activeSection, onSectionChange, variant, idPrefix = '' } = props
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
        default:
          return (
            <SharedBackendSections
              form={ruleForm}
              onChange={setRuleForm}
              activeSection={activeSection}
              idPrefix={idPrefix}
              variant={variant}
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
      default:
        return (
          <SharedBackendSections
            form={pathForm}
            onChange={setPathForm}
            activeSection={activeSection}
            idPrefix={idPrefix}
            variant={variant}
          />
        )
    }
  }

  const handleSectionChange = (id: string) => {
    onSectionChange(id as RuleEntitySectionId)
  }

  return (
    <EntityFormLayout
      sections={sections}
      activeSection={activeSection}
      onSectionChange={handleSectionChange}
    >
      {renderSection()}
    </EntityFormLayout>
  )
}

export function defaultRuleEntitySection(_variant: 'rule' | 'path'): RuleEntitySectionId {
  return 'basic'
}
