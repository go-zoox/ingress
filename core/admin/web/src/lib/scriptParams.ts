import type { JobParams } from '../api/client'

export type ScriptEngine = 'shell' | 'javascript' | 'go'

/** Scheduled-job script engine (not the same as shell interpreter type). */
export const SCRIPT_ENGINES: { value: ScriptEngine; label: string; hint: string }[] = [
  {
    value: 'shell',
    label: 'Shell',
    hint: '在主机上通过 Shell 解释器执行脚本',
  },
  {
    value: 'javascript',
    label: 'JavaScript',
    hint: '内置 goja；见下方可用 API',
  },
  {
    value: 'go',
    label: 'Go',
    hint: '内置 yaegi 标准库；见下方可用 API',
  },
]

/** Shell interpreter presets — only when engine is `shell`. */
export const SCRIPT_SHELL_PRESETS = ['sh', 'bash', 'zsh'] as const

export type ScriptShellPreset = (typeof SCRIPT_SHELL_PRESETS)[number]

export const DEFAULT_JOB_SCRIPT: Record<ScriptEngine, string> = {
  shell: '#!/bin/sh\necho "job started"',
  javascript: `console.log("job started", new Date().toISOString())
// await fetch("https://example.com/health")
`,
  go: `import "fmt"

fmt.Println("job started")
`,
}

/** Built-in APIs available in embedded script engines (shown in UI). */
export const SCRIPT_ENGINE_BUILTIN_LIBS: Record<'javascript' | 'go', string> = {
  javascript: 'console（log / error / warn）· fetch(url, { method, body }) → { status, ok, text(), json() }',
  go: 'Go 标准库（yaegi）：fmt · strings · strconv · time · encoding/json · os · net/http · bytes · errors …；输出请用 fmt.Println',
}

export type ScriptParamsInput = {
  engine: ScriptEngine
  script: string
  /** Shell interpreter (sh/bash/…); only valid when engine is shell. */
  shell?: string
  workdir?: string
}

export function defaultScriptEngine(): ScriptEngine {
  return 'shell'
}

export function defaultScriptShell(): ScriptShellPreset {
  return 'sh'
}

export function normalizeScriptEngine(engine?: string): ScriptEngine {
  const v = (engine || '').trim().toLowerCase()
  if (v === 'javascript' || v === 'js') return 'javascript'
  if (v === 'go' || v === 'golang') return 'go'
  return 'shell'
}

export function scriptEngineLabel(engine: ScriptEngine): string {
  return SCRIPT_ENGINES.find((e) => e.value === engine)?.label ?? engine
}

/** Map stored params to editor fields (legacy command/args supported). */
export function scriptParamsFromJob(params: JobParams): ScriptParamsInput {
  const engine = normalizeScriptEngine(params.engine)
  let script = params.script?.trim() || ''
  if (!script) {
    script = [params.command, ...(params.args ?? [])].filter(Boolean).join(' ').trim()
  }
  const out: ScriptParamsInput = {
    engine,
    script,
    workdir: params.workdir || '',
  }
  if (engine === 'shell') {
    out.shell = params.shell?.trim() || defaultScriptShell()
  }
  return out
}

export function scriptParamsToJob(input: ScriptParamsInput): JobParams {
  const engine = normalizeScriptEngine(input.engine)
  const body: JobParams = {
    engine,
    script: input.script,
  }
  if (engine === 'shell') {
    body.shell = (input.shell || defaultScriptShell()).trim() || defaultScriptShell()
  }
  if (input.workdir?.trim()) {
    body.workdir = input.workdir.trim()
  }
  return body
}

export function isPresetShell(shell: string): shell is ScriptShellPreset {
  return (SCRIPT_SHELL_PRESETS as readonly string[]).includes(shell)
}

export function scriptEngineHint(engine: ScriptEngine): string {
  return SCRIPT_ENGINES.find((e) => e.value === engine)?.hint ?? ''
}

export function scriptEngineKindLabel(engine: ScriptEngine): string {
  if (engine === 'shell') return '脚本执行 · Shell'
  if (engine === 'javascript') return '脚本执行 · JavaScript'
  return '脚本执行 · Go'
}

/** Normalize script text for default-sample comparison. */
export function normalizeScriptText(script: string): string {
  return script.replace(/\r\n/g, '\n').trimEnd()
}

/** True when script is empty or still the built-in sample for the given engine. */
export function isDefaultScriptForEngine(script: string, engine: ScriptEngine): boolean {
  const trimmed = script.trim()
  if (!trimmed) return true
  return normalizeScriptText(script) === normalizeScriptText(DEFAULT_JOB_SCRIPT[engine])
}

/**
 * Pick script body when switching engines: keep user edits, otherwise use the new engine sample.
 */
export function scriptEditorHint(engine: ScriptEngine): string {
  if (engine === 'javascript') return SCRIPT_ENGINE_BUILTIN_LIBS.javascript
  if (engine === 'go') return SCRIPT_ENGINE_BUILTIN_LIBS.go
  return '由 Shell 解释器以 -c 方式执行'
}

/**
 * Pick script body when switching engines: keep user edits, otherwise use the new engine sample.
 */
export function scriptWhenEngineChanges(
  currentScript: string,
  fromEngine: ScriptEngine,
  toEngine: ScriptEngine,
): string {
  if (fromEngine === toEngine) return currentScript
  if (isDefaultScriptForEngine(currentScript, fromEngine)) {
    return DEFAULT_JOB_SCRIPT[toEngine]
  }
  return currentScript
}
