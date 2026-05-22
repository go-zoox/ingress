import { useState } from 'react'
import { api } from '../api/client'

type Step = 'validate' | 'write' | 'reload'

export function PublishModal({
  open,
  configPath,
  content,
  onClose,
  onDone,
}: {
  open: boolean
  configPath: string
  content: string
  onClose: () => void
  onDone: () => void
}) {
  const [note, setNote] = useState('')
  const [status, setStatus] = useState('')
  const [stepState, setStepState] = useState<Record<Step, 'idle' | 'active' | 'done'>>({
    validate: 'idle',
    write: 'idle',
    reload: 'idle',
  })

  if (!open) return null

  const run = async () => {
    setStepState({ validate: 'active', write: 'idle', reload: 'idle' })
    setStatus('正在校验…')
    try {
      await api.validateConfig(content)
      setStepState({ validate: 'done', write: 'active', reload: 'idle' })
      setStatus('正在写入 YAML 并发送 reload…')
      await api.publishConfig(content, note.trim() || 'publish')
      setStepState({ validate: 'done', write: 'done', reload: 'done' })
      setStatus('发布成功。')
      setTimeout(() => {
        onDone()
        onClose()
        setNote('')
      }, 900)
    } catch (e) {
      setStatus(e instanceof Error ? e.message : String(e))
      setStepState({ validate: 'idle', write: 'idle', reload: 'idle' })
    }
  }

  const stepClass = (s: Step) => {
    const st = stepState[s]
    if (st === 'done') return 'done'
    if (st === 'active') return 'active'
    return ''
  }

  return (
    <div className="modal-overlay open" onClick={(e) => e.target === e.currentTarget && onClose()}>
      <div className="modal" role="dialog">
        <header>
          <h2>发布配置</h2>
        </header>
        <div className="content">
          <p style={{ marginTop: 0, color: 'var(--text-muted)', fontSize: 13 }}>
            将执行：<strong>validate</strong> → 写入 <code>{configPath}</code> →{' '}
            <strong>SIGHUP reload</strong>，并记录版本。
          </p>
          <label className="field-label">版本说明（可选）</label>
          <input
            type="text"
            className="field-input"
            placeholder="例如：调整 WAF 规则 / 新增 CDN 路由"
            value={note}
            onChange={(e) => setNote(e.target.value)}
          />
          <ul className="publish-steps">
            <li className={stepClass('validate')}>
              <span className="step-icon">1</span> 校验配置
            </li>
            <li className={stepClass('write')}>
              <span className="step-icon">2</span> 保存 YAML + 版本
            </li>
            <li className={stepClass('reload')}>
              <span className="step-icon">3</span> 热加载
            </li>
          </ul>
          <p style={{ fontSize: 13, margin: 0 }}>{status}</p>
        </div>
        <footer>
          <button type="button" className="btn" onClick={onClose}>
            取消
          </button>
          <button type="button" className="btn btn-primary" onClick={run}>
            开始发布
          </button>
        </footer>
      </div>
    </div>
  )
}
