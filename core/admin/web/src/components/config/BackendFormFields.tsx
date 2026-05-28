import { FormGrid } from '../Form'
import { AuthFormFields } from './AuthFormFields'
import { BackendCacheFormFields } from './BackendCacheFormFields'
import { BackendCoreFormFields } from './BackendCoreFormFields'
import { HealthCheckFormFields } from './HealthCheckFormFields'
import type { BackendForm } from '../../lib/configEntities'

/** @deprecated Use RuleEntityFormSections for partitioned editing. */
export function BackendFormFields<T extends BackendForm>({
  form,
  onChange,
  idPrefix = '',
  variant = 'host',
}: {
  form: T
  onChange: (next: T) => void
  idPrefix?: string
  variant?: 'host' | 'path'
}) {
  return (
    <>
      <BackendCoreFormFields form={form} onChange={onChange} idPrefix={idPrefix} variant={variant} />
      <BackendCacheFormFields form={form} onChange={onChange} idPrefix={idPrefix} />
      {form.backend_type === 'service' && (
        <AuthFormFields form={form} onChange={onChange} idPrefix={idPrefix} />
      )}
      {form.backend_type === 'service' && (
        <HealthCheckFormFields form={form} onChange={onChange} idPrefix={idPrefix} />
      )}
    </>
  )
}

export function BackendFormGrid<T extends BackendForm>({
  form,
  onChange,
  idPrefix = '',
  variant = 'host',
}: {
  form: T
  onChange: (next: T) => void
  idPrefix?: string
  variant?: 'host' | 'path'
}) {
  return (
    <FormGrid columns={1}>
      <BackendFormFields form={form} onChange={onChange} idPrefix={idPrefix} variant={variant} />
    </FormGrid>
  )
}

export function backendFormWide(_form: BackendForm): boolean {
  return true
}
