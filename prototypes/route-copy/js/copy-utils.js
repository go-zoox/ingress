/**
 * 复制路由的纯逻辑（无 DOM 依赖）：
 * - Host 类型推断 / 自动生成副本 Host（含冲突顺延）
 * - Host 冲突检测
 * - 规则对象 → YAML 片段（用于弹窗预览与发布 diff）
 *
 * 浏览器挂到 window.CopyUtils；Node 下可 require 做冒烟测试。
 */
(function (root, factory) {
  var api = factory()
  if (typeof module !== 'undefined' && module.exports) module.exports = api
  root.CopyUtils = api
})(typeof window !== 'undefined' ? window : globalThis, function () {
  var REGEX_META = /[()[\]{}^$|+?\\]/

  function effectiveHostType(host) {
    var h = String(host == null ? '' : host)
    if (REGEX_META.test(h)) return 'regex'
    if (h.indexOf('*') >= 0) return 'wildcard'
    return 'exact'
  }

  function normalizeHost(host) {
    return String(host == null ? '' : host).trim().toLowerCase()
  }

  /** 在 host 的第一个 label 上加 -copy / -copy-N；wildcard 只改 * 之后的部分。 */
  function patchFirstLabel(body, attempt) {
    var labels = body.split('.')
    var idx = 0
    while (idx < labels.length && labels[idx] === '') idx++
    if (idx >= labels.length) return body + '-copy'
    var current = labels[idx]
    var existing = current.match(/-copy(-(\d+))?$/)
    var existingLevel = existing ? (existing[2] ? Number(existing[2]) : 1) : 0
    var level = existingLevel + attempt
    var label = current.replace(/-copy(-\d+)?$/, '')
    if (!label) label = 'copy'
    labels[idx] = label + (level === 0 ? '-copy' : '-copy-' + (level + 1))
    return labels.join('.')
  }

  /**
   * 生成副本 Host。
   * - 正则 Host 不改写（auto=false，交由人工确认，避免破坏正则语义）
   * - 通配符保留 *. 前缀，只改写后面的第一个 label
   * - 自动避开已存在的 Host：admin.example.com → admin-copy.example.com（已占用则 -copy-2）
   *
   * @returns {{host:string, auto:boolean, reason:string, firstSuggestion:string, collisionAvoided:boolean}}
   */
  function suggestCopyHost(host, existingHosts) {
    var raw = String(host == null ? '' : host).trim()
    var existing = (existingHosts || []).map(normalizeHost)
    if (effectiveHostType(raw) === 'regex') {
      return {
        host: raw,
        auto: false,
        reason: 'regex',
        firstSuggestion: raw,
        collisionAvoided: false,
      }
    }

    var prefix = ''
    var body = raw
    if (effectiveHostType(raw) === 'wildcard' && raw.indexOf('*.') === 0) {
      prefix = '*.'
      body = raw.slice(2)
    }

    var first = prefix + patchFirstLabel(body, 0)
    var candidate = first
    var attempt = 0
    while (existing.indexOf(normalizeHost(candidate)) >= 0 && attempt < 20) {
      attempt++
      candidate = prefix + patchFirstLabel(body, attempt)
    }
    return {
      host: candidate,
      auto: true,
      reason: 'auto',
      firstSuggestion: first,
      collisionAvoided: normalizeHost(candidate) !== normalizeHost(first),
    }
  }

  /** 返回第一条 Host 相同的规则（下标 + 规则），没有则 null。 */
  function findHostConflict(host, rules) {
    var norm = normalizeHost(host)
    if (!norm) return null
    for (var i = 0; i < (rules || []).length; i++) {
      if (normalizeHost(rules[i].host) === norm) return { index: i, rule: rules[i] }
    }
    return null
  }

  /** 基础校验：仅原型需要的最小集（真实实现由 ingress validate 兜底）。 */
  function validateHost(host) {
    var h = String(host == null ? '' : host).trim()
    if (!h) return 'Host 不能为空'
    if (/\s/.test(h)) return 'Host 不能包含空白字符'
    if (/^[*]?\.[^.]*$/.test(h)) return 'Host 形如 *.example.com，通配符后需有域名'
    return ''
  }

  function yamlScalar(value) {
    if (typeof value === 'number' || typeof value === 'boolean') return String(value)
    var s = String(value)
    if (s === '') return "''"
    if (/[:#{}\[\],&*?|<>=!%@`^$\\"'’]/.test(s) || /^\s|\s$/.test(s) || /^-/.test(s)) {
      return "'" + s.replace(/'/g, "''") + "'"
    }
    return s
  }

  function toYAML(obj, indent) {
    indent = indent || 0
    var pad = new Array(indent + 1).join('  ')
    var lines = []
    Object.keys(obj).forEach(function (key) {
      var value = obj[key]
      if (value === undefined) return
      if (Array.isArray(value)) {
        if (value.length === 0) {
          lines.push(pad + key + ': []')
          return
        }
        var allScalar = value.every(function (item) {
          return item === null || typeof item !== 'object'
        })
        if (allScalar) {
          lines.push(pad + key + ': [' + value.map(yamlScalar).join(', ') + ']')
          return
        }
        lines.push(pad + key + ':')
        value.forEach(function (item) {
          lines.push(arrayItem(item, indent + 1))
        })
        return
      }
      if (value && typeof value === 'object') {
        lines.push(pad + key + ':')
        lines.push(toYAML(value, indent + 1))
        return
      }
      lines.push(pad + key + ': ' + yamlScalar(value))
    })
    return lines.join('\n')
  }

  function arrayItem(item, indent) {
    var pad = new Array(indent + 1).join('  ')
    if (item === null || typeof item !== 'object') return pad + '- ' + yamlScalar(item)
    var body = toYAML(item, indent + 1)
    var lines = body.split('\n')
    lines[0] = pad + '- ' + lines[0].slice(pad.length + 2)
    return lines.join('\n')
  }

  var RULE_KEYS = ['host', 'host_type', 'enabled', 'backend', 'connect_proxy', 'paths', 'rate_limit', 'security', 'waf']

  /** 把内部规则对象序列化成与 ingress.yaml 同形的片段（去掉原型元字段）。 */
  function ruleYAML(rule) {
    var copy = {}
    RULE_KEYS.forEach(function (key) {
      if (rule[key] !== undefined && rule[key] !== null) copy[key] = rule[key]
    })
    if (copy.host_type === 'auto') delete copy.host_type
    if (copy.enabled === false) copy.enabled = false
    else delete copy.enabled
    if (Array.isArray(copy.paths) && copy.paths.length === 0) delete copy.paths
    return toYAML(copy)
  }

  return {
    effectiveHostType: effectiveHostType,
    normalizeHost: normalizeHost,
    suggestCopyHost: suggestCopyHost,
    findHostConflict: findHostConflict,
    validateHost: validateHost,
    ruleYAML: ruleYAML,
    toYAML: toYAML,
    yamlScalar: yamlScalar,
  }
})
