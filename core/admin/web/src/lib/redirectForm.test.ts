import { describe, expect, it } from 'vitest'
import {
  redirectFromYAML,
  redirectStatusLabel,
  redirectToYAML,
} from './redirectForm'

describe('redirectForm', () => {
  it('maps scheme A fields to yaml', () => {
    expect(redirectToYAML({
      redirect_url: 'https://new.example.com',
      redirect_duration: 'permanent',
      redirect_preserve_request: true,
    })).toEqual({
      url: 'https://new.example.com',
      duration: 'permanent',
      preserve_request: true,
    })
  })

  it('omits defaults from yaml', () => {
    expect(redirectToYAML({
      redirect_url: 'https://new.example.com',
      redirect_duration: 'temporary',
      redirect_preserve_request: false,
    })).toEqual({ url: 'https://new.example.com' })
  })

  it('reads legacy permanent and with_origin_method_and_body', () => {
    const form = redirectFromYAML({
      url: 'https://old.example.com',
      permanent: true,
      with_origin_method_and_body: true,
    })
    expect(form.redirect_duration).toBe('permanent')
    expect(form.redirect_preserve_request).toBe(true)
  })

  it('prefers duration over legacy permanent', () => {
    const form = redirectFromYAML({
      url: 'https://old.example.com',
      duration: 'temporary',
      permanent: true,
    })
    expect(form.redirect_duration).toBe('temporary')
  })

  it('labels status codes', () => {
    expect(redirectStatusLabel('temporary', false)).toBe('302 Found')
    expect(redirectStatusLabel('permanent', false)).toBe('301 Moved Permanently')
    expect(redirectStatusLabel('temporary', true)).toBe('307 Temporary Redirect')
    expect(redirectStatusLabel('permanent', true)).toBe('308 Permanent Redirect')
  })
})
