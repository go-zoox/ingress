export type PageAbout = {
  title: string
  desc: string
}

const PAGE_ABOUT: Record<string, PageAbout> = {
  '/': {
    title: '总览',
    desc: 'SRE 监控面板：流量、质量、基础设施与异常',
  },
  '/events': {
    title: '事件',
    desc: '待处理、已处理与已忽略；WAF 拦截与日志解析可弹窗处置，支持批量与全部标记',
  },
  '/investigate': {
    title: '请求调查',
    desc: '聚合路由裁决、访问样本、策略与健康状态，用于排查慢请求、5xx 与 WAF 拦截',
  },
  '/logs': {
    title: '日志',
    desc: '实时 tail 访问日志，支持筛选与调查跳转；总览指标也依赖访问日志。',
  },
  '/routes': {
    title: '路由',
    desc: '路由规则增删改查、拓扑与试匹配',
  },
  '/services': {
    title: '服务',
    desc: '可复用的 upstream Service 目录，供路由 backend 选用',
  },
  '/cache': {
    title: '缓存',
    desc: '全局 cache 后端、路由级 HTTP 响应缓存策略与 access log 命中统计',
  },
  '/waf': {
    title: 'WAF',
    desc: '全局规则、运行时开关、攻击地图可视化与 block/audit 事件',
  },
  '/tls': {
    title: 'TLS / 证书',
    desc: 'HTTPS 监听、证书文件路径与有效期；总览与侧栏角标会提示即将过期域名',
  },
  '/healths': {
    title: '健康检查',
    desc: '探测路由 backend.service.healthcheck 配置的后端可用性',
  },
  '/maintenance': {
    title: '维护',
    desc: '全局 maintenance.hosts 登记与默认 503；规则级 scope 在路由编辑器配置',
  },
  '/terminal': {
    title: 'Web 终端',
    desc: '通过 Xterm 连接 Admin 主机 Shell；断线 60 秒内自动重连并恢复同一会话',
  },
  '/config': {
    title: '配置',
    desc: '分模块编辑 ingress.yaml → 保存与发布（查看变更 → 仅保存或热加载）',
  },
  '/settings': {
    title: '设置',
    desc: 'Admin 服务配置、Ingress 集成路径、数据存储与界面偏好',
  },
  '/messages': {
    title: '消息通知',
    desc: '系统提示与运维告警摘要；已读状态保存在浏览器本地',
  },
}

export function pageAboutForPath(pathname: string): PageAbout | null {
  if (PAGE_ABOUT[pathname]) {
    return PAGE_ABOUT[pathname]
  }
  if (pathname.startsWith('/routes/')) {
    return { title: '路由详情', desc: '单条路由的后端、策略与 WAF 配置摘要' }
  }
  if (pathname.startsWith('/services/')) {
    return { title: '服务详情', desc: '上游 Service 配置、引用路由与访问指标' }
  }
  return null
}
