import {
  useId,
  type InputHTMLAttributes,
  type ReactNode,
  type SelectHTMLAttributes,
  type TextareaHTMLAttributes,
} from 'react'
import { CodeEditor } from './CodeEditor'

export function FormSwitch({
  checked,
  onChange,
  label,
  id: idProp,
}: {
  checked: boolean
  onChange: (checked: boolean) => void
  label?: string
  id?: string
}) {
  const autoId = useId()
  const id = idProp ?? autoId
  return (
    <label className="form-switch" htmlFor={id} title={label}>
      <input
        id={id}
        type="checkbox"
        role="switch"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
      />
      <span className="form-switch-track" aria-hidden />
    </label>
  )
}

export function CollapsibleFormSection({
  title,
  open,
  onOpenChange,
  children,
}: {
  title?: string
  open: boolean
  onOpenChange: (open: boolean) => void
  children: ReactNode
}) {
  return (
    <section className={`form-section${open ? ' form-section--open' : ' form-section--collapsed'}`}>
      <div className="form-section-head">
        {title ? <h4 className="form-section-title">{title}</h4> : <span />}
        <FormSwitch checked={open} onChange={onOpenChange} label={open ? '展开' : '折叠'} />
      </div>
      {open ? <div className="form-section-body">{children}</div> : null}
    </section>
  )
}

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
  hint,
  full,
  htmlFor,
  children,
}: {
  label: string
  hint?: string
  full?: boolean
  htmlFor?: string
  children: ReactNode
}) {
  return (
    <div className={`form-item${full ? ' form-item--full' : ''}`}>
      <div className="form-label-row">
        <label className="form-label" htmlFor={htmlFor}>
          <span className="form-label-text">{label}</span>
        </label>
        {hint ? <p className="form-hint form-hint--field">{hint}</p> : null}
      </div>
      <div className="form-control-wrap">{children}</div>
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
  keyName: _keyName,
  hint,
  full,
  ...inputProps
}: {
  label: string
  /** @deprecated Config key is no longer shown in the UI */
  keyName?: string
  hint?: string
  full?: boolean
} & Omit<InputHTMLAttributes<HTMLInputElement>, 'className'>) {
  const id = useId()
  return (
    <FormItem label={label} hint={hint} full={full} htmlFor={id}>
      <FormInput id={id} {...inputProps} />
    </FormItem>
  )
}

/** Multiline text with label spacing baked in (compact; not full-page YAML editors). */
export function FormTextareaField({
  label,
  keyName: _keyName,
  hint,
  full,
  mono,
  rows = 4,
  className,
  ...textareaProps
}: {
  label: string
  /** @deprecated Config key is no longer shown in the UI */
  keyName?: string
  hint?: string
  full?: boolean
  mono?: boolean
  rows?: number
  className?: string
} & Omit<TextareaHTMLAttributes<HTMLTextAreaElement>, 'className' | 'rows'>) {
  const id = useId()
  return (
    <FormItem label={label} hint={hint} full={full} htmlFor={id}>
      <textarea
        id={id}
        rows={rows}
        className={[
          'form-control',
          'form-textarea',
          mono ? 'form-textarea--mono' : '',
          className,
        ]
          .filter(Boolean)
          .join(' ')}
        {...textareaProps}
      />
    </FormItem>
  )
}

/** Select with label spacing baked in. */
export function FormSelectField({
  label,
  keyName: _keyName,
  hint,
  full,
  children,
  ...selectProps
}: {
  label: string
  /** @deprecated Config key is no longer shown in the UI */
  keyName?: string
  hint?: string
  full?: boolean
  children: ReactNode
} & Omit<SelectHTMLAttributes<HTMLSelectElement>, 'className'>) {
  const id = useId()
  return (
    <FormItem label={label} hint={hint} full={full} htmlFor={id}>
      <FormSelect id={id} {...selectProps}>
        {children}
      </FormSelect>
    </FormItem>
  )
}

/** Checkbox multi-select in a dropdown panel (details/summary). */
export function FormMultiSelectField({
  label,
  keyName: _keyName,
  hint,
  options,
  value,
  onChange,
  placeholder = '点击选择…',
}: {
  label: string
  /** @deprecated Config key is no longer shown in the UI */
  keyName?: string
  hint?: string
  options: readonly string[]
  value: string[]
  onChange: (next: string[]) => void
  placeholder?: string
}) {
  const selected = value.map((v) => v.toUpperCase())
  const summary =
    selected.length > 0 ? selected.join(', ') : placeholder

  const toggle = (opt: string, checked: boolean) => {
    const u = opt.toUpperCase()
    if (checked) {
      if (selected.includes(u)) return
      onChange([...selected, u])
      return
    }
    onChange(selected.filter((x) => x !== u))
  }

  return (
    <FormItem label={label} hint={hint}>
      <details className="form-multi-select">
        <summary className="form-control form-multi-select-summary">{summary}</summary>
        <div className="form-multi-select-panel">
          {options.map((opt) => {
            const u = opt.toUpperCase()
            return (
              <label key={opt} className="form-check form-check--inline">
                <input
                  type="checkbox"
                  checked={selected.includes(u)}
                  onChange={(e) => toggle(opt, e.target.checked)}
                  onClick={(e) => e.stopPropagation()}
                />
                <span>{opt}</span>
              </label>
            )
          })}
        </div>
      </details>
    </FormItem>
  )
}

/** Shell/script editor with label spacing baked in. */
export function FormCodeEditorField({
  label,
  hint,
  full,
  value,
  onChange,
  language,
  minHeight,
  readOnly,
}: {
  label: string
  hint?: string
  full?: boolean
  value: string
  onChange: (value: string) => void
  language?: import('../lib/scriptParams').ScriptEngine
  minHeight?: string
  readOnly?: boolean
}) {
  return (
    <FormItem label={label} hint={hint} full={full}>
      <CodeEditor value={value} onChange={onChange} language={language} minHeight={minHeight} readOnly={readOnly} />
    </FormItem>
  )
}

/** Link FormItem label to a control when the control manages its own id. */
export function useFormControlId() {
  return useId()
}
