import { useEffect, useMemo, useState } from 'react'
import { HoverTooltip } from '../EllipsisTooltip'
import { FormCheckbox } from '../Form'
import {
  filterPermissionCatalog,
  flattenPermissionCatalog,
  groupPermissionsByMenu,
  RBAC_UNASSIGNED_MENU_KEY,
  type RbacFlatMenuSection,
} from '../../lib/rbacMenuCatalog'
import type { RBACPermissionRow } from '../../api/client'

type Props = {
  permissions: RBACPermissionRow[]
  value: number[]
  onChange: (ids: number[]) => void
  disabled?: boolean
  /** Case-insensitive filter on name, code, description, menu label. */
  search?: string
}

function selectedCount(section: RbacFlatMenuSection, value: number[]): number {
  const ids = new Set(section.permissions.map((p) => p.id))
  return value.filter((id) => ids.has(id)).length
}

export function PermissionPicker({ permissions, value, onChange, disabled, search = '' }: Props) {
  const menuSections = useMemo(() => {
    const grouped = groupPermissionsByMenu(permissions)
    return flattenPermissionCatalog(filterPermissionCatalog(grouped, search))
  }, [permissions, search])

  const [activeKey, setActiveKey] = useState('')

  useEffect(() => {
    setActiveKey((prev) => {
      if (menuSections.length === 0) return ''
      if (prev && menuSections.some((s) => s.menu.key === prev)) return prev
      return menuSections[0].menu.key
    })
  }, [menuSections])

  const active =
    menuSections.find((s) => s.menu.key === activeKey) ?? menuSections[0] ?? null

  const navByGroup = useMemo(() => {
    const map = new Map<string, RbacFlatMenuSection[]>()
    for (const section of menuSections) {
      const list = map.get(section.navGroup) ?? []
      list.push(section)
      map.set(section.navGroup, list)
    }
    return map
  }, [menuSections])

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

  if (menuSections.length === 0) {
    return <p className="empty-hint">无匹配权限，请调整搜索关键词</p>
  }

  return (
    <div className="rbac-permission-picker-split">
      <nav className="rbac-permission-picker-nav" aria-label="按菜单选择权限">
        {[...navByGroup.entries()].map(([navGroup, sections]) => (
          <div key={navGroup} className="rbac-permission-picker-nav-group">
            <div className="rbac-permission-picker-nav-heading">{navGroup}</div>
            {sections.map((section) => {
              const picked = selectedCount(section, value)
              const isActive = section.menu.key === active?.menu.key
              return (
                <button
                  key={section.menu.key}
                  type="button"
                  className={`rbac-permission-picker-nav-item${isActive ? ' active' : ''}`}
                  onClick={() => setActiveKey(section.menu.key)}
                >
                  <span className="rbac-permission-picker-nav-label">{section.menu.label}</span>
                  <span className="rbac-permission-picker-nav-meta">
                    {picked > 0 ? `${picked}/` : ''}
                    {section.permissions.length}
                  </span>
                </button>
              )
            })}
          </div>
        ))}
      </nav>
      <div className="rbac-permission-picker-detail">
        {active ? (
          <>
            <header className="rbac-permission-picker-detail-head">
              <h3>{active.menu.label}</h3>
              {active.menu.key === RBAC_UNASSIGNED_MENU_KEY ? (
                <span className="chart-hint">自定义或未映射到侧栏的权限</span>
              ) : null}
            </header>
            <div className="rbac-permission-list">
              {active.permissions.map((perm) => (
                <label key={perm.id} className="rbac-permission-item">
                  <HoverTooltip content={perm.description ?? ''} className="rbac-permission-item-trigger">
                    <FormCheckbox
                      label={perm.name}
                      checked={value.includes(perm.id)}
                      onChange={(checked) => toggle(perm.id, checked)}
                    />
                  </HoverTooltip>
                </label>
              ))}
            </div>
          </>
        ) : null}
      </div>
    </div>
  )
}
