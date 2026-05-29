import type { OverviewMetrics } from '../api/client'
import {
  getNotificationReadAt,
  isNotificationRead,
  notificationFingerprint,
} from './notificationReadState'

export type AppNotification = {
  id: string
  level: 'info' | 'warn'
  title: string
  detail: string
  href?: string
  fingerprint: string
  read: boolean
  readAt?: string
}

type BuildOptions = {
  runtimeDrift?: boolean
  revisionDrift?: boolean
}

export function buildAppNotifications(
  metrics: OverviewMetrics | null,
  options: BuildOptions = {},
): AppNotification[] {
  if (!metrics) return []

  const raw: Array<Omit<AppNotification, 'read' | 'readAt' | 'fingerprint'>> = []
  const openIssues = metrics.parse_issue_open ?? 0
  if (openIssues > 0) {
    raw.push({
      id: 'parse-issues',
      level: 'warn',
      title: '日志解析异常',
      detail: `${openIssues} 条待处理，可在事件中标记已处理或忽略。`,
      href: '/events',
    })
  }
  if (metrics.window_stale) {
    raw.push({
      id: 'window-stale',
      level: 'info',
      title: '指标时间窗口回退',
      detail: `当前 ${metrics.window} 窗口内无近期流量，总览已展示 tail 内 ${metrics.parseable_in_tail ?? metrics.total} 条可解析日志。`,
      href: '/',
    })
  }
  if (options.runtimeDrift) {
    raw.push({
      id: 'runtime-drift',
      level: 'warn',
      title: '运行配置未对齐',
      detail: '磁盘上的 ingress.yaml 与当前运行中配置 hash 不一致。',
      href: '/config',
    })
  }
  if (options.revisionDrift) {
    raw.push({
      id: 'revision-drift',
      level: 'info',
      title: '配置版本未发布',
      detail: '文件内容与最近发布版本不一致，可在配置中心查看。',
      href: '/config',
    })
  }

  return raw.map((item) => {
    const fingerprint = notificationFingerprint(item.id, item.detail)
    const read = isNotificationRead(item.id, fingerprint)
    return {
      ...item,
      fingerprint,
      read,
      readAt: read ? getNotificationReadAt(item.id, fingerprint) : undefined,
    }
  })
}
