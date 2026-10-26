import { describe, expect, it } from 'vitest'
import { emptyRuleForm, formToRule, ruleSaveDisabled, ruleSummary, ruleToForm } from './configEntities'

const doc = {
  host: 'anchor.yss.example.com',
  backend: {
    service: { name: 'anchor-frontend', port: 8080, protocol: 'http' },
  },
  connect_proxy: {
    enabled: true,
    allowed_targets: ['^.+:(80|443)$', '^db\\.internal:5432$'],
    upstream: { name: 'anchor-backend', port: 3128, protocol: 'http' },
    connect_timeout: 10,
    response_timeout: 12,
    idle_timeout: 600,
    max_connections: 500,
  },
}

describe('CONNECT proxy rule form', () => {
  it('round-trips connect_proxy fields and leaves the normal backend intact', () => {
    const form = ruleToForm(doc)
    expect(form.connect_proxy_enabled).toBe(true)
    expect(form.connect_proxy_allowed_targets).toContain('^db\\.internal:5432$')
    expect(form.connect_proxy_upstream_name).toBe('anchor-backend')
    expect(form.connect_proxy_upstream_port).toBe(3128)
    expect(form.connect_proxy_max_connections).toBe(500)

    const row = formToRule(form, doc)
    const backend = row.backend as { service?: { name?: string } }
    expect(backend.service?.name).toBe('anchor-frontend')
    const cp = row.connect_proxy as {
      enabled: boolean
      allowed_targets: string[]
      upstream: { name: string; port: number; protocol: string }
      connect_timeout: number
      response_timeout: number
      idle_timeout: number
      max_connections: number
    }
    expect(cp).toMatchObject({
      enabled: true,
      upstream: { name: 'anchor-backend', port: 3128, protocol: 'http' },
      connect_timeout: 10,
      response_timeout: 12,
      idle_timeout: 600,
      max_connections: 500,
    })
    expect(cp.allowed_targets).toEqual(['^.+:(80|443)$', '^db\\.internal:5432$'])
  })

  it('does not emit connect_proxy when disabled', () => {
    const form = ruleToForm(doc)
    form.connect_proxy_enabled = false
    const row = formToRule(form, doc) as Record<string, unknown>
    expect(row.connect_proxy).toBeUndefined()
  })

  it('shows CONNECT upstream in the list summary', () => {
    expect(ruleSummary(doc)).toContain('PROXY → anchor-backend:3128')
  })

  it('empty form keeps CONNECT disabled', () => {
    expect(emptyRuleForm().connect_proxy_enabled).toBe(false)
  })
})


describe('CONNECT proxy form field coverage', () => {
  it('serializes every editable field', () => {
    const form = ruleToForm(doc)
    form.connect_proxy_allowed_targets = '^api.example.com:443$\n^cdn.example.com:80$'
    form.connect_proxy_upstream_name = 'new-proxy'
    form.connect_proxy_upstream_port = 4180
    form.connect_proxy_upstream_protocol = 'https'
    form.connect_proxy_connect_timeout = 3
    form.connect_proxy_response_timeout = 4
    form.connect_proxy_idle_timeout = 5
    form.connect_proxy_max_connections = 6

    const row = formToRule(form, doc)
    const cp = row.connect_proxy as Record<string, unknown>
    expect(cp).toMatchObject({
      enabled: true,
      allowed_targets: ['^api.example.com:443$', '^cdn.example.com:80$'],
      upstream: { name: 'new-proxy', port: 4180, protocol: 'https' },
      connect_timeout: 3,
      response_timeout: 4,
      idle_timeout: 5,
      max_connections: 6,
    })
  })

  it('omits upstream when the field is blank so backend fallback is preserved', () => {
    const form = ruleToForm(doc)
    form.connect_proxy_upstream_name = ''
    const row = formToRule(form, doc)
    const cp = row.connect_proxy as Record<string, unknown>
    expect(cp.upstream).toBeUndefined()
    expect(cp.enabled).toBe(true)
  })

  it('blocks save for invalid UI values', () => {
    const form = ruleToForm(doc)
    form.connect_proxy_upstream_port = 0
    expect(ruleSaveDisabled(form)).toBe(true)

    const negative = ruleToForm(doc)
    negative.connect_proxy_connect_timeout = -1
    expect(ruleSaveDisabled(negative)).toBe(true)

    const handlerWithoutUpstream = ruleToForm(doc)
    handlerWithoutUpstream.backend_type = 'handler'
    handlerWithoutUpstream.connect_proxy_upstream_name = ''
    expect(ruleSaveDisabled(handlerWithoutUpstream)).toBe(true)
  })

  it('loads absent fields with backend defaults and unlimited max connections', () => {
    const form = ruleToForm({
      host: 'legacy.example.com',
      backend: { service: { name: 'legacy', port: 8080 } },
      connect_proxy: { enabled: true },
    })
    expect(form.connect_proxy_connect_timeout).toBe(10)
    expect(form.connect_proxy_response_timeout).toBe(10)
    expect(form.connect_proxy_idle_timeout).toBe(600)
    expect(form.connect_proxy_max_connections).toBe(0)
    expect(form.connect_proxy_upstream_name).toBe('')
  })
})
