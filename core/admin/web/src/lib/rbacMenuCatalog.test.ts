import { describe, expect, it } from 'vitest'
import type { RBACPermissionRow } from '../api/client'
import { groupPermissionsByMenu } from './rbacMenuCatalog'

function perm(partial: Partial<RBACPermissionRow> & Pick<RBACPermissionRow, 'id' | 'code' | 'name'>): RBACPermissionRow {
  return {
    group: '监控',
    description: '',
    builtin: true,
    ...partial,
  }
}

describe('groupPermissionsByMenu', () => {
  it('places menu and action grants under the same sidebar menu', () => {
    const view = groupPermissionsByMenu([
      perm({ id: 1, code: 'menu:overview', name: '菜单：总览', group: '菜单' }),
      perm({ id: 2, code: 'overview:read', name: '查看总览', group: '监控' }),
    ])
    const monitoring = view.navGroups.find((g) => g.navGroup === '监控')
    expect(monitoring?.menus[0]?.menu.key).toBe('overview')
    expect(monitoring?.menus[0]?.permissions.map((p) => p.code)).toEqual([
      'menu:overview',
      'overview:read',
    ])
  })

  it('attaches rbac actions to permissions menu only', () => {
    const view = groupPermissionsByMenu([
      perm({ id: 1, code: 'menu:rbac-users', name: '菜单：用户管理', group: '菜单' }),
      perm({ id: 2, code: 'menu:rbac-permissions', name: '菜单：权限管理', group: '菜单' }),
      perm({ id: 3, code: 'rbac:read', name: '查看权限', group: '权限' }),
    ])
    const rbacMenu = view.navGroups
      .find((g) => g.navGroup === '权限')
      ?.menus.find((m) => m.menu.key === 'rbac-permissions')
    expect(rbacMenu?.permissions.some((p) => p.code === 'rbac:read')).toBe(true)
    const usersMenu = view.navGroups
      .find((g) => g.navGroup === '权限')
      ?.menus.find((m) => m.menu.key === 'rbac-users')
    expect(usersMenu?.permissions.map((p) => p.code)).toEqual(['menu:rbac-users'])
  })

  it('collects unmapped custom permissions', () => {
    const view = groupPermissionsByMenu([
      perm({ id: 9, code: 'custom:foo', name: '自定义能力', group: '自定义', builtin: false }),
    ])
    expect(view.unassigned).toHaveLength(1)
    expect(view.unassigned[0]?.code).toBe('custom:foo')
  })
})
