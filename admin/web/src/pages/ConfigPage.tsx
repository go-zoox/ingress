import { useEffect, useRef, useState } from 'react'
import { PageHeader } from '../components/PageHeader'
import { DiffModal } from '../components/DiffModal'
import { PublishModal } from '../components/PublishModal'
import { ToastContainer, useToast } from '../components/Toast'
import { api } from '../api/client'

function buildDiff(baseline: string, current: string): string {
  const a = baseline.split('\n')
  const b = current.split('\n')
  const max = Math.max(a.length, b.length)
  const lines: string[] = []
  for (let i = 0; i < max; i++) {
    const x = a[i]
    const y = b[i]
    if (x === y) {
      if (x !== undefined) lines.push('  ' + escapeHtml(x))
    } else {
      if (x !== undefined) lines.push('<span class="del">- ' + escapeHtml(x) + '</span>')
      if (y !== undefined) lines.push('<span class="add">+ ' + escapeHtml(y) + '</span>')
    }
  }
  return lines.join('\n') || '(无变更)'
}

function escapeHtml(s: string) {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

export function ConfigPage() {
  const [content, setContent] = useState('')
  const [saved, setSaved] = useState('')
  const [path, setPath] = useState('')
  const [validateOut, setValidateOut] = useState('')
  const [err, setErr] = useState('')
  const [publishOpen, setPublishOpen] = useState(false)
  const [diffOpen, setDiffOpen] = useState(false)
  const [diffHtml, setDiffHtml] = useState('')
  const { toast, show, clear } = useToast()
  const loaded = useRef(false)

  useEffect(() => {
    api
      .getConfig()
      .then((r) => {
        setContent(r.content)
        setSaved(r.content)
        setPath(r.path)
        loaded.current = true
      })
      .catch((e: Error) => setErr(e.message))
  }, [])

  const validate = () => {
    setErr('')
    setValidateOut('')
    api
      .validateConfig(content)
      .then(() => setValidateOut('<p class="validate-ok">✓ 校验通过</p>'))
      .catch((e: Error) => {
        setValidateOut('<p class="validate-err">' + escapeHtml(e.message) + '</p>')
      })
  }

  const save = () => {
    setErr('')
    api
      .validateConfig(content)
      .then(() => api.putConfig(content))
      .then(() => {
        setSaved(content)
        show('已保存到 ' + path)
      })
      .catch((e: Error) => {
        setErr(e.message)
        show(e.message, 'error')
      })
  }

  const showDiff = () => {
    setDiffHtml(buildDiff(saved, content))
    setDiffOpen(true)
  }

  return (
    <div className="page">
      <PageHeader title="配置" desc="编辑 YAML → 校验 → 保存到磁盘 → 发布（SIGHUP reload）" />
      {err && <p className="err">{err}</p>}
      <div className="panel">
        <div className="panel-head">
          <h2>ingress.yaml</h2>
          <div className="toolbar">
            <button type="button" className="btn" onClick={validate}>
              校验
            </button>
            <button type="button" className="btn" onClick={showDiff}>
              查看变更
            </button>
            <button type="button" className="btn" onClick={save}>
              保存到 YAML
            </button>
            <button type="button" className="btn btn-primary" onClick={() => setPublishOpen(true)}>
              发布
            </button>
          </div>
        </div>
        <div className="panel-body">
          <textarea
            className="code"
            spellCheck={false}
            value={content}
            onChange={(e) => {
              setContent(e.target.value)
              if (loaded.current) setValidateOut('')
            }}
          />
          <div style={{ marginTop: 12 }} dangerouslySetInnerHTML={{ __html: validateOut }} />
        </div>
      </div>
      <PublishModal
        open={publishOpen}
        configPath={path}
        content={content}
        onClose={() => setPublishOpen(false)}
        onDone={() => {
          setSaved(content)
          show('配置已保存并 reload')
        }}
      />
      <DiffModal open={diffOpen} diffHtml={diffHtml} onClose={() => setDiffOpen(false)} />
      {toast && <ToastContainer message={toast.message} type={toast.type} onDone={clear} />}
    </div>
  )
}
