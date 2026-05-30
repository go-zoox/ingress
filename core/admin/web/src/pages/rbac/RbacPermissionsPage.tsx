import { useCallback, useEffect, useMemo, useState } from 'react'
import { Plus, RefreshCw } from 'lucide-react'
import { Drawer } from '../../components/Drawer'
import { FormField, FormGrid, FormTextareaField } from '../../components/Form'
import { PageHeader } from '../../components/PageHeader'
import { ToastContainer, useToast } from '../../components/Toast'
import { api, type RBACPermissionInput, type RBACPermissionRow } from '../../api/client'

type PermissionDraft = {
  code: string
  name: string
  group: string
  description: string
}

const emptyDraft = (): PermissionDraft => ({
  code: '',
  name: '',
  group: '自定义',
  description: '',
})

export function RbacPermissionsPage() {
  const [permissions, setPermissions] = useState<RBACPermissionRow[]>([])
  const [loading, setLoading] = useState(true)
  const [err, setErr] = useState('')
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [editPerm, setEditPerm] = useState<RBACPermissionRow | null>(null)
  const [draft, setDraft] = useState<PermissionDraft>(emptyDraft())
  const [saving, setSaving] = useState(false)
  const { toast, show, clear } = useToast()

  const load = useCallback(() => {
    setLoading(true)
    setErr('')
    api
      .rbacPermissions()
      .then((rows) => {
        setPermissions(rows)
        setLoading(false)
      })
      .catch((e: Error) => {
        setErr(e.message)
        setLoading(false)
      })
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const groups = useMemo(() => {
    const set = new Set(permissions.map((p) => p.group))
    return [...set].sort((a, b) => a.localeCompare(b, 'zh-CN'))
  }, [permissions])

  const openCreate = () => {
    setEditPerm(null)
    setDraft(emptyDraft())
    setDrawerOpen(true)
  }

  const openEdit = (perm: RBACPermissionRow) => {
    setEditPerm(perm)
    setDraft({
      code: perm.code,
      name: perm.name,
      group: perm.group,
      description: perm.description ?? '',
    })
    setDrawerOpen(true)
  }

  const savePermission = async () => {
    setSaving(true)
    try {
      const body: RBACPermissionInput = {
        code: draft.code,
        name: draft.name,
        group: draft.group,
        description: draft.description,
      }
      if (editPerm) {
        await api.updateRbacPermission(editPerm.id, body)
        show(`已更新权限 ${draft.name}`)
      } else {
        await api.createRbacPermission(body)
        show(`已创建权限 ${draft.name}`)
      }
      setDrawerOpen(false)
      load()
    } catch (e: unknown) {
      show(e instanceof Error ? e.message : String(e), 'error')
    } finally {
      setSaving(false)
    }
  }

  const removePermission = async (perm: RBACPermissionRow) => {
    if (!window.confirm(`确定删除权限 ${perm.code}？`)) return
    try {
      await api.deleteRbacPermission(perm.id)
      show(`已删除权限 ${perm.code}`)
      load()
    } catch (e: unknown) {
      show(e instanceof Error ? e.message : String(e), 'error')
    }
  }

  return (
    <div className="page">
      <PageHeader
        title="权限管理"
        desc="Admin Console 功能权限目录；内置权限随版本同步，可追加自定义权限"
        actions={
          <>
            <button type="button" className="btn btn-sm" onClick={load}>
              <RefreshCw size={14} aria-hidden /> 刷新
            </button>
            <button type="button" className="btn btn-sm btn-primary" onClick={openCreate}>
              <Plus size={14} aria-hidden /> 新建权限
            </button>
          </>
        }
      />
      {err ? <p className="err">{err}</p> : null}

      {groups.map((group) => {
        const rows = permissions.filter((p) => p.group === group)
        return (
          <div key={group} className="panel">
            <div className="panel-head">
              <h2>{group}</h2>
              <span className="chart-hint">{rows.length} 项</span>
            </div>
            <div className="panel-body panel-table-wrap">
              {loading ? (
                <p className="empty-hint">加载中…</p>
              ) : (
                <table className="data">
                  <thead>
                    <tr>
                      <th>标识</th>
                      <th>名称</th>
                      <th>描述</th>
                      <th>关联角色</th>
                      <th>类型</th>
                      <th>操作</th>
                    </tr>
                  </thead>
                  <tbody>
                    {rows.map((perm) => (
                      <tr key={perm.id}>
                        <td><code>{perm.code}</code></td>
                        <td>{perm.name}</td>
                        <td>{perm.description || '—'}</td>
                        <td>{perm.role_count}</td>
                        <td>{perm.builtin ? '内置' : '自定义'}</td>
                        <td>
                          <div className="table-actions">
                            {!perm.builtin ? (
                              <>
                                <button type="button" className="btn btn-sm btn-ghost" onClick={() => openEdit(perm)}>
                                  编辑
                                </button>
                                <button type="button" className="btn btn-sm btn-ghost" onClick={() => removePermission(perm)}>
                                  删除
                                </button>
                              </>
                            ) : (
                              <span className="chart-hint">只读</span>
                            )}
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          </div>
        )
      })}

      <Drawer
        open={drawerOpen}
        title={editPerm ? `编辑权限 · ${editPerm.code}` : '新建权限'}
        onClose={() => setDrawerOpen(false)}
        footer={
          <>
            <button type="button" className="btn btn-ghost" onClick={() => setDrawerOpen(false)}>
              取消
            </button>
            <button type="button" className="btn btn-primary" disabled={saving} onClick={savePermission}>
              保存
            </button>
          </>
        }
      >
        <FormGrid columns={1}>
          <FormField
            label="权限标识"
            hint="例如 custom:action"
            value={draft.code}
            readOnly={!!editPerm}
            onChange={(e) => setDraft((d) => ({ ...d, code: e.target.value }))}
          />
          <FormField
            label="显示名称"
            value={draft.name}
            onChange={(e) => setDraft((d) => ({ ...d, name: e.target.value }))}
          />
          <FormField
            label="分组"
            value={draft.group}
            onChange={(e) => setDraft((d) => ({ ...d, group: e.target.value }))}
          />
          <FormTextareaField
            label="描述"
            rows={2}
            value={draft.description}
            onChange={(e) => setDraft((d) => ({ ...d, description: e.target.value }))}
          />
        </FormGrid>
      </Drawer>

      {toast && <ToastContainer message={toast.message} type={toast.type} onDone={clear} />}
    </div>
  )
}
