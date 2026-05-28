import { useId, type InputHTMLAttributes, type ReactNode, type SelectHTMLAttributes } from 'react'

export function FormGrid({
  children,
  columns = 2,
}: {
  children: ReactNode
  columns?: 1 | 2
}) {
  return <div className={`form-grid form-grid--${columns}`}>{children}</div>
}

export function FormSection({ title, children }: { title?: string; children: ReactNode }) {
  return (
    <section className="form-section">
      {title ? <h4 className="form-section-title">{title}</h4> : null}
      <div className="form-section-body">{children}</div>
    </section>
  )
}

export function FormItem({
  label,
  keyName,
  hint,
  full,
  htmlFor,
  children,
}: {
  label: string
  keyName?: string
  hint?: string
  full?: boolean
  htmlFor?: string
  children: ReactNode
}) {
  return (
    <div className={`form-item${full ? ' form-item--full' : ''}`}>
      <label className="form-label" htmlFor={htmlFor}>
        <span className="form-label-text">{label}</span>
        {keyName && <code className="form-key">{keyName}</code>}
      </label>
      <div className="form-control-wrap">{children}</div>
      {hint && <p className="form-hint">{hint}</p>}
    </div>
  )
}

type FormInputProps = Omit<InputHTMLAttributes<HTMLInputElement>, 'className'> & {
  className?: string
}

export function FormInput({ className, id, ...props }: FormInputProps) {
  const autoId = useId()
  const inputId = id ?? autoId
  return (
    <input
      id={inputId}
      className={['form-control', className].filter(Boolean).join(' ')}
      {...props}
    />
  )
}

type FormSelectProps = Omit<SelectHTMLAttributes<HTMLSelectElement>, 'className'> & {
  className?: string
  children: ReactNode
}

export function FormSelect({ className, id, children, ...props }: FormSelectProps) {
  const autoId = useId()
  const selectId = id ?? autoId
  return (
    <select
      id={selectId}
      className={['form-control', className].filter(Boolean).join(' ')}
      {...props}
    >
      {children}
    </select>
  )
}

export function FormCheckbox({
  label,
  checked,
  onChange,
}: {
  label: string
  checked: boolean
  onChange: (value: boolean) => void
}) {
  return (
    <label className="form-check">
      <input type="checkbox" checked={checked} onChange={(e) => onChange(e.target.checked)} />
      <span>{label}</span>
    </label>
  )
}

/** Text/number/password input with label spacing baked in. */
export function FormField({
  label,
  keyName,
  hint,
  full,
  ...inputProps
}: {
  label: string
  keyName?: string
  hint?: string
  full?: boolean
} & Omit<InputHTMLAttributes<HTMLInputElement>, 'className'>) {
  const id = useId()
  return (
    <FormItem label={label} keyName={keyName} hint={hint} full={full} htmlFor={id}>
      <FormInput id={id} {...inputProps} />
    </FormItem>
  )
}

/** Select with label spacing baked in. */
export function FormSelectField({
  label,
  keyName,
  hint,
  full,
  children,
  ...selectProps
}: {
  label: string
  keyName?: string
  hint?: string
  full?: boolean
  children: ReactNode
} & Omit<SelectHTMLAttributes<HTMLSelectElement>, 'className'>) {
  const id = useId()
  return (
    <FormItem label={label} keyName={keyName} hint={hint} full={full} htmlFor={id}>
      <FormSelect id={id} {...selectProps}>
        {children}
      </FormSelect>
    </FormItem>
  )
}

/** Link FormItem label to a control when the control manages its own id. */
export function useFormControlId() {
  return useId()
}
