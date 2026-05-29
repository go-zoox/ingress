export function metricsSourceLabel(source?: string) {
  switch (source) {
    case 'access_log':
      return 'access.log'
    case 'access_log_empty':
      return '空文件'
    case 'access_log_parse_fail':
      return '解析异常'
    case 'unconfigured':
      return '未配置'
    case 'error':
      return '读取失败'
    default:
      return source || '—'
  }
}

export function overviewStreamLabel(stream: {
  connected: boolean
  reconnecting: boolean
  fallbackPolling: boolean
}) {
  if (stream.connected) return 'SSE 已连接'
  if (stream.reconnecting) return 'SSE 重连中'
  if (stream.fallbackPolling) return '轮询刷新'
  return '连接中…'
}
