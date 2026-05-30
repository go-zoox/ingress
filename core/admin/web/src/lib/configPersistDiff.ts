import { buildDiff } from './config'
import { stringifyModuleDoc } from './ingressModuleForms'

export type PersistDiffOptions = {
  savedYAML: string
  nextYAML: string
  savedDoc: Record<string, unknown>
  doc: Record<string, unknown>
  moduleLabel: string
}

/** Diff for save/publish preview: prefer full ingress.yaml, fall back to module YAML. */
export function buildPersistDiff(opts: PersistDiffOptions): string {
  const full = buildDiff(opts.savedYAML, opts.nextYAML)
  if (full !== '(无变更)') {
    return `<div class="diff-section-label">完整 ingress.yaml（已发布 → 即将写入）</div>\n${full}`
  }

  const modSaved = stringifyModuleDoc(opts.savedDoc)
  const modNext = stringifyModuleDoc(opts.doc)
  const moduleDiff = buildDiff(modSaved, modNext)
  if (moduleDiff !== '(无变更)') {
    return `<div class="diff-section-label">${opts.moduleLabel} 模块（已发布 → 草稿）</div>\n${moduleDiff}`
  }

  return '(无变更)'
}

/** Module draft dirty check — aligned with YAML merge / diff. */
export function isModuleDocDirty(
  doc: Record<string, unknown>,
  savedDoc: Record<string, unknown>,
): boolean {
  return stringifyModuleDoc(doc) !== stringifyModuleDoc(savedDoc)
}
