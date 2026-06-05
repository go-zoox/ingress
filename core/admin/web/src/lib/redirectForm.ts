export type RedirectDuration = 'temporary' | 'permanent'

export type RedirectFormSlice = {
  redirect_url: string
  redirect_duration: RedirectDuration
  redirect_preserve_request: boolean
}

function str(v: unknown): string {
  return typeof v === 'string' ? v : v == null ? '' : String(v)
}

function bool(v: unknown): boolean {
  return v === true
}

function obj(v: unknown): Record<string, unknown> {
  return v != null && typeof v === 'object' && !Array.isArray(v) ? (v as Record<string, unknown>) : {}
}

export function emptyRedirectFormSlice(): RedirectFormSlice {
  return {
    redirect_url: '',
    redirect_duration: 'temporary',
    redirect_preserve_request: false,
  }
}

export function redirectStatusLabel(duration: RedirectDuration, preserveRequest: boolean): string {
  if (preserveRequest) {
    return duration === 'permanent' ? '308 Permanent Redirect' : '307 Temporary Redirect'
  }
  return duration === 'permanent' ? '301 Moved Permanently' : '302 Found'
}

export function redirectFromYAML(redirectRaw: Record<string, unknown>): RedirectFormSlice {
  const redirect = obj(redirectRaw)
  const durationRaw = str(redirect.duration).toLowerCase()
  let redirect_duration: RedirectDuration = 'temporary'
  if (durationRaw === 'permanent') redirect_duration = 'permanent'
  else if (durationRaw === 'temporary') redirect_duration = 'temporary'
  else if (bool(redirect.permanent)) redirect_duration = 'permanent'

  let redirect_preserve_request = bool(redirect.preserve_request)
  if (redirect.preserve_request === undefined) {
    redirect_preserve_request = bool(redirect.with_origin_method_and_body)
  }

  return {
    redirect_url: str(redirect.url),
    redirect_duration,
    redirect_preserve_request,
  }
}

export function redirectBehaviorToYAML(
  duration: RedirectDuration,
  preserveRequest: boolean,
): Record<string, unknown> {
  const redirect: Record<string, unknown> = {}
  if (duration === 'permanent') redirect.duration = 'permanent'
  if (preserveRequest) redirect.preserve_request = true
  return redirect
}

export function redirectToYAML(
  form: Pick<RedirectFormSlice, 'redirect_url' | 'redirect_duration' | 'redirect_preserve_request'>,
): Record<string, unknown> {
  const redirect: Record<string, unknown> = { url: form.redirect_url.trim() }
  if (form.redirect_duration === 'permanent') redirect.duration = 'permanent'
  if (form.redirect_preserve_request) redirect.preserve_request = true
  return redirect
}
