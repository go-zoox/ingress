import type { LucideIcon } from 'lucide-react'
import {
  Activity,
  ArrowLeftRight,
  Clock,
  Construction,
  FileCode2,
  HardDrive,
  HeartPulse,
  KeyRound,
  LayoutDashboard,
  Lock,
  ScrollText,
  Search,
  Server,
  Settings,
  Shield,
  Terminal,
  UserCog,
  Users,
} from 'lucide-react'

export type NavBadgeKey = 'events' | 'healths' | 'tls' | 'waf'

const navIconMap: Record<string, LucideIcon> = {
  'layout-dashboard': LayoutDashboard,
  activity: Activity,
  search: Search,
  'scroll-text': ScrollText,
  'arrow-left-right': ArrowLeftRight,
  server: Server,
  'hard-drive': HardDrive,
  shield: Shield,
  lock: Lock,
  'heart-pulse': HeartPulse,
  construction: Construction,
  clock: Clock,
  terminal: Terminal,
  users: Users,
  'user-cog': UserCog,
  'key-round': KeyRound,
  'file-code-2': FileCode2,
  settings: Settings,
}

export function navIcon(name: string): LucideIcon {
  return navIconMap[name] ?? LayoutDashboard
}

export function isNavBadgeKey(value: string | undefined): value is NavBadgeKey {
  return value === 'events' || value === 'healths' || value === 'tls' || value === 'waf'
}
