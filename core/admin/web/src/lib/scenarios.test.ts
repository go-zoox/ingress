import { describe, expect, it } from 'vitest'
import {
  DEFAULT_SCENARIO_ID,
  effectiveScenarioActive,
  isDefaultScenario,
  patchModuleDocScenarios,
  scenarioIdFromLabel,
  scenariosFromModuleDoc,
  validateScenariosFormState,
} from './scenarios'

describe('isDefaultScenario', () => {
  it('treats empty and default as baseline', () => {
    expect(isDefaultScenario('')).toBe(true)
    expect(isDefaultScenario('default')).toBe(true)
    expect(isDefaultScenario('live')).toBe(false)
  })
})

describe('effectiveScenarioActive', () => {
  it('normalizes empty active to default', () => {
    expect(effectiveScenarioActive('')).toBe(DEFAULT_SCENARIO_ID)
    expect(effectiveScenarioActive('live')).toBe('live')
  })
})

describe('scenarioIdFromLabel', () => {
  it('slugifies ascii labels', () => {
    expect(scenarioIdFromLabel('Live Stream', [])).toBe('live-stream')
  })

  it('falls back when slug is reserved or empty', () => {
    expect(scenarioIdFromLabel('直播', [])).toMatch(/^scenario-\d+$/)
    expect(scenarioIdFromLabel('default', [])).toMatch(/^scenario-\d+$/)
  })
})

describe('scenariosFromModuleDoc / patchModuleDocScenarios', () => {
  it('round-trips active default and items', () => {
    const doc = {
      scenarios: {
        active: 'default',
        items: [{ id: 'live', label: '直播', overlay: { cache: { host: 'redis' } } }],
      },
    }
    const state = scenariosFromModuleDoc(doc)
    expect(state.active).toBe(DEFAULT_SCENARIO_ID)
    expect(state.items).toHaveLength(1)
    expect(state.items[0].sections.cache).toBe(true)

    const next = patchModuleDocScenarios({}, state)
    expect(next.scenarios).toMatchObject({
      active: DEFAULT_SCENARIO_ID,
      items: [{ id: 'live', label: '直播' }],
    })
    expect((next.scenarios as { items: Array<{ overlay: { cache: { host: string } } }> }).items[0].overlay.cache.host).toBe('redis')
  })
})

describe('validateScenariosFormState', () => {
  it('allows active default without item entry', () => {
    expect(
      validateScenariosFormState({
        active: DEFAULT_SCENARIO_ID,
        items: [{ id: 'live', label: '直播', description: '', sections: { cache: false, rate_limit: false, waf: false, maintenance: false, security: false, rules: false }, cache: { ttl: 300, host: '', port: 6379, prefix: '', username: '', password: '', db: 0 }, rate_limit: { rate_limit_enabled: undefined, rate_limit_requests: 0, rate_limit_period: 60, rate_limit_key: 'ip', rate_limit_header: '', rate_limit_trust_proxy: false, rate_limit_xff_index: 0 }, rule_patches: [], waf: {}, maintenance: {}, security: {} }],
      }),
    ).toBeNull()
  })

  it('rejects reserved default item id', () => {
    const err = validateScenariosFormState({
      active: DEFAULT_SCENARIO_ID,
      items: [{
        id: DEFAULT_SCENARIO_ID,
        label: 'x',
        description: '',
        sections: { cache: false, rate_limit: false, waf: false, maintenance: false, security: false, rules: false },
        cache: { ttl: 300, host: '', port: 6379, prefix: '', username: '', password: '', db: 0 },
        rate_limit: { rate_limit_enabled: undefined, rate_limit_requests: 0, rate_limit_period: 60, rate_limit_key: 'ip', rate_limit_header: '', rate_limit_trust_proxy: false, rate_limit_xff_index: 0 },
        rule_patches: [],
        waf: {},
        maintenance: {},
        security: {},
      }],
    })
    expect(err).toContain('default')
  })
})
