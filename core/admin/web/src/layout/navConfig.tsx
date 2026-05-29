import type { LucideIcon } from 'lucide-react'
import {
  LayoutDashboard,
  ScrollText,
  Activity,
  Search,
  ArrowLeftRight,
  HardDrive,
  Shield,
  Lock,
  HeartPulse,
  FileCode2,
  Settings,
} from 'lucide-react'

export type NavBadgeKey = 'events' | 'healths' | 'tls' | 'waf'

export type NavItem = {
  to: string
  label: string
  icon: LucideIcon
  end?: boolean
  badgeKey?: NavBadgeKey
}

export type NavGroup = {
  label: string
  items: NavItem[]
}

export const navGroups: NavGroup[] = [
  {
    label: '监控',
    items: [
      { to: '/', label: '总览', icon: LayoutDashboard, end: true },
      { to: '/events', label: '事件', icon: Activity, badgeKey: 'events' },
      { to: '/investigate', label: '调查', icon: Search },
      { to: '/logs', label: '日志', icon: ScrollText },
    ],
  },
  {
    label: '流量',
    items: [
      { to: '/routes', label: '路由', icon: ArrowLeftRight },
      { to: '/cache', label: '缓存', icon: HardDrive },
    ],
  },
  {
    label: '安全',
    items: [
      { to: '/waf', label: 'WAF', icon: Shield },
      { to: '/tls', label: 'TLS', icon: Lock, badgeKey: 'tls' },
      { to: '/healths', label: '健康检查', icon: HeartPulse, badgeKey: 'healths' },
    ],
  },
  {
    label: '系统',
    items: [
      { to: '/config', label: '配置', icon: FileCode2 },
      { to: '/settings', label: '设置', icon: Settings },
    ],
  },
]
