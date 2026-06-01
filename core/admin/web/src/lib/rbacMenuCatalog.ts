import type { RBACPermissionRow } from '../api/client'

/** Mirrors core/admin/service/rbac/menu.go BuiltinMenus + menuGroupOrder. */
export type RbacMenuDef = {
  key: string
  label: string
  navGroup: string
  /** Action permission prefix (e.g. healths menu → health:read). */
  resourcePrefix?: string
  /** When multiple menus share a prefix (rbac), attach actions to this menu key only. */
  actionOwner?: boolean
}

const NAV_GROUP_ORDER = ['监控', '流量', '安全', '维护', '权限', '系统'] as const

export const RBAC_MENU_CATALOG: RbacMenuDef[] = [
  { key: 'overview', label: '总览', navGroup: '监控', actionOwner: true },
  { key: 'events', label: '事件', navGroup: '监控', actionOwner: true },
  { key: 'investigate', label: '调查', navGroup: '监控', actionOwner: true },
  { key: 'logs', label: '日志', navGroup: '监控', actionOwner: true },

  { key: 'routes', label: '路由', navGroup: '流量', actionOwner: true },
  { key: 'services', label: '服务', navGroup: '流量', actionOwner: true },
  { key: 'cache', label: '缓存', navGroup: '流量', actionOwner: true },

  { key: 'waf', label: 'WAF', navGroup: '安全', actionOwner: true },
  { key: 'tls', label: 'TLS', navGroup: '安全', actionOwner: true },
  { key: 'healths', label: '健康检查', navGroup: '安全', resourcePrefix: 'health', actionOwner: true },

  { key: 'maintenance', label: '维护模式', navGroup: '维护', actionOwner: true },
  { key: 'scenarios', label: '场景管理', navGroup: '维护', actionOwner: true },
  { key: 'jobs', label: '定时任务', navGroup: '维护', actionOwner: true },
  { key: 'terminal', label: 'Web 终端', navGroup: '维护', actionOwner: true },

  { key: 'rbac-users', label: '用户管理', navGroup: '权限', resourcePrefix: 'rbac' },
  { key: 'rbac-roles', label: '角色管理', navGroup: '权限', resourcePrefix: 'rbac' },
  { key: 'rbac-permissions', label: '权限管理', navGroup: '权限', resourcePrefix: 'rbac', actionOwner: true },

  { key: 'config', label: '配置', navGroup: '系统', actionOwner: true },
  { key: 'settings', label: '设置', navGroup: '系统', actionOwner: true },
]

const menuByKey = new Map(RBAC_MENU_CATALOG.map((m) => [m.key, m]))

function resourcePrefix(menu: RbacMenuDef): string {
  if (menu.resourcePrefix) return menu.resourcePrefix
  if (menu.key === 'healths') return 'health'
  return menu.key
}

function menuKeyFromCode(code: string): string | null {
  if (!code.startsWith('menu:')) return null
  const key = code.slice('menu:'.length)
  return menuByKey.has(key) ? key : null
}

function actionMenuKeyForPermission(perm: RBACPermissionRow): string | null {
  const idx = perm.code.indexOf(':')
  if (idx <= 0) return null
  const prefix = perm.code.slice(0, idx)
  const owners = RBAC_MENU_CATALOG.filter(
    (m) => resourcePrefix(m) === prefix && m.actionOwner,
  )
  if (owners.length === 1) return owners[0].key
  if (owners.length > 1) {
    return owners[0].key
  }
  const any = RBAC_MENU_CATALOG.find((m) => resourcePrefix(m) === prefix)
  return any?.key ?? null
}

function assignMenuKey(perm: RBACPermissionRow): string | null {
  const fromMenu = menuKeyFromCode(perm.code)
  if (fromMenu) return fromMenu
  return actionMenuKeyForPermission(perm)
}

export type RbacMenuPermissionSection = {
  menu: RbacMenuDef
  permissions: RBACPermissionRow[]
}

export type RbacNavGroupPermissionView = {
  navGroup: string
  menus: RbacMenuPermissionSection[]
}

export type RbacPermissionCatalogView = {
  navGroups: RbacNavGroupPermissionView[]
  unassigned: RBACPermissionRow[]
}

function sortPermissions(rows: RBACPermissionRow[]): RBACPermissionRow[] {
  return [...rows].sort((a, b) => {
    const aMenu = a.code.startsWith('menu:') ? 0 : 1
    const bMenu = b.code.startsWith('menu:') ? 0 : 1
    if (aMenu !== bMenu) return aMenu - bMenu
    return a.name.localeCompare(b.name, 'zh-CN')
  })
}

/** Group permissions under sidebar menus, then under nav sections (监控 / 流量 / …). */
export function groupPermissionsByMenu(permissions: RBACPermissionRow[]): RbacPermissionCatalogView {
  const byMenuKey = new Map<string, RBACPermissionRow[]>()
  const unassigned: RBACPermissionRow[] = []

  for (const perm of permissions) {
    const menuKey = assignMenuKey(perm)
    if (!menuKey) {
      unassigned.push(perm)
      continue
    }
    const list = byMenuKey.get(menuKey) ?? []
    list.push(perm)
    byMenuKey.set(menuKey, list)
  }

  const navGroups: RbacNavGroupPermissionView[] = []
  for (const navGroup of NAV_GROUP_ORDER) {
    const menus: RbacMenuPermissionSection[] = []
    for (const menu of RBAC_MENU_CATALOG) {
      if (menu.navGroup !== navGroup) continue
      const rows = byMenuKey.get(menu.key)
      if (!rows?.length) continue
      menus.push({ menu, permissions: sortPermissions(rows) })
    }
    if (menus.length > 0) {
      navGroups.push({ navGroup, menus })
    }
  }

  const customMenus: RbacMenuPermissionSection[] = []
  for (const [key, rows] of byMenuKey) {
    if (!menuByKey.has(key) && rows.length > 0) {
      customMenus.push({
        menu: { key, label: key, navGroup: '自定义' },
        permissions: sortPermissions(rows),
      })
    }
  }

  if (customMenus.length > 0) {
    navGroups.push({ navGroup: '自定义', menus: customMenus })
  }

  unassigned.sort((a, b) => a.name.localeCompare(b.name, 'zh-CN'))

  return { navGroups, unassigned }
}
