import { useEffect, useState, type ReactNode } from 'react'
import { PageHeader } from '../components/PageHeader'
import { api, type SettingsView } from '../api/client'
import {
  DEFAULT_PREFERENCES,
  displayPath,
  loadPreferences,
  savePreferences,
  type UIPreferences,
} from '../lib/preferences'

function boolLabel(v: boolean) {
  return v ? '是' : '否'
}

function PathValue({ path }: { path: string }) {
  const p = displayPath(path)
  return (
    <code className="settings-path" title={p.full || undefined}>
      {p.display}
    </code>
  )
}

function SettingsRow({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="settings-row">
      <div className="settings-label">{label}</div>
      <div className="settings-value">{children}</div>
    </div>
  )
}

export function SettingsPage() {
  const [data, setData] = useState<SettingsView | null>(null)
  const [prefs, setPrefs] = useState<UIPreferences>(() => loadPreferences())
  const [saved, setSaved] = useState('')
  const [err, setErr] = useState('')

  useEffect(() => {
    api
      .settings()
      .then(setData)
      .catch((e: Error) => setErr(e.message))
  }, [])

  const applyPrefs = () => {
    savePreferences(prefs)
    setSaved('已保存界面偏好（仅本浏览器）')
    window.setTimeout(() => setSaved(''), 2500)
  }

  const resetPrefs = () => {
    setPrefs({ ...DEFAULT_PREFERENCES })
    savePreferences(DEFAULT_PREFERENCES)
    setSaved('已恢复默认偏好')
    window.setTimeout(() => setSaved(''), 2500)
  }

  return (
    <div className="page">
      <PageHeader
        title="设置"
        desc="Admin 服务配置、Ingress 集成路径、数据存储与界面偏好"
      />
      {err && <p className="err">{err}</p>}
      {saved && <p className="settings-saved">{saved}</p>}

      <div className="settings-grid">
        <div className="panel">
          <div className="panel-head">
            <h2>Admin 服务</h2>
            <span className="chart-hint">修改 ingress.yaml 中 admin 段后需重启</span>
          </div>
          <div className="panel-body settings-body">
            <SettingsRow label="已启用">{boolLabel(Boolean(data?.admin.enabled))}</SettingsRow>
            <SettingsRow label="监听端口">{data?.admin.port ?? '—'}</SettingsRow>
            <SettingsRow label="Dev 代理">{boolLabel(Boolean(data?.admin.dev_proxy))}</SettingsRow>
            <SettingsRow label="嵌入 UI">{boolLabel(Boolean(data?.admin.ui_embedded))}</SettingsRow>
          </div>
        </div>

        <div className="panel">
          <div className="panel-head">
            <h2>Ingress 集成</h2>
          </div>
          <div className="panel-body settings-body">
            <SettingsRow label="ingress.yaml">
              {data ? <PathValue path={data.ingress.config_path} /> : '—'}
            </SettingsRow>
            <SettingsRow label="配置 hash">
              <code>{data?.ingress.config_hash || '—'}</code>
            </SettingsRow>
            <SettingsRow label="PID 文件">
              {data ? <PathValue path={data.ingress.pid_file} /> : '—'}
            </SettingsRow>
            <SettingsRow label="热加载">
              <span className={data?.ingress.reload_ready ? 'text-ok' : 'text-warn'}>
                {data?.ingress.reload_ready ? '就绪' : '不可用'}
              </span>
            </SettingsRow>
            <SettingsRow label="access.log">
              {data ? <PathValue path={data.ingress.log_path} /> : '—'}
              {data?.logs.access_exists === false && data.logs.access_configured ? (
                <span className="settings-hint"> · 文件不存在</span>
              ) : null}
            </SettingsRow>
            <SettingsRow label="error.log">
              {data ? <PathValue path={data.ingress.error_log_path} /> : '—'}
              {data?.logs.error_exists === false && data.logs.error_configured ? (
                <span className="settings-hint"> · 文件不存在</span>
              ) : null}
            </SettingsRow>
          </div>
        </div>

        <div className="panel">
          <div className="panel-head">
            <h2>数据存储</h2>
          </div>
          <div className="panel-body settings-body">
            <SettingsRow label="引擎">{data?.database.driver ?? '—'}</SettingsRow>
            <SettingsRow label="DSN">
              {data?.database.dsn ? <PathValue path={data.database.dsn.replace(/^file:/, '')} /> : '—'}
            </SettingsRow>
            <SettingsRow label="WAF 事件">{data?.database.waf_events ?? '—'}</SettingsRow>
            <SettingsRow label="审计日志">{data?.database.audit_logs ?? '—'}</SettingsRow>
            <SettingsRow label="配置版本">{data?.database.config_revisions ?? '—'}</SettingsRow>
            <p className="settings-note">
              清空样例数据：<code>rm -f examples/admin-console/admin.db</code> 后重启 ingress（空库会
              bootstrap seed）。
            </p>
          </div>
        </div>

        <div className="panel">
          <div className="panel-head">
            <h2>界面偏好</h2>
            <span className="chart-hint">保存在浏览器 localStorage</span>
          </div>
          <div className="panel-body settings-body">
            <SettingsRow label="日志实时刷新">
              <select
                value={prefs.logLiveIntervalMs}
                onChange={(e) =>
                  setPrefs((p) => ({ ...p, logLiveIntervalMs: Number(e.target.value) }))
                }
              >
                <option value={1000}>1 秒</option>
                <option value={2000}>2 秒</option>
                <option value={5000}>5 秒</option>
                <option value={10000}>10 秒</option>
                <option value={30000}>30 秒</option>
              </select>
            </SettingsRow>
            <SettingsRow label="总览指标窗口">
              <select
                value={prefs.metricsWindow}
                onChange={(e) => setPrefs((p) => ({ ...p, metricsWindow: e.target.value }))}
              >
                <option value="5m">5 分钟</option>
                <option value="15m">15 分钟</option>
                <option value="1h">1 小时</option>
              </select>
            </SettingsRow>
            <div className="toolbar" style={{ marginTop: 12 }}>
              <button type="button" className="btn btn-primary" onClick={applyPrefs}>
                保存偏好
              </button>
              <button type="button" className="btn btn-ghost" onClick={resetPrefs}>
                恢复默认
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
