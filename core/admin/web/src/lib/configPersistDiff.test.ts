import { describe, expect, it } from 'vitest'
import { buildPersistDiff, isModuleDocDirty } from './configPersistDiff'

describe('isModuleDocDirty', () => {
  it('is false when module YAML is unchanged', () => {
    const saved = { services: [{ name: 'a', port: 8080 }] }
    const doc = { services: [{ name: 'a', port: 8080 }] }
    expect(isModuleDocDirty(doc, saved)).toBe(false)
  })

  it('detects real module edits', () => {
    const saved = { services: [{ name: 'a', port: 8080 }] }
    const doc = { services: [{ name: 'a', port: 9090 }] }
    expect(isModuleDocDirty(doc, saved)).toBe(true)
  })
})

describe('buildPersistDiff', () => {
  it('shows full config diff when merged yaml changes', () => {
    const html = buildPersistDiff({
      savedYAML: 'services:\n  - name: a\n    port: 8080\n',
      nextYAML: 'services:\n  - name: a\n    port: 9090\n',
      savedDoc: { services: [{ name: 'a', port: 8080 }] },
      doc: { services: [{ name: 'a', port: 9090 }] },
      moduleLabel: '服务',
    })
    expect(html).toContain('完整 ingress.yaml')
    expect(html).toContain('9090')
  })

  it('falls back to module diff when full yaml matches but module draft differs', () => {
    const yaml = 'rules:\n  - host: a.example.com\n'
    const html = buildPersistDiff({
      savedYAML: yaml,
      nextYAML: yaml,
      savedDoc: { rules: [{ host: 'a.example.com', backend: { type: 'service', service: { name: 'x', port: 8080 } } }] },
      doc: { rules: [{ host: 'a.example.com', backend: { type: 'service', service: { name: 'x', port: 9090 } } }] },
      moduleLabel: '路由规则',
    })
    expect(html).toContain('路由规则 模块')
    expect(html).toContain('9090')
  })
})
