import type { FeedEvent } from './buildEventsFeed'
import { healthLink, investigateLink, logsLink, wafLink } from './deepLinks'

export type RunbookStep = {
  text: string
  href?: string
  label?: string
}

export function runbookForEvent(e: FeedEvent): RunbookStep[] {
  const host = e.host || ''
  const path = e.path || '/'

  switch (e.kind) {
    case 'waf':
      if (!host) {
        return [{ text: '打开调查页查看 WAF 相关请求' }]
      }
      return [
        {
          text: '打开请求调查，确认命中路由与近期样本',
          href: investigateLink({ host, path }),
          label: '调查',
        },
        {
          text: '在 WAF 页用相同 Host/Path 试匹配，核对规则',
          href: wafLink({ host, path, trial: true }),
          label: 'WAF 试匹配',
        },
        {
          text: '筛选 access 日志中 waf_block=1',
          href: logsLink({ host, path, waf_block: '1', log: 'access' }),
          label: '拦截日志',
        },
      ]
    case 'health':
      if (!host) {
        return [{ text: '查看健康检查页', href: healthLink({ status: 'down' }), label: '健康检查' }]
      }
      return [
        {
          text: '调查该 Host 的路由与健康探测',
          href: investigateLink({ host, path }),
          label: '调查',
        },
        {
          text: '查看全部 DOWN 探测目标',
          href: healthLink({ status: 'down', host }),
          label: '健康检查',
        },
        {
          text: '查该 Host 的 5xx access 日志',
          href: logsLink({ host, log: 'access', status: '5' }),
          label: '错误日志',
        },
      ]
    case 'tls':
      return [
        { text: '在 TLS 页确认证书链与到期日', href: '/tls', label: '证书管理' },
        { text: '检查 ingress.yaml 证书路径', href: '/config', label: '配置' },
        { text: '更新证书后 validate 并 reload', href: '/config', label: '发布' },
      ]
    default:
      return [{ text: '使用「处理」进入对应页面' }]
  }
}
