import { useMemo } from 'react'
import { FormCheckbox } from '../Form'
import { groupPermissionsByMenu } from '../../lib/rbacMenuCatalog'
import type { RBACPermissionRow } from '../../api/client'

type Props = {
  permissions: RBACPermissionRow[]
  value: number[]
  onChange: (ids: number[]) => void
  disabled?: boolean
}

export function PermissionPicker({ permissions, value, onChange, disabled }: Props) {
  const catalog = useMemo(() => groupPermissionsByMenu(permissions), [permissions])

  const toggle = (id: number, checked: boolean) => {
    if (disabled) return
    if (checked) {
      onChange([...new Set([...value, id])])
      return
    }
    onChange(value.filter((item) => item !== id))
  }

  if (permissions.length === 0) {
    return <p className="empty-hint">暂无可用权限</p>
  }

  return (
    <div className="rbac-permission-picker">
      {catalog.navGroups.map((section) => (
        <section key={section.navGroup} className="rbac-permission-nav-group">
          <h3 className="rbac-permission-nav-title">{section.navGroup}</h3>
          {section.menus.map(({ menu, permissions: menuPerms }) => (
            <div key={menu.key} className="rbac-permission-group">
              <h4>{menu.label}</h4>
              <div className="rbac-permission-list">
                {menuPerms.map((perm) => (
                  <label key={perm.id} className="rbac-permission-item">
                    <FormCheckbox
                      label={perm.name}
                      checked={value.includes(perm.id)}
                      onChange={(checked) => toggle(perm.id, checked)}
                    />
                    <code className="rbac-permission-item-code">{perm.code}</code>
                    {perm.description ? <span className="chart-hint">{perm.description}</span> : null}
                  </label>
                ))}
              </div>
            </div>
          ))}
        </section>
      ))}
      {catalog.unassigned.length > 0 ? (
        <section className="rbac-permission-nav-group">
          <h3 className="rbac-permission-nav-title">未关联菜单</h3>
          <div className="rbac-permission-list">
            {catalog.unassigned.map((perm) => (
              <label key={perm.id} className="rbac-permission-item">
                <FormCheckbox
                  label={perm.name}
                  checked={value.includes(perm.id)}
                  onChange={(checked) => toggle(perm.id, checked)}
                />
                <code className="rbac-permission-item-code">{perm.code}</code>
              </label>
            ))}
          </div>
        </section>
      ) : null}
    </div>
  )
}
