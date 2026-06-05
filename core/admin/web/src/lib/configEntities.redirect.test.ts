import { describe, expect, it } from 'vitest'
import { backendToForm, formToBackend } from './configEntities'
import { redirectFromYAML, redirectToYAML } from './redirectForm'

describe('configEntities redirect roundtrip', () => {
  it('roundtrips scheme A redirect fields through backend form', () => {
    const original = {
      type: 'redirect',
      redirect: {
        url: 'https://new.example.com',
        duration: 'permanent',
        preserve_request: true,
      },
    }

    const form = backendToForm(original)
    expect(form.redirect_url).toBe('https://new.example.com')
    expect(form.redirect_duration).toBe('permanent')
    expect(form.redirect_preserve_request).toBe(true)

    const out = formToBackend(form, original)
    expect(out.redirect).toEqual({
      url: 'https://new.example.com',
      duration: 'permanent',
      preserve_request: true,
    })
    expect(out.type).toBe('redirect')
  })

  it('drops legacy permanent when admin writes duration', () => {
    const original = {
      redirect: {
        url: 'https://old.example.com',
        permanent: true,
      },
    }
    const form = backendToForm({ type: 'redirect', ...original })
    expect(form.redirect_duration).toBe('permanent')

    const out = formToBackend({
      ...form,
      redirect_duration: 'temporary',
      redirect_preserve_request: false,
    }, { type: 'redirect', ...original })

    expect(out.redirect).toEqual({
      url: 'https://old.example.com',
    })
    expect((out.redirect as Record<string, unknown>).permanent).toBeUndefined()
  })
})

describe('redirectForm extras', () => {
  it('redirectBehaviorToYAML omits defaults', async () => {
    const { redirectBehaviorToYAML } = await import('./redirectForm')
    expect(redirectBehaviorToYAML('temporary', false)).toEqual({})
    expect(redirectBehaviorToYAML('permanent', true)).toEqual({
      duration: 'permanent',
      preserve_request: true,
    })
  })

  it('redirectFromYAML reads preserve_request directly', () => {
    const form = redirectFromYAML({
      url: 'https://x.example/',
      preserve_request: true,
      duration: 'temporary',
    })
    expect(form.redirect_preserve_request).toBe(true)
    expect(redirectToYAML(form)).toEqual({
      url: 'https://x.example/',
      preserve_request: true,
    })
  })
})
