import { obj, str } from './ingressModuleForms'

export const ERROR_PAGE_STATUS_CODES = [
  { code: '401', label: 'Unauthorized', hint: '鉴权失败（Basic / Bearer / JWT / OAuth 等）' },
  { code: '403', label: 'Forbidden', hint: 'WAF 默认拦截等' },
  { code: '404', label: 'Not Found', hint: '路由未匹配、上游未找到' },
  { code: '500', label: 'Internal Server Error', hint: 'Handler / 配置错误' },
  { code: '502', label: 'Bad Gateway', hint: '上游无效响应' },
  { code: '503', label: 'Service Unavailable', hint: 'DNS 失败、上游不可用' },
  { code: '504', label: 'Gateway Timeout', hint: '上游超时' },
] as const

export type ErrorPageCode = (typeof ERROR_PAGE_STATUS_CODES)[number]['code']

export type ErrorPageMode = 'default' | 'builtin' | 'file' | 'inline'

export type ErrorPageEntryForm = {
  mode: ErrorPageMode
  title: string
  subtitle: string
  file: string
  body: string
}

export function emptyErrorPageEntry(): ErrorPageEntryForm {
  return { mode: 'default', title: '', subtitle: '', file: '', body: '' }
}

function parseErrorPageSpec(raw: unknown): ErrorPageEntryForm {
  const spec = obj(raw)
  const pageType = str(spec.type, 'builtin').toLowerCase()
  if (pageType === 'file') {
    return {
      mode: 'file',
      title: '',
      subtitle: '',
      file: str(spec.file),
      body: '',
    }
  }
  if (pageType === 'inline') {
    return {
      mode: 'inline',
      title: '',
      subtitle: '',
      file: '',
      body: str(spec.body),
    }
  }
  const title = str(spec.title)
  const subtitle = str(spec.subtitle)
  return { mode: 'builtin', title, subtitle, file: '', body: '' }
}

export function errorPagesFromDoc(doc: Record<string, unknown>): Record<ErrorPageCode, ErrorPageEntryForm> {
  const pages = obj(obj(doc.error_pages).pages)
  const out = {} as Record<ErrorPageCode, ErrorPageEntryForm>
  for (const { code } of ERROR_PAGE_STATUS_CODES) {
    out[code] = pages[code] != null ? parseErrorPageSpec(pages[code]) : emptyErrorPageEntry()
  }
  return out
}

export function errorPagesToDocPages(
  entries: Record<ErrorPageCode, ErrorPageEntryForm>,
): Record<string, unknown> | null {
  const pages: Record<string, unknown> = {}
  for (const { code } of ERROR_PAGE_STATUS_CODES) {
    const entry = entries[code]
    if (!entry || entry.mode === 'default') continue
    if (entry.mode === 'file') {
      pages[code] = { type: 'file', file: entry.file.trim() }
      continue
    }
    if (entry.mode === 'inline') {
      pages[code] = { type: 'inline', body: entry.body }
      continue
    }
    if (entry.mode === 'builtin') {
      const spec: Record<string, unknown> = { type: 'builtin' }
      if (entry.title.trim()) spec.title = entry.title.trim()
      if (entry.subtitle.trim()) spec.subtitle = entry.subtitle.trim()
      pages[code] = spec
    }
  }
  return Object.keys(pages).length > 0 ? pages : null
}

export function patchErrorPagesOnDoc(
  doc: Record<string, unknown>,
  entries: Record<ErrorPageCode, ErrorPageEntryForm>,
) {
  const pages = errorPagesToDocPages(entries)
  if (!pages) {
    delete doc.error_pages
    return
  }
  doc.error_pages = { pages }
}

export function errorPageEntryConfigured(entry: ErrorPageEntryForm): boolean {
  return entry.mode !== 'default'
}
