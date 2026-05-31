import { useMemo } from 'react'
import CodeMirror from '@uiw/react-codemirror'
import { vscodeDark } from '@uiw/codemirror-theme-vscode'
import { EditorView } from '@codemirror/view'
import type { ScriptEngine } from '../lib/scriptParams'
import { codeMirrorLanguage } from '../lib/codeMirrorLanguages'

const editorChrome = EditorView.theme({
  '&': {
    fontSize: '13px',
    borderRadius: '8px',
    overflow: 'hidden',
    border: '1px solid var(--border)',
  },
  '&.cm-focused': {
    outline: 'none',
    borderColor: 'var(--accent)',
    boxShadow: '0 0 0 1px color-mix(in srgb, var(--accent) 35%, transparent)',
  },
  '.cm-scroller': {
    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
    lineHeight: '1.5',
  },
  '.cm-content': {
    minHeight: '120px',
    padding: '8px 0',
  },
})

type Props = {
  value: string
  onChange: (value: string) => void
  language?: ScriptEngine
  minHeight?: string
  readOnly?: boolean
  placeholder?: string
  /** Remount editor when external value is loaded (e.g. edit form hydrate). */
  sessionKey?: string | number
}

export function CodeEditor({
  value,
  onChange,
  language = 'shell',
  minHeight = '180px',
  readOnly = false,
  placeholder,
  sessionKey,
}: Props) {
  const extensions = useMemo(
    () => [
      codeMirrorLanguage(language),
      editorChrome,
      EditorView.lineWrapping,
      EditorView.editable.of(!readOnly),
      EditorView.contentAttributes.of({ 'aria-label': placeholder || '脚本内容' }),
    ],
    [language, readOnly, placeholder],
  )

  return (
    <div className="code-editor" data-language={language} style={{ minHeight }}>
      <CodeMirror
        key={`${language}:${sessionKey ?? 'default'}`}
        value={value}
        height={minHeight}
        theme={vscodeDark}
        extensions={extensions}
        onChange={onChange}
        basicSetup={{
          lineNumbers: true,
          foldGutter: true,
          dropCursor: false,
          allowMultipleSelections: false,
          indentOnInput: true,
          bracketMatching: true,
          closeBrackets: true,
          autocompletion: false,
          highlightActiveLine: true,
          highlightActiveLineGutter: true,
          highlightSelectionMatches: true,
          syntaxHighlighting: true,
        }}
      />
    </div>
  )
}
