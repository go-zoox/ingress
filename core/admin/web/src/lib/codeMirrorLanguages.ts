import type { Extension } from '@codemirror/state'
import { StreamLanguage } from '@codemirror/language'
import { javascript } from '@codemirror/lang-javascript'
import { go } from '@codemirror/lang-go'
import { shell } from '@codemirror/legacy-modes/mode/shell'
import type { ScriptEngine } from '../lib/scriptParams'

const shellLanguage = StreamLanguage.define(shell)

/** CodeMirror language extension for each scheduled-job script engine. */
export function codeMirrorLanguage(engine: ScriptEngine): Extension {
  switch (engine) {
    case 'javascript':
      return javascript({ jsx: false, typescript: false })
    case 'go':
      return go()
    case 'shell':
    default:
      return shellLanguage
  }
}

export const CODE_MIRROR_LANGUAGE_LABEL: Record<ScriptEngine, string> = {
  shell: 'Shell',
  javascript: 'JavaScript',
  go: 'Go',
}
