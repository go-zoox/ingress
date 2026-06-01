import { useEffect, useState } from 'react'
import { Drawer } from '../Drawer'
import { PermissionPicker } from './PermissionPicker'
import type { RBACPermissionRow, RBACRoleRow } from '../../api/client'

type Props = {
  open: boolean
  role: RBACRoleRow | null
  permissions: RBACPermissionRow[]
  value: number[]
  saving?: boolean
  onChange: (ids: number[]) => void
  onClose: () => void
  onSave: () => void
}

export function RolePermissionsDrawer({
  open,
  role,
  permissions,
  value,
  saving,
  onChange,
  onClose,
  onSave,
}: Props) {
  const [search, setSearch] = useState('')

  useEffect(() => {
    if (!open) setSearch('')
  }, [open])

  return (
    <Drawer
      open={open}
      title={role ? `角色权限 · ${role.name}` : '角色权限'}
      width={920}
      onClose={onClose}
      footer={
        <>
          <button type="button" className="btn btn-ghost" onClick={onClose}>
            取消
          </button>
          <button type="button" className="btn btn-primary" disabled={saving || !role} onClick={onSave}>
            保存
          </button>
        </>
      }
    >
      <div className="rbac-role-permissions-drawer">
        <div className="rbac-permission-picker-toolbar">
          <input
            type="search"
            className="rbac-permission-picker-search"
            placeholder="搜索名称、标识、描述、菜单…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            aria-label="搜索权限"
          />
          <span className="chart-hint rbac-permission-picker-summary">
            已选 <strong>{value.length}</strong> / {permissions.length} 项
          </span>
        </div>
        <div className="rbac-permission-picker-scroll">
          <PermissionPicker
            permissions={permissions}
            value={value}
            onChange={onChange}
            search={search}
          />
        </div>
      </div>
    </Drawer>
  )
}
