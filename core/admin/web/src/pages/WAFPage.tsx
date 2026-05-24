import { useCallback, useEffect, useState } from 'react'
import { PageHeader } from '../components/PageHeader'
import { api, type WAFEvent } from '../api/client'

export function WAFPage() {
  const [status, setStatus] = useState<Record<string, unknown> | null>(null)
  const [events, setEvents] = useState<WAFEvent[]>([])
  const [filter, setFilter] = useState('all')
  const [err, setErr] = useState('')
  const [toggling, setToggling] = useState(false)

  const load = useCallback(() => {
    api.status().then(setStatus).catch(() => setStatus(null))
    api
      .wafEvents()
      .then((data) => setEvents(Array.isArray(data) ? data : []))
      .catch((e: Error) => setErr(e.message))
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const rows = events.filter((e) => filter === 'all' || e.action === filter)
  const configWafLabel = status?.waf_enabled
    ? status.waf_log_only
      ? '仅审计'
      : '拦截'
    : '关闭'
  const runtimeWafEnabled = status?.waf_runtime_enabled !== undefined
    ? Boolean(status.waf_runtime_enabled)
    : Boolean(status?.waf_enabled)
  const configWafEnabled = Boolean(status?.waf_enabled)
  const overrideActive = runtimeWafEnabled !== configWafEnabled

  const handleToggle = async (enabled: boolean) => {
    setToggling(true)
    try {
      await api.wafToggle(enabled ? true : null)
      load() // refresh status
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e))
    } finally {
      setToggling(false)
    }
  }

  return (
    <div className="page">
      <PageHeader title="WAF" desc="全局基线与近期 block / audit 事件" />
      {err && <p className="err">{err}</p>}
      <div className="cards">
        <div className="card">
          <div className="label">状态</div>
          <div className="value">{status?.waf_enabled ? '已启用' : '关闭'}</div>
          <div className="sub">全局 waf.enabled</div>
        </div>
        <div className={`card ${status?.waf_log_only ? 'warn' : ''}`}>
          <div className="label">模式</div>
          <div className="value">{configWafLabel}</div>
          <div className="sub">log_only</div>
        </div>
        <div className="card">
          <div className="label">内置规则</div>
          <div className="value">已加载</div>
          <div className="sub">builtin: true</div>
        </div>
        <div className={`card ${overrideActive ? 'warn' : ''}`}>
          <div className="label">WAF 实时开关</div>
          <div className="value">
            <label className="live-toggle">
              <input
                type="checkbox"
                checked={runtimeWafEnabled}
                disabled={toggling}
                onChange={(e) => handleToggle(e.target.checked)}
              />{' '}
              {toggling ? '切换中…' : runtimeWafEnabled ? '已启用' : '已关闭'}
            </label>
          </div>
          <div className="sub">
            {overrideActive
              ? '运行时覆盖生效，与配置文件不同'
              : '无需 reload，立即生效'}
          </div>
        </div>
      </div>
      <div className="panel">
        <div className="panel-head">
          <h2>近期事件</h2>
          <select value={filter} onChange={(e) => setFilter(e.target.value)}>
            <option value="all">全部</option>
            <option value="block">仅 block</option>
            <option value="audit">仅 audit</option>
          </select>
        </div>
        <div className="panel-body panel-table-wrap">
          <table className="data">
            <thead>
              <tr>
                <th>时间</th>
                <th>动作</th>
                <th>规则</th>
                <th>Host</th>
                <th>Path</th>
                <th>客户端</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((e) => (
                <tr key={e.id}>
                  <td>{formatTime(e.created_at)}</td>
                  <td>
                    <span className={`badge badge-${e.action}`}>{e.action}</span>
                  </td>
                  <td>{e.rule}</td>
                  <td>{e.host}</td>
                  <td>
                    <code>{e.path}</code>
                  </td>
                  <td>{e.client_ip}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}

function formatTime(iso: string) {
  try {
    return new Date(iso).toLocaleTimeString('zh-CN', { hour12: false })
  } catch {
    return iso
  }
}
