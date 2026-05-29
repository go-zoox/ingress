import type { SecurityFormSlice, SecurityProfile } from './configEntities'

/** Mirrors core/security/profiles.go profileTable — keep in sync when presets change. */
export const SECURITY_PROFILE_DEFAULTS = {
  strict: {
    label: '通用 Web 站点',
    summary: '完整 CSP · 禁止 iframe · 无跨域',
    hsts: 'auto' as const,
    frameEffective: 'deny' as const,
    frameLabel: '禁止嵌入 (DENY)',
    contentTypeOptions: true,
    referrerPolicy: 'strict-origin-when-cross-origin',
    csp: "default-src 'self'; frame-ancestors 'none'",
    corsEnabled: false,
  },
  api: {
    label: 'API 接口（含跨域）',
    summary: '最严 CSP · 禁止 iframe · 开启 CORS',
    hsts: 'auto' as const,
    frameEffective: 'deny' as const,
    frameLabel: '禁止嵌入 (DENY)',
    contentTypeOptions: true,
    referrerPolicy: 'strict-origin-when-cross-origin',
    csp: "default-src 'none'; frame-ancestors 'none'",
    corsEnabled: true,
  },
  embeddable: {
    label: '可被 iframe 嵌入',
    summary: '允许同源嵌入 · 无跨域',
    hsts: 'auto' as const,
    frameEffective: 'sameorigin' as const,
    frameLabel: '同源可嵌入 (SAMEORIGIN)',
    contentTypeOptions: true,
    referrerPolicy: 'strict-origin-when-cross-origin',
    csp: "frame-ancestors 'self'",
    corsEnabled: false,
  },
} as const

export type ActiveSecurityProfile = keyof typeof SECURITY_PROFILE_DEFAULTS

export function isActiveSecurityProfile(profile: string): profile is ActiveSecurityProfile {
  return profile in SECURITY_PROFILE_DEFAULTS
}

export function profileDefaults(profile: string) {
  if (!isActiveSecurityProfile(profile)) return null
  return SECURITY_PROFILE_DEFAULTS[profile]
}

/** Reset overrides when user picks a different preset so UI matches runtime behavior. */
export function applySecurityProfileSwitch(
  form: SecurityFormSlice,
  nextProfile: SecurityProfile | '' | 'off',
): SecurityFormSlice {
  const profile = (nextProfile || 'off') as SecurityProfile | 'off'
  if (profile === 'off' || profile === '') {
    return { ...form, security_profile: 'off' }
  }
  const def = SECURITY_PROFILE_DEFAULTS[profile]
  return {
    ...form,
    security_profile: profile,
    security_hsts: def.hsts,
    security_frame: 'inherit',
    security_referrer_policy: '',
    security_csp: '',
    security_content_type_options: def.contentTypeOptions,
    security_cors_enabled: def.corsEnabled,
    security_cors_origins: '',
    security_cors_methods: '',
    security_cors_headers: '',
    security_cors_credentials: false,
    security_cors_max_age: 0,
  }
}

export function effectiveFrame(form: SecurityFormSlice): 'deny' | 'sameorigin' | 'off' {
  const mode = form.security_frame || 'inherit'
  if (mode === 'off') return 'off'
  if (mode === 'deny' || mode === 'sameorigin') return mode
  const def = profileDefaults(form.security_profile || 'off')
  return def?.frameEffective ?? 'off'
}

export function effectiveReferrerPolicy(form: SecurityFormSlice): string {
  const custom = form.security_referrer_policy.trim()
  if (custom && custom.toLowerCase() !== 'off') return custom
  const def = profileDefaults(form.security_profile || 'off')
  return def?.referrerPolicy ?? ''
}

export function effectiveCSP(form: SecurityFormSlice): string {
  const custom = form.security_csp.trim()
  if (custom && custom.toLowerCase() !== 'off') return custom
  const def = profileDefaults(form.security_profile || 'off')
  return def?.csp ?? ''
}

export function effectiveCORSEnabled(form: SecurityFormSlice): boolean {
  if (form.security_cors_enabled === false) return false
  if (form.security_cors_enabled === true) return true
  if (form.security_cors_origins.trim()) return true
  const def = profileDefaults(form.security_profile || 'off')
  return def?.corsEnabled ?? false
}
