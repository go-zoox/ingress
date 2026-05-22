import { parse, stringify } from 'yaml'

export function parseModuleDoc(yamlText: string): Record<string, unknown> {
  const text = yamlText.trim()
  if (!text) return {}
  const doc = parse(text)
  if (doc && typeof doc === 'object' && !Array.isArray(doc)) {
    return doc as Record<string, unknown>
  }
  return {}
}

export function stringifyModuleDoc(doc: Record<string, unknown>): string {
  if (Object.keys(doc).length === 0) return ''
  return stringify(doc, { lineWidth: 0 }).trimEnd()
}

export function num(v: unknown, fallback = 0): number {
  if (typeof v === 'number' && Number.isFinite(v)) return v
  if (typeof v === 'string' && v.trim() !== '') {
    const n = Number(v)
    if (Number.isFinite(n)) return n
  }
  return fallback
}

export function str(v: unknown, fallback = ''): string {
  return typeof v === 'string' ? v : v == null ? fallback : String(v)
}

export function bool(v: unknown, fallback = false): boolean {
  if (typeof v === 'boolean') return v
  if (v === 'true') return true
  if (v === 'false') return false
  return fallback
}

export function obj(v: unknown): Record<string, unknown> {
  if (v && typeof v === 'object' && !Array.isArray(v)) {
    return v as Record<string, unknown>
  }
  return {}
}

export function arr<T = unknown>(v: unknown): T[] {
  return Array.isArray(v) ? (v as T[]) : []
}

export function setNum(doc: Record<string, unknown>, key: string, value: number) {
  if (!value && value !== 0) delete doc[key]
  else doc[key] = value
}

export function setStr(doc: Record<string, unknown>, key: string, value: string) {
  if (!value.trim()) delete doc[key]
  else doc[key] = value
}

export function setBool(doc: Record<string, unknown>, key: string, value: boolean) {
  if (!value) delete doc[key]
  else doc[key] = true
}
