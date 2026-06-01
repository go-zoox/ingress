import { useCallback, useEffect, useMemo, useState } from 'react'
import { Plus, RefreshCw } from 'lucide-react'
import { Drawer } from '../../components/Drawer'
import { FormCheckbox, FormField, FormGrid } from '../../components/Form'
import { EllipsisTooltip } from '../../components/EllipsisTooltip'
import { PageHeader } from '../../components/PageHeader'
import { ToastContainer, useToast } from '../../components/Toast'
import { api, type RBACRoleRow, type RBACUserInput, type RBACUserRow } from '../../api/client'

type UserDraft = {
  username: string
  display_name: string
  email: string
  password: string
  enabled: boolean
  role_ids: number[]
}

const emptyDraft = (): UserDraft => ({
  username: '',
  display_name: '',
  email: '',
  password: '',
  enabled: true,
  role_ids: [],
})

export function RbacUsersPage() {
  const [users, setUsers] = useState<RBACUserRow[]>([])
  const [roles, setRoles] = useState<RBACRoleRow[]>([])
  const [loading, setLoading] = useState(true)
  const [err, setErr] = useState('')
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [passwordOpen, setPasswordOpen] = useState(false)
  const [editUser, setEditUser] = useState<RBACUserRow | null>(null)
  const [draft, setDraft] = useState<UserDraft>(emptyDraft())
  const [passwordDraft, setPasswordDraft] = useState('')
  const [saving, setSaving] = useState(false)
  const { toast, show, clear } = useToast()

  const load = useCallback(() => {
    setLoading(true)
    setErr('')
    Promise.all([api.rbacUsers(), api.rbacRoles()])
      .then(([userRows, roleRows]) => {
        setUsers(userRows)
        setRoles(roleRows)
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
    setEditUser(null)
    setDraft(emptyDraft())
    setDrawerOpen(true)
  }

  const openEdit = (user: RBACUserRow) => {
    setEditUser(user)
    setDraft({
      username: user.username,
      display_name: user.display_name,
      email: user.email ?? '',
      password: '',
      enabled: user.enabled,
      role_ids: [...user.role_ids],
    })
    setDrawerOpen(true)
  }

  const openPassword = (user: RBACUserRow) => {
    setEditUser(user)
    setPasswordDraft('')
    setPasswordOpen(true)
  }

  const toggleRole = (roleId: number, checked: boolean) => {
    setDraft((prev) => ({
      ...prev,
      role_ids: checked
        ? [...new Set([...prev.role_ids, roleId])]
        : prev.role_ids.filter((id) => id !== roleId),
    }))
  }

  const saveUser = async () => {
    setSaving(true)
    try {
      const body: RBACUserInput = {
        username: draft.username,
        display_name: draft.display_name,
        email: draft.email,
        enabled: draft.enabled,
        role_ids: draft.role_ids,
      }
      if (editUser) {
        await api.updateRbacUser(editUser.id, body)
        show(`已更新用户 ${draft.username}`)
      } else {
        body.password = draft.password
        await api.createRbacUser(body)
        show(`已创建用户 ${draft.username}`)
      }
      setDrawerOpen(false)
      load()
    } catch (e: unknown) {
      show(e instanceof Error ? e.message : String(e), 'error')
    } finally {
      setSaving(false)
    }
  }

  const savePassword = async () => {
    if (!editUser) return
    setSaving(true)
    try {
      await api.updateRbacUserPassword(editUser.id, passwordDraft)
      show(`已重置 ${editUser.username} 的密码`)
      setPasswordOpen(false)
    } catch (e: unknown) {
      show(e instanceof Error ? e.message : String(e), 'error')
    } finally {
      setSaving(false)
    }
  }

  const removeUser = async (user: RBACUserRow) => {
    if (!window.confirm(`确定删除用户 ${user.username}？`)) return
    try {
      await api.deleteRbacUser(user.id)
      show(`已删除用户 ${user.username}`)
      load()
    } catch (e: unknown) {
      show(e instanceof Error ? e.message : String(e), 'error')
    }
  }

  const roleMap = useMemo(() => new Map(roles.map((role) => [role.id, role.name])), [roles])

  return (
    <div className="page">
      <PageHeader
        title="用户管理"
        desc="Admin Console 操作员账号；通过角色继承 RBAC 权限"
        actions={
          <>
            <button type="button" className="btn btn-sm" onClick={load}>
              <RefreshCw size={14} aria-hidden /> 刷新
            </button>
            <button type="button" className="btn btn-sm btn-primary" onClick={openCreate}>
              <Plus size={14} aria-hidden /> 新建用户
            </button>
          </>
        }
      />
      {err ? <p className="err">{err}</p> : null}

      <div className="panel">
        <div className="panel-head">
          <h2>用户列表</h2>
        </div>
        <div className="panel-body panel-table-wrap">
          {loading ? (
            <p className="empty-hint">加载中…</p>
          ) : users.length === 0 ? (
            <p className="empty-hint">暂无用户</p>
          ) : (
            <table className="data rbac-data-table">
              <thead>
                <tr>
                  <th className="col-id">ID</th>
                  <th className="col-name">名称</th>
                  <th className="col-code">标识</th>
                  <th className="col-email">邮箱</th>
                  <th className="col-roles">角色</th>
                  <th className="col-status">状态</th>
                  <th className="col-type">类型</th>
                  <th className="col-actions">操作</th>
                </tr>
              </thead>
              <tbody>
                {users.map((user) => {
                  const roleLabels =
                    user.role_ids.length === 0
                      ? ''
                      : user.role_ids.map((id) => roleMap.get(id) ?? `#${id}`).join('、')
                  return (
                  <tr key={user.id}>
                    <td className="col-id">{user.id}</td>
                    <td className="col-name">
                      <EllipsisTooltip text={user.display_name} className="rbac-cell-ellipsis" />
                    </td>
                    <td className="col-code">
                      <EllipsisTooltip text={user.username} className="rbac-cell-ellipsis rbac-cell-code" />
                    </td>
                    <td className="col-email">
                      <EllipsisTooltip text={user.email ?? ''} className="rbac-cell-ellipsis" />
                    </td>
                    <td className="col-roles">
                      <EllipsisTooltip text={roleLabels} className="rbac-cell-ellipsis" />
                    </td>
                    <td className="col-status">
                      <span className={`badge ${user.enabled ? 'badge-exact' : 'badge-block'}`}>
                        {user.enabled ? '启用' : '禁用'}
                      </span>
                    </td>
                    <td className="col-type">{user.builtin ? '内置' : '自定义'}</td>
                    <td className="col-actions">
                      <div className="table-actions">
                        <button type="button" className="btn btn-sm btn-ghost" onClick={() => openEdit(user)}>
                          编辑
                        </button>
                        <button type="button" className="btn btn-sm btn-ghost" onClick={() => openPassword(user)}>
                          重置密码
                        </button>
                        {!user.builtin ? (
                          <button type="button" className="btn btn-sm btn-ghost" onClick={() => removeUser(user)}>
                            删除
                          </button>
                        ) : null}
                      </div>
                    </td>
                  </tr>
                  )
                })}
              </tbody>
            </table>
          )}
        </div>
      </div>

      <Drawer
        open={drawerOpen}
        title={editUser ? `编辑用户 · ${editUser.username}` : '新建用户'}
        onClose={() => setDrawerOpen(false)}
        footer={
          <>
            <button type="button" className="btn btn-ghost" onClick={() => setDrawerOpen(false)}>
              取消
            </button>
            <button type="button" className="btn btn-primary" disabled={saving} onClick={saveUser}>
              保存
            </button>
          </>
        }
      >
        <FormGrid columns={1}>
          <FormField
            label="显示名称"
            value={draft.display_name}
            onChange={(e) => setDraft((d) => ({ ...d, display_name: e.target.value }))}
          />
          <FormField
            label="用户名"
            hint="登录标识，小写字母、数字、下划线"
            value={draft.username}
            readOnly={!!editUser?.builtin}
            onChange={(e) => setDraft((d) => ({ ...d, username: e.target.value }))}
          />
          <FormField
            label="邮箱"
            type="email"
            value={draft.email}
            onChange={(e) => setDraft((d) => ({ ...d, email: e.target.value }))}
          />
          {!editUser ? (
            <FormField
              label="初始密码"
              type="password"
              hint="至少 6 位"
              value={draft.password}
              onChange={(e) => setDraft((d) => ({ ...d, password: e.target.value }))}
            />
          ) : null}
          <FormCheckbox
            label="启用账号"
            checked={draft.enabled}
            onChange={(enabled) => setDraft((d) => ({ ...d, enabled }))}
          />
        </FormGrid>
        <section className="form-section">
          <h4 className="form-section-title">角色</h4>
          <div className="form-section-body">
            {roles.length === 0 ? (
              <p className="empty-hint">暂无角色，请先在角色管理中创建</p>
            ) : (
              roles.map((role) => (
                <FormCheckbox
                  key={role.id}
                  label={`${role.name} (${role.code})`}
                  checked={draft.role_ids.includes(role.id)}
                  onChange={(checked) => toggleRole(role.id, checked)}
                />
              ))
            )}
          </div>
        </section>
      </Drawer>

      <Drawer
        open={passwordOpen}
        title={editUser ? `重置密码 · ${editUser.username}` : '重置密码'}
        onClose={() => setPasswordOpen(false)}
        footer={
          <>
            <button type="button" className="btn btn-ghost" onClick={() => setPasswordOpen(false)}>
              取消
            </button>
            <button type="button" className="btn btn-primary" disabled={saving} onClick={savePassword}>
              保存
            </button>
          </>
        }
      >
        <FormField
          label="新密码"
          type="password"
          hint="至少 6 位"
          full
          value={passwordDraft}
          onChange={(e) => setPasswordDraft(e.target.value)}
        />
      </Drawer>

      {toast && <ToastContainer message={toast.message} type={toast.type} onDone={clear} />}
    </div>
  )
}
