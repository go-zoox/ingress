import { useCallback, useEffect, useState } from 'react'
import { Plus, RefreshCw } from 'lucide-react'
import { Drawer } from '../../components/Drawer'
import { FormField, FormGrid, FormTextareaField } from '../../components/Form'
import { PageHeader } from '../../components/PageHeader'
import { PermissionPicker } from '../../components/rbac/PermissionPicker'
import { ToastContainer, useToast } from '../../components/Toast'
import {
  api,
  type RBACPermissionRow,
  type RBACRoleInput,
  type RBACRoleRow,
} from '../../api/client'

type RoleDraft = {
  code: string
  name: string
  description: string
  permission_ids: number[]
}

const emptyDraft = (): RoleDraft => ({
  code: '',
  name: '',
  description: '',
  permission_ids: [],
})

export function RbacRolesPage() {
  const [roles, setRoles] = useState<RBACRoleRow[]>([])
  const [permissions, setPermissions] = useState<RBACPermissionRow[]>([])
  const [loading, setLoading] = useState(true)
  const [err, setErr] = useState('')
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [editRole, setEditRole] = useState<RBACRoleRow | null>(null)
  const [draft, setDraft] = useState<RoleDraft>(emptyDraft())
  const [saving, setSaving] = useState(false)
  const { toast, show, clear } = useToast()

  const load = useCallback(() => {
    setLoading(true)
    setErr('')
    Promise.all([api.rbacRoles(), api.rbacPermissions()])
      .then(([roleRows, permRows]) => {
        setRoles(roleRows)
        setPermissions(permRows)
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

  const openCreate = () => {
    setEditRole(null)
    setDraft(emptyDraft())
    setDrawerOpen(true)
  }

  const openEdit = (role: RBACRoleRow) => {
    setEditRole(role)
    setDraft({
      code: role.code,
      name: role.name,
      description: role.description ?? '',
      permission_ids: [...role.permission_ids],
    })
    setDrawerOpen(true)
  }

  const saveRole = async () => {
    setSaving(true)
    try {
      const body: RBACRoleInput = {
        code: draft.code,
        name: draft.name,
        description: draft.description,
        permission_ids: draft.permission_ids,
      }
      if (editRole) {
        await api.updateRbacRole(editRole.id, body)
        show(`已更新角色 ${draft.name}`)
      } else {
        await api.createRbacRole(body)
        show(`已创建角色 ${draft.name}`)
      }
      setDrawerOpen(false)
      load()
    } catch (e: unknown) {
      show(e instanceof Error ? e.message : String(e), 'error')
    } finally {
      setSaving(false)
    }
  }

  const removeRole = async (role: RBACRoleRow) => {
    if (!window.confirm(`确定删除角色 ${role.name}？`)) return
    try {
      await api.deleteRbacRole(role.id)
      show(`已删除角色 ${role.name}`)
      load()
    } catch (e: unknown) {
      show(e instanceof Error ? e.message : String(e), 'error')
    }
  }

  return (
    <div className="page">
      <PageHeader
        title="角色管理"
        desc="将权限组合为角色，再分配给用户"
        actions={
          <>
            <button type="button" className="btn btn-sm" onClick={load}>
              <RefreshCw size={14} aria-hidden /> 刷新
            </button>
            <button type="button" className="btn btn-sm btn-primary" onClick={openCreate}>
              <Plus size={14} aria-hidden /> 新建角色
            </button>
          </>
        }
      />
      {err ? <p className="err">{err}</p> : null}

      <div className="panel">
        <div className="panel-head">
          <h2>角色列表</h2>
        </div>
        <div className="panel-body panel-table-wrap">
          {loading ? (
            <p className="empty-hint">加载中…</p>
          ) : roles.length === 0 ? (
            <p className="empty-hint">暂无角色</p>
          ) : (
            <table className="data">
              <thead>
                <tr>
                  <th>标识</th>
                  <th>名称</th>
                  <th>描述</th>
                  <th>权限数</th>
                  <th>用户数</th>
                  <th>类型</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                {roles.map((role) => (
                  <tr key={role.id}>
                    <td><code>{role.code}</code></td>
                    <td>{role.name}</td>
                    <td>{role.description || '—'}</td>
                    <td>{role.permission_ids.length}</td>
                    <td>{role.user_count}</td>
                    <td>{role.builtin ? '内置' : '自定义'}</td>
                    <td>
                      <div className="table-actions">
                        <button type="button" className="btn btn-sm btn-ghost" onClick={() => openEdit(role)}>
                          编辑
                        </button>
                        {!role.builtin ? (
                          <button type="button" className="btn btn-sm btn-ghost" onClick={() => removeRole(role)}>
                            删除
                          </button>
                        ) : null}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>

      <Drawer
        open={drawerOpen}
        title={editRole ? `编辑角色 · ${editRole.name}` : '新建角色'}
        onClose={() => setDrawerOpen(false)}
        width={560}
        footer={
          <>
            <button type="button" className="btn btn-ghost" onClick={() => setDrawerOpen(false)}>
              取消
            </button>
            <button type="button" className="btn btn-primary" disabled={saving} onClick={saveRole}>
              保存
            </button>
          </>
        }
      >
        <FormGrid columns={1}>
          <FormField
            label="角色标识"
            hint="小写字母、数字、连字符"
            value={draft.code}
            readOnly={!!editRole?.builtin}
            onChange={(e) => setDraft((d) => ({ ...d, code: e.target.value }))}
          />
          <FormField
            label="显示名称"
            value={draft.name}
            onChange={(e) => setDraft((d) => ({ ...d, name: e.target.value }))}
          />
          <FormTextareaField
            label="描述"
            rows={2}
            value={draft.description}
            onChange={(e) => setDraft((d) => ({ ...d, description: e.target.value }))}
          />
        </FormGrid>
        <section className="form-section">
          <h4 className="form-section-title">权限</h4>
          <div className="form-section-body">
            <PermissionPicker
              permissions={permissions}
              value={draft.permission_ids}
              onChange={(permission_ids) => setDraft((d) => ({ ...d, permission_ids }))}
            />
          </div>
        </section>
      </Drawer>

      {toast && <ToastContainer message={toast.message} type={toast.type} onDone={clear} />}
    </div>
  )
}
