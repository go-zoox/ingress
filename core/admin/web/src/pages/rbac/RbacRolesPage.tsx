import { useCallback, useEffect, useState } from 'react'
import { Plus, RefreshCw } from 'lucide-react'
import { Drawer } from '../../components/Drawer'
import { FormField, FormGrid, FormTextareaField } from '../../components/Form'
import { EllipsisTooltip } from '../../components/EllipsisTooltip'
import { PageHeader } from '../../components/PageHeader'
import { RolePermissionsDrawer } from '../../components/rbac/RolePermissionsDrawer'
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
}

const emptyDraft = (): RoleDraft => ({
  code: '',
  name: '',
  description: '',
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
  const [permDrawerOpen, setPermDrawerOpen] = useState(false)
  const [permRole, setPermRole] = useState<RBACRoleRow | null>(null)
  const [permIds, setPermIds] = useState<number[]>([])
  const [permSaving, setPermSaving] = useState(false)
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
    })
    setDrawerOpen(true)
  }

  const openPermissions = (role: RBACRoleRow) => {
    setPermRole(role)
    setPermIds([...role.permission_ids])
    setPermDrawerOpen(true)
  }

  const saveRole = async () => {
    setSaving(true)
    try {
      const body: RBACRoleInput = {
        code: draft.code,
        name: draft.name,
        description: draft.description,
        permission_ids: editRole ? [...editRole.permission_ids] : [],
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

  const savePermissions = async () => {
    if (!permRole) return
    setPermSaving(true)
    try {
      const body: RBACRoleInput = {
        code: permRole.code,
        name: permRole.name,
        description: permRole.description ?? '',
        permission_ids: permIds,
      }
      await api.updateRbacRole(permRole.id, body)
      show(`已更新角色 ${permRole.name} 的权限`)
      setPermDrawerOpen(false)
      load()
    } catch (e: unknown) {
      show(e instanceof Error ? e.message : String(e), 'error')
    } finally {
      setPermSaving(false)
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
        desc="将权限组合为角色，再分配给用户；在列表中点「权限」按侧栏菜单勾选"
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
            <table className="data rbac-data-table rbac-roles-table">
              <thead>
                <tr>
                  <th className="col-id">ID</th>
                  <th className="col-name">名称</th>
                  <th className="col-code">标识</th>
                  <th className="col-desc">描述</th>
                  <th className="col-num">权限数</th>
                  <th className="col-num">用户数</th>
                  <th className="col-type">类型</th>
                  <th className="col-actions">操作</th>
                </tr>
              </thead>
              <tbody>
                {roles.map((role) => (
                  <tr key={role.id}>
                    <td className="col-id">{role.id}</td>
                    <td className="col-name">
                      <EllipsisTooltip text={role.name} className="rbac-cell-ellipsis" />
                    </td>
                    <td className="col-code">
                      <EllipsisTooltip text={role.code} className="rbac-cell-ellipsis rbac-cell-code" />
                    </td>
                    <td className="col-desc">
                      <EllipsisTooltip text={role.description ?? ''} className="rbac-cell-ellipsis" />
                    </td>
                    <td className="col-num">{role.permission_ids.length}</td>
                    <td className="col-num">{role.user_count}</td>
                    <td className="col-type">{role.builtin ? '内置' : '自定义'}</td>
                    <td className="col-actions">
                      <div className="table-actions">
                        <button type="button" className="btn btn-sm btn-ghost" onClick={() => openEdit(role)}>
                          编辑
                        </button>
                        <button type="button" className="btn btn-sm btn-ghost" onClick={() => openPermissions(role)}>
                          权限
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
        width={480}
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
            label="显示名称"
            value={draft.name}
            onChange={(e) => setDraft((d) => ({ ...d, name: e.target.value }))}
          />
          <FormField
            label="角色标识"
            hint="小写字母、数字、连字符"
            value={draft.code}
            readOnly={!!editRole?.builtin}
            onChange={(e) => setDraft((d) => ({ ...d, code: e.target.value }))}
          />
          <FormTextareaField
            label="描述"
            rows={2}
            value={draft.description}
            onChange={(e) => setDraft((d) => ({ ...d, description: e.target.value }))}
          />
        </FormGrid>
        {!editRole ? (
          <p className="chart-hint" style={{ marginTop: 16 }}>
            保存后可在列表中点击「权限」为该角色勾选菜单与操作权限。
          </p>
        ) : null}
      </Drawer>

      <RolePermissionsDrawer
        open={permDrawerOpen}
        role={permRole}
        permissions={permissions}
        value={permIds}
        saving={permSaving}
        onChange={setPermIds}
        onClose={() => setPermDrawerOpen(false)}
        onSave={savePermissions}
      />

      {toast && <ToastContainer message={toast.message} type={toast.type} onDone={clear} />}
    </div>
  )
}
