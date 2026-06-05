import { describe, expect, it } from 'vitest'
import { formToRule, ruleToForm } from './configEntities'

describe('rule enabled', () => {
  it('defaults enabled to true when omitted in yaml', () => {
    const form = ruleToForm({
      host: 'app.example.com',
      backend: { service: { name: 'svc', port: 8080 } },
    })
    expect(form.enabled).toBe(true)
    expect(formToRule(form).enabled).toBeUndefined()
  })

  it('persists enabled: false', () => {
    const form = ruleToForm({
      host: 'app.example.com',
      enabled: false,
      backend: { service: { name: 'svc', port: 8080 } },
    })
    expect(form.enabled).toBe(false)
    expect(formToRule(form).enabled).toBe(false)
  })

  it('re-enabling removes enabled key from yaml', () => {
    const original = {
      host: 'app.example.com',
      enabled: false,
      backend: { service: { name: 'svc', port: 8080 } },
    }
    const form = ruleToForm(original)
    const out = formToRule({ ...form, enabled: true }, original)
    expect(out.enabled).toBeUndefined()
  })
})
