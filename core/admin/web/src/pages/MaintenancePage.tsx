import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { RefreshCw } from 'lucide-react'
import { PageHeader } from '../components/PageHeader'
import { GlobalMaintenanceFormFields } from '../components/config/GlobalMaintenanceFormFields'
import { ToastContainer, useToast } from '../components/Toast'
import { api, type RouteRow } from '../api/client'
import {
  emptyGlobalMaintenanceForm,
  globalMaintenanceFromDoc,
  globalMaintenanceConfigured,
  maintenanceHostCount,
  patchGlobalMaintenance,
} from '../lib/maintenance'
import { parseModuleDoc, stringifyModuleDoc } from '../lib/ingressModuleForms'

function maintenanceBadge(label: string) {
  if (label === 'on') return <span className="badge badge-block">整规则</span>
  if (label === 'partial') return <span className="badge badge-wildcard">部分 Host</span>
  return null
}

export function MaintenancePage() {
  const [configPath, setConfigPath] = useState('')
  const [savedYAML, setSavedYAML] = useState('')
  const [form, setForm] = useState(emptyGlobalMaintenanceForm())
  const [savedForm, setSavedForm] = useState(emptyGlobalMaintenanceForm())
  const [routes, setRoutes] = useState<RouteRow[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [err, setErr] = useState('')
  const { toast, show, clear } = useToast()

  const load = useCallback(() => {
    setLoading(true)
    setErr('')
    Promise.all([api.getConfig(), api.routes()])
      .then(([cfg, rows]) => {
        setConfigPath(cfg.path)
        setSavedYAML(cfg.content)
        return api.configModules(cfg.content).then((modules) => {
          const maint = modules.find((m) => m.id === 'maintenance')
          const doc = parseModuleDoc(maint?.yaml ?? '')
          const nextForm = globalMaintenanceFromDoc(doc)
          setForm(nextForm)
          setSavedForm(nextForm)
          setRoutes(Array.isArray(rows) ? rows : [])
          setLoading(false)
        })
      })
      .catch((e: Error) => {
        setErr(e.message)
        setLoading(false)
      })
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const dirty = useMemo(
    () => JSON.stringify(form) !== JSON.stringify(savedForm),
    [form, savedForm],
  )

  const ruleRows = useMemo(
    () => routes.filter((r) => r.path_index < 0 && r.maintenance),
    [routes],
  )

  const globalHostCount = maintenanceHostCount(form)
  const globalActive = globalHostCount > 0

  const save = async () => {
    setSaving(true)
    setErr('')
    try {
      const doc = patchGlobalMaintenance({}, form)
      const moduleYAML = stringifyModuleDoc(doc)
      const merged = await api.mergeConfigModule(savedYAML, 'maintenance', moduleYAML)
      await api.validateConfig(merged.content)
      await api.putConfig(merged.content, 'save')
      setSavedYAML(merged.content)
      setSavedForm(form)
      show('已保存全局维护配置')
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e)
      setErr(msg)
      show(msg, 'error')
    } finally {
      setSaving(false)
    }
  }

  const publish = async () => {
    setSaving(true)
    setErr('')
    try {
      const doc = patchGlobalMaintenance({}, form)
      const moduleYAML = stringifyModuleDoc(doc)
      const merged = await api.mergeConfigModule(savedYAML, 'maintenance', moduleYAML)
      await api.validateConfig(merged.content)
      await api.putConfig(merged.content, 'publish')
      await api.reload()
      setSavedYAML(merged.content)
      setSavedForm(form)
      show('已发布并 reload')
      load()
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e)
      setErr(msg)
      show(msg, 'error')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="page">
      <PageHeader
        title="维护"
        desc="全局 maintenance.hosts 登记与默认 503 响应；规则级 scope 在路由编辑器中配置"
        actions={
          <button type="button" className="btn btn-sm" onClick={load} disabled={loading}>
            <RefreshCw size={14} aria-hidden /> 刷新
          </button>
        }
      />
      {err && <p className="err">{err}</p>}

      <div className="cards">
        <div className={`card ${globalActive ? 'warn' : ''}`}>
          <div className="label">全局维护域名</div>
          <div className="value">{globalHostCount}</div>
          <div className="sub">{globalActive ? 'maintenance.hosts 已配置' : '未登记全局域名'}</div>
        </div>
        <div className="card">
          <div className="label">规则级维护</div>
          <div className="value">{ruleRows.length}</div>
          <div className="sub">Host 规则 backend.service.maintenance</div>
        </div>
        <div className="card">
          <div className="label">配置文件</div>
          <div className="value" style={{ fontSize: '0.85rem', wordBreak: 'break-all' }}>
            {configPath || '—'}
          </div>
          <div className="sub">
            <Link to="/config">打开配置中心</Link>
          </div>
        </div>
      </div>

      <div className="panel">
        <div className="panel-head">
          <h2>全局维护 maintenance</h2>
          <div className="toolbar">
            {dirty ? <span className="config-draft-badge">未保存</span> : null}
            <button type="button" className="btn btn-sm" disabled={!dirty || saving} onClick={() => void save()}>
              保存
            </button>
            <button
              type="button"
              className="btn btn-sm btn-primary"
              disabled={!dirty || saving}
              onClick={() => void publish()}
            >
              发布 reload
            </button>
          </div>
        </div>
        <div className="panel-body">
          {loading ? (
            <p className="empty-hint">加载中…</p>
          ) : (
            <GlobalMaintenanceFormFields form={form} onChange={setForm} />
          )}
          {!loading && !globalMaintenanceConfigured(form) ? (
            <p className="form-hint" style={{ marginTop: '1rem' }}>
              未配置任何全局维护字段。添加 <code>maintenance.hosts</code> 后，匹配路由且 Host 命中的请求会返回
              503（跳过 auth）。
            </p>
          ) : null}
        </div>
      </div>

      <div className="panel">
        <div className="panel-head">
          <h2>规则级维护 ({ruleRows.length})</h2>
        </div>
        <div className="panel-body panel-table-wrap">
          {ruleRows.length === 0 ? (
            <p className="empty-hint">
              暂无启用的规则级维护。在{' '}
              <Link to="/config">配置 → 路由规则</Link> 或{' '}
              <Link to="/routes">路由详情</Link> 中编辑 backend.service.maintenance。
            </p>
          ) : (
            <table className="data">
              <thead>
                <tr>
                  <th>Host 规则</th>
                  <th>Backend</th>
                  <th>目标</th>
                  <th>维护</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                {ruleRows.map((r) => (
                  <tr key={r.id}>
                    <td>
                      <code>{r.host}</code>
                    </td>
                    <td>{r.backend_type}</td>
                    <td>
                      <code>{r.target}</code>
                    </td>
                    <td>{maintenanceBadge(r.maintenance ?? '')}</td>
                    <td>
                      <Link to={`/routes/${r.rule_index}/${r.path_index}`} className="btn btn-ghost btn-sm">
                        详情
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>

      {toast && <ToastContainer message={toast.message} type={toast.type} onDone={clear} />}
    </div>
  )
}
