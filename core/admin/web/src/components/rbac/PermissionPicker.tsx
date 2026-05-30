import { useMemo } from 'react'
import { FormCheckbox } from '../Form'
import type { RBACPermissionRow } from '../../api/client'

type Props = {
  permissions: RBACPermissionRow[]
  value: number[]
  onChange: (ids: number[]) => void
  disabled?: boolean
}

export function PermissionPicker({ permissions, value, onChange, disabled }: Props) {
  const groups = useMemo(() => {
    const map = new Map<string, RBACPermissionRow[]>()
    for (const perm of permissions) {
      const list = map.get(perm.group) ?? []
      list.push(perm)
      map.set(perm.group, list)
    }
    return [...map.entries()].sort(([a], [b]) => a.localeCompare(b, 'zh-CN'))
  }, [permissions])

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
      {groups.map(([group, items]) => (
        <section key={group} className="rbac-permission-group">
          <h4>{group}</h4>
          <div className="rbac-permission-list">
            {items.map((perm) => (
              <label key={perm.id} className="rbac-permission-item">
                <FormCheckbox
                  label={`${perm.name} (${perm.code})`}
                  checked={value.includes(perm.id)}
                  onChange={(checked) => toggle(perm.id, checked)}
                />
                {perm.description ? <span className="chart-hint">{perm.description}</span> : null}
              </label>
            ))}
          </div>
        </section>
      ))}
    </div>
  )
}
