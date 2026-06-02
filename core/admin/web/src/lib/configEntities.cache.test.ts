import { describe, expect, it } from 'vitest'
import { backendToForm, emptyBackendForm, formToBackend } from './configEntities'

describe('buildCache response_header', () => {
  it('omits response_header when name and value are empty', () => {
    const form = { ...emptyBackendForm(), cache_enabled: true }
    const backend = formToBackend(form)
    const cache = backend.cache as Record<string, unknown>
    expect(cache.response_header).toBeUndefined()
  })

  it('persists custom response_header', () => {
    const form = {
      ...emptyBackendForm(),
      cache_enabled: true,
      cache_response_header_name: 'X-Custom-Cache',
      cache_response_header_value: 'HIT',
    }
    const backend = formToBackend(form)
    const cache = backend.cache as Record<string, unknown>
    expect(cache.response_header).toEqual({ name: 'X-Custom-Cache', value: 'HIT' })
  })

  it('loads response_header from backend.cache', () => {
    const form = backendToForm({
      type: 'handler',
      handler: { type: 'static_response', body: 'ok' },
      cache: {
        enabled: true,
        response_header: { name: 'X-Ingress-Cache', value: 'hit' },
      },
    })
    expect(form.cache_response_header_name).toBe('X-Ingress-Cache')
    expect(form.cache_response_header_value).toBe('hit')
  })
})
