import { useMemo } from 'react'
import { FormCheckbox } from '../Form'
import { filterPermissionCatalog, groupPermissionsByMenu } from '../../lib/rbacMenuCatalog'
import type { RBACPermissionRow } from '../../api/client'

type Props = {
  permissions: RBACPermissionRow[]
  value: number[]
  onChange: (ids: number[]) => void
  disabled?: boolean
  /** Case-insensitive filter on name, code, description, menu label. */
  search?: string
}

export function PermissionPicker({ permissions, value, onChange, disabled, search = '' }: Props) {
  const catalog = useMemo(() => {
    const grouped = groupPermissionsByMenu(permissions)
    return filterPermissionCatalog(grouped, search)
  }, [permissions, search])

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

  const hasResults = catalog.navGroups.length > 0 || catalog.unassigned.length > 0
  if (!hasResults) {
    return <p className="empty-hint">无匹配权限，请调整搜索关键词</p>
  }

  return (
    <div className="rbac-permission-picker">
      {catalog.navGroups.map((section) => {
        const sectionCount = section.menus.reduce((n, m) => n + m.permissions.length, 0)
        return (
          <div key={section.navGroup} className="panel rbac-permission-picker-panel">
            <div className="panel-head">
              <h2>{section.navGroup}</h2>
              <span className="chart-hint">{sectionCount} 项</span>
            </div>
            <div className="panel-body rbac-permission-picker-stack">
              {section.menus.map(({ menu, permissions: menuPerms }) => (
                <section key={menu.key} className="rbac-permission-picker-menu">
                  <div className="rbac-permissions-menu-head rbac-permissions-menu-head--inline">
                    <span className="rbac-permissions-menu-label">{menu.label}</span>
                    <span className="chart-hint">
                      <code>menu:{menu.key}</code> · {menuPerms.length} 项
                    </span>
                  </div>
                  <div className="rbac-permission-list">
                    {menuPerms.map((perm) => (
                      <label key={perm.id} className="rbac-permission-item">
                        <FormCheckbox
                          label={perm.name}
                          checked={value.includes(perm.id)}
                          onChange={(checked) => toggle(perm.id, checked)}
                        />
                        <code className="rbac-permission-item-code">{perm.code}</code>
                        {perm.description ? (
                          <span className="rbac-permission-item-desc chart-hint">{perm.description}</span>
                        ) : null}
                      </label>
                    ))}
                  </div>
                </section>
              ))}
            </div>
          </div>
        )
      })}
      {catalog.unassigned.length > 0 ? (
        <div className="panel rbac-permission-picker-panel">
          <div className="panel-head">
            <h2>未关联菜单</h2>
            <span className="chart-hint">{catalog.unassigned.length} 项</span>
          </div>
          <div className="panel-body">
            <div className="rbac-permission-list">
              {catalog.unassigned.map((perm) => (
                <label key={perm.id} className="rbac-permission-item">
                  <FormCheckbox
                    label={perm.name}
                    checked={value.includes(perm.id)}
                    onChange={(checked) => toggle(perm.id, checked)}
                  />
                  <code className="rbac-permission-item-code">{perm.code}</code>
                  {perm.description ? (
                    <span className="rbac-permission-item-desc chart-hint">{perm.description}</span>
                  ) : null}
                </label>
              ))}
            </div>
          </div>
        </div>
      ) : null}
    </div>
  )
}
