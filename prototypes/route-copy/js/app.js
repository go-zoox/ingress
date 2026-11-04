(function () {
  'use strict'

  var U = window.CopyUtils
  var MOCK = window.MOCK

  var $ = function (sel, root) { return (root || document).querySelector(sel) }
  var $$ = function (sel, root) { return Array.prototype.slice.call((root || document).querySelectorAll(sel)) }

  function cloneDeep(value) {
    return JSON.parse(JSON.stringify(value))
  }

  function esc(value) {
    return String(value == null ? '' : value).replace(/[&<>"']/g, function (ch) {
      return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[ch]
    })
  }

  var state = {
    rules: cloneDeep(MOCK.rules),
    dirty: false,
    view: 'routes',
    detailRuleId: null,
    detailTab: 'logs',
    draft: null,
    highlightId: null,
    toastTimer: null,
    menuEl: null,
    menuCleanup: null,
    copySeq: 0,
  }

  var drawers = []

  // ---------------------------------------------------------------- helpers

  function indexOfRule(id) {
    for (var i = 0; i < state.rules.length; i++) {
      if (state.rules[i].id === id) return i
    }
    return -1
  }

  function findRule(id) {
    var index = indexOfRule(id)
    return index < 0 ? null : state.rules[index]
  }

  function backendSummary(backend) {
    if (!backend) return '—'
    var type = backend.type || 'service'
    if (type === 'redirect') {
      var redirect = backend.redirect || {}
      return 'redirect → ' + (redirect.url || '?') + (redirect.status === 'permanent' ? ' · 301' : ' · 302')
    }
    if (type === 'handler') {
      var handler = backend.handler || {}
      var handlerText = 'handler → ' + (handler.type || 'static_response')
      if (handler.root_dir) handlerText += ' (' + handler.root_dir + ')'
      return handlerText
    }
    var service = backend.service || {}
    var text = 'service → ' + (service.name || '?') + (service.port ? ':' + service.port : '')
    if (service.mode) text += ' (' + service.mode + ')'
    return text
  }

  function ruleSummary(rule) {
    var out = backendSummary(rule.backend)
    var proxy = rule.connect_proxy
    if (proxy && proxy.enabled) {
      var upstream = proxy.upstream || {}
      out = 'PROXY → ' + (upstream.name || 'upstream') + (upstream.port ? ':' + upstream.port : '') + ' · ' + out
    }
    var cache = rule.backend && rule.backend.cache
    if (cache && cache.enabled) out += ' · cache ' + (cache.ttl || 300) + 's'
    if (rule.rate_limit && rule.rate_limit.requests) {
      out += ' · 限流 ' + rule.rate_limit.requests + '/' + (rule.rate_limit.period || 60) + 's'
    }
    var auth = rule.backend && rule.backend.service && rule.backend.service.auth
    if (auth && auth.type) out += ' · auth:' + auth.type
    return out
  }

  function ipBadge(rule) {
    var allow = (rule.waf && rule.waf.allow) || []
    if (!allow.length) return '<span class="badge badge-audit">未限制</span>'
    return '<span class="badge badge-wildcard">' + allow.length + ' 条</span>'
  }

  function enabledBadge(rule) {
    var enabled = rule.enabled !== false
    return '<span class="badge ' + (enabled ? 'badge-exact' : 'badge-block') + '">' + (enabled ? '启用' : '禁用') + '</span>'
  }

  function copyChip(rule) {
    if (!rule.isNew || rule.published) return ''
    if (rule.savedOnly) return '<span class="chip chip-new">已保存 · 未发布</span>'
    return '<span class="chip chip-new">副本 · 未发布</span>'
  }

  // ---------------------------------------------------------------- rules table

  function renderRules() {
    var tbody = $('#rules-tbody')
    tbody.innerHTML = state.rules.map(function (rule, index) {
      var enabled = rule.enabled !== false
      var pathCount = (rule.paths || []).length
      return [
        '<tr data-id="' + esc(rule.id) + '" class="' + (enabled ? '' : 'row-muted') + '">',
        '  <td class="col-drag"><span class="row-drag-handle" title="拖拽排序（原型未实现拖拽）">⋮⋮</span></td>',
        '  <td class="col-idx">' + (index + 1) + '</td>',
        '  <td>' + enabledBadge(rule) + '</td>',
        '  <td>',
        '    <button type="button" class="host-link" data-detail="' + esc(rule.id) + '" title="查看路由详情"><code>' + esc(rule.host) + '</code></button>',
        copyChip(rule),
        '  </td>',
        '  <td>' + esc(rule.host_type || 'auto') + '</td>',
        '  <td class="backend-cell" title="' + esc(ruleSummary(rule)) + '">' + esc(ruleSummary(rule)) + '</td>',
        '  <td>' + ipBadge(rule) + '</td>',
        '  <td>',
        '    <button type="button" class="action-link" data-paths="' + esc(rule.id) + '">' + (pathCount > 0 ? pathCount + ' 条' : '配置') + '</button>',
        '  </td>',
        '  <td>',
        '    <div class="row-actions">',
        '      <button type="button" class="action-link" data-edit="' + esc(rule.id) + '">编辑</button>',
        '      <button type="button" class="action-link action-advanced" data-menu="' + esc(rule.id) + '" aria-expanded="false">高级</button>',
        '    </div>',
        '  </td>',
        '</tr>',
      ].join('\n')
    }).join('')

    if (state.highlightId) {
      var row = $('tr[data-id="' + state.highlightId + '"]')
      if (row) {
        row.classList.add('row-flash')
        if (row.scrollIntoView) row.scrollIntoView({ block: 'center', behavior: 'smooth' })
        var id = state.highlightId
        window.setTimeout(function () {
          if (state.highlightId === id) state.highlightId = null
          row.classList.remove('row-flash')
        }, 3200)
      }
    }
    updateToolbar()
  }

  function updateToolbar() {
    var badge = $('#dirty-badge')
    badge.hidden = !state.dirty
    $('#btn-save').disabled = !state.dirty
  }

  // ---------------------------------------------------------------- advanced menu

  function closeRowMenu() {
    if (state.menuCleanup) {
      state.menuCleanup()
      state.menuCleanup = null
    }
    if (!state.menuEl) return
    state.menuEl.remove()
    state.menuEl = null
    $$('[data-menu][aria-expanded="true"]').forEach(function (btn) {
      btn.setAttribute('aria-expanded', 'false')
    })
  }

  function openRowMenu(trigger, ruleId) {
    var wasOpen = trigger.getAttribute('aria-expanded') === 'true'
    closeRowMenu()
    if (wasOpen) return

    var index = indexOfRule(ruleId)
    if (index < 0) return
    trigger.setAttribute('aria-expanded', 'true')

    var items = [
      { label: '复制路由', onSelect: function () { openCopyDrawer(ruleId) } },
      { label: 'Paths', onSelect: function () { toast('Paths 编辑沿用现有弹窗（原型聚焦复制流程）') } },
      { label: '详情', onSelect: function () { showDetail(ruleId) } },
      { separator: true },
      { label: '上移', disabled: index === 0, onSelect: function () { moveRule(ruleId, -1) } },
      { label: '下移', disabled: index === state.rules.length - 1, onSelect: function () { moveRule(ruleId, 1) } },
      { separator: true },
      { label: '删除', danger: true, onSelect: function () { removeRule(ruleId) } },
    ]

    var panel = document.createElement('div')
    panel.className = 'action-menu-panel'
    panel.setAttribute('role', 'menu')

    items.forEach(function (item) {
      if (item.separator) {
        var separator = document.createElement('div')
        separator.className = 'action-menu-sep'
        separator.setAttribute('aria-hidden', 'true')
        panel.appendChild(separator)
        return
      }
      var button = document.createElement('button')
      button.type = 'button'
      button.setAttribute('role', 'menuitem')
      button.className = 'action-menu-item' + (item.danger ? ' action-danger' : '')
      button.textContent = item.label
      button.disabled = Boolean(item.disabled)
      button.addEventListener('click', function () {
        if (button.disabled) return
        closeRowMenu()
        item.onSelect()
      })
      panel.appendChild(button)
    })

    var rect = trigger.getBoundingClientRect()
    var width = 164
    panel.style.minWidth = width + 'px'
    panel.style.left = Math.max(12, rect.right - width) + 'px'
    panel.style.top = rect.bottom + 4 + 'px'

    $('#row-menu-root').appendChild(panel)
    state.menuEl = panel

    var panelHeight = panel.offsetHeight
    if (rect.bottom + 4 + panelHeight > window.innerHeight - 12) {
      panel.style.top = Math.max(12, rect.top - panelHeight - 4) + 'px'
    }

    var onDocumentMouseDown = function (event) {
      var target = event.target
      if (panel.contains(target)) return
      if (target.closest && target.closest('[data-menu]')) return
      closeRowMenu()
    }
    var onScrollOrResize = function () { closeRowMenu() }
    document.addEventListener('mousedown', onDocumentMouseDown, true)
    window.addEventListener('scroll', onScrollOrResize, true)
    window.addEventListener('resize', onScrollOrResize)
    state.menuCleanup = function () {
      document.removeEventListener('mousedown', onDocumentMouseDown, true)
      window.removeEventListener('scroll', onScrollOrResize, true)
      window.removeEventListener('resize', onScrollOrResize)
    }
  }

  function moveRule(ruleId, delta) {
    var index = indexOfRule(ruleId)
    var target = index + delta
    if (index < 0 || target < 0 || target >= state.rules.length) return
    var rule = state.rules[index]
    state.rules.splice(index, 1)
    state.rules.splice(target, 0, rule)
    state.dirty = true
    renderRules()
    toast('已调整顺序：' + rule.host + ' 现为第 ' + (target + 1) + ' 条')
  }

  function removeRule(ruleId) {
    var rule = findRule(ruleId)
    if (!rule) return
    if (!window.confirm('删除路由规则 ' + rule.host + '？')) return
    state.rules = state.rules.filter(function (item) { return item.id !== ruleId })
    state.dirty = true
    renderRules()
    toast('已删除 ' + rule.host + '（未保存）')
  }

  // ---------------------------------------------------------------- toast

  function toast(message, options) {
    options = options || {}
    var root = $('#toast-root')
    root.innerHTML = ''
    var el = document.createElement('div')
    el.className = 'toast' + (options.tone === 'error' ? ' is-error' : '')
    el.innerHTML = '<span>' + esc(message) + '</span>'

    if (options.actionLabel) {
      var action = document.createElement('button')
      action.type = 'button'
      action.className = 'toast-action'
      action.textContent = options.actionLabel
      action.addEventListener('click', function () {
        hideToast()
        options.onAction()
      })
      el.appendChild(action)
    }

    root.appendChild(el)
    window.clearTimeout(state.toastTimer)
    state.toastTimer = window.setTimeout(hideToast, options.duration || 6000)
  }

  function hideToast() {
    window.clearTimeout(state.toastTimer)
    $('#toast-root').innerHTML = ''
  }

  // ---------------------------------------------------------------- views

  function showView(view) {
    state.view = view
    $('#view-routes').hidden = view !== 'routes'
    $('#view-detail').hidden = view !== 'detail'
    window.scrollTo(0, 0)
  }

  // ---------------------------------------------------------------- copy drawer

  function copySummaryChips(rule) {
    var chips = []
    chips.push(backendSummary(rule.backend))
    var pathCount = (rule.paths || []).length
    chips.push(pathCount + ' 条 Path')
    var allow = (rule.waf && rule.waf.allow) || []
    chips.push(allow.length ? 'IP 白名单 ' + allow.length + ' 条' : 'IP 白名单 未限制')
    if (rule.connect_proxy && rule.connect_proxy.enabled) chips.push('正向代理 已启用')
    if (rule.rate_limit && rule.rate_limit.requests) chips.push('限流 ' + rule.rate_limit.requests + '/' + (rule.rate_limit.period || 60) + 's')
    var cache = rule.backend && rule.backend.cache
    if (cache && cache.enabled) chips.push('响应缓存 ' + (cache.ttl || 300) + 's')
    if (rule.security && rule.security.profile) chips.push('安全 ' + rule.security.profile)
    var auth = rule.backend && rule.backend.service && rule.backend.service.auth
    if (auth && auth.type) chips.push('认证 ' + auth.type)
    return chips
  }

  function copySections() {
    var draft = state.draft
    var source = draft.source
    var pathCount = (source.paths || []).length
    var cache = source.backend && source.backend.cache
    return [
      { id: 'basic', label: '基础信息' },
      { id: 'backend', label: 'Backend' },
      { id: 'ip_allowlist', label: 'IP 白名单', badge: ((source.waf && source.waf.allow) || []).length ? ((source.waf.allow || []).length + ' 条') : '' },
      { id: 'connect_proxy', label: '正向代理', badge: source.connect_proxy && source.connect_proxy.enabled ? 'PROXY' : '' },
      { id: 'paths', label: 'Path 规则', badge: pathCount || '' },
      { id: 'auth', label: '认证授权', badge: (source.backend && source.backend.service && source.backend.service.auth && source.backend.service.auth.type) || '' },
      { id: 'upstream', label: '上游转发' },
      { id: 'cache', label: '响应缓存', badge: cache && cache.enabled ? '开' : '' },
      { id: 'security', label: '安全', badge: (source.security && source.security.profile) || '' },
      { id: 'rate_limit', label: '路由限流', badge: source.rate_limit && source.rate_limit.requests ? source.rate_limit.requests + '/' + (source.rate_limit.period || 60) + 's' : '' },
    ]
  }

  function typeHint(hostType, host) {
    if (hostType === 'auto') {
      return 'auto：编译时按 Host 推断为 ' + U.effectiveHostType(host) + '（含 * → wildcard，含正则元字符 → regex，否则 exact）'
    }
    return '已显式指定为 ' + hostType + '，不会再自动推断'
  }

  function draftRule() {
    var draft = state.draft
    var rule = cloneDeep(draft.source)
    rule.host = draft.host.trim()
    if (draft.hostType === 'auto') delete rule.host_type
    else rule.host_type = draft.hostType
    rule.enabled = draft.enabled
    return rule
  }

  function basicPanelHTML() {
    var draft = state.draft
    var source = draft.source
    var sourceIndex = indexOfRule(source.id) + 1
    var suggestion = draft.suggestion
    var note = ''
    if (!suggestion.auto) {
      note = '<p class="field-warn" id="copy-host-note">正则 Host 不做自动改名：请手动确认副本要匹配的域名，避免与源规则完全重复。</p>'
    } else if (suggestion.collisionAvoided) {
      note = '<p class="field-hint" id="copy-host-note">' + esc(suggestion.firstSuggestion) + ' 已被现有规则占用，已自动顺延为 <code>' + esc(suggestion.host) + '</code></p>'
    } else {
      note = '<p class="field-hint" id="copy-host-note">已按原 Host 自动生成区分名，可直接修改</p>'
    }

    var enabledHint = source.enabled !== false
      ? '源规则为「启用」，副本默认一致；仍需「保存与发布」后才生效。'
      : '源规则为「禁用」，副本默认一致；启用后仍需「保存与发布」才生效。'

    return [
      '<div class="form-grid form-grid--2">',
      '  <div class="form-field">',
      '    <label for="copy-host">Host <span class="req">*</span></label>',
      '    <input type="text" id="copy-host" class="mono" autocomplete="off" spellcheck="false" value="' + esc(draft.host) + '" />',
      '    <div id="copy-host-note-wrapper">' + note + '</div>',
      '    <p class="field-error" id="copy-host-error" hidden></p>',
      '    <p class="field-warn" id="copy-host-warn" hidden></p>',
      '  </div>',
      '  <div class="form-field">',
      '    <label for="copy-host-type">Host 类型</label>',
      '    <select id="copy-host-type">',
      ['auto', 'exact', 'wildcard', 'regex'].map(function (type) {
        return '<option value="' + type + '"' + (draft.hostType === type ? ' selected' : '') + '>' + type + '</option>'
      }).join(''),
      '    </select>',
      '    <p class="field-hint" id="copy-type-hint">' + esc(typeHint(draft.hostType, draft.host)) + '</p>',
      '  </div>',
      '  <div class="form-field">',
      '    <span>状态</span>',
      '    <label class="switch-row">',
      '      <input type="checkbox" id="copy-enabled"' + (draft.enabled ? ' checked' : '') + ' />',
      '      <span class="switch-track"></span>',
      '      <span id="copy-enabled-label">' + (draft.enabled ? '启用' : '禁用') + '</span>',
      '    </label>',
      '    <p class="field-hint">' + esc(enabledHint) + '</p>',
      '  </div>',
      '  <div class="form-field">',
      '    <label for="copy-position">插入位置</label>',
      '    <select id="copy-position">',
      '      <option value="after">紧随规则 #' + sourceIndex + ' 之后（推荐）</option>',
      '      <option value="end">列表末尾</option>',
      '    </select>',
      '    <p class="field-hint">列表顺序即匹配优先级；紧随源规则便于对照，也便于控制先后。</p>',
      '  </div>',
      '</div>',
      '<div class="copy-section-title">将复制的配置（除 Host 外全部沿用）</div>',
      '<div class="copy-chips">',
      copySummaryChips(source).map(function (chip) { return '<span class="chip">' + esc(chip) + '</span>' }).join(''),
      '</div>',
      '<div class="copy-block-note">副本与源规则互相独立：深拷贝后各自维护，修改副本不会影响源规则，反之亦然。</div>',
      '<details class="yaml-preview" open>',
      '  <summary>预览新增的 YAML 片段</summary>',
      '  <pre class="yaml" id="copy-yaml">' + esc(U.ruleYAML(draftRule())) + '</pre>',
      '</details>',
    ].join('\n')
  }

  function readOnlyPanel(section) {
    var source = state.draft.source
    var rows = []
    var note = '副本沿用源规则配置；如需调整，创建后在列表中点「编辑」。'

    if (section === 'backend') {
      rows = [
        ['类型', source.backend && source.backend.type ? source.backend.type : 'service'],
        ['上游', backendSummary(source.backend)],
      ]
      var service = (source.backend && source.backend.service) || {}
      if (service.protocol) rows.push(['协议', service.protocol])
      if (service.mode) rows.push(['Host 模式', service.mode])
      var cache = source.backend && source.backend.cache
      if (cache && cache.enabled) rows.push(['响应缓存', (cache.ttl || 300) + 's'])
      note = '复制保留 backend 全部字段（service / redirect / handler、cache、auth 等）。'
    } else if (section === 'ip_allowlist') {
      var allow = (source.waf && source.waf.allow) || []
      var deny = (source.waf && source.waf.deny) || []
      rows = [
        ['放行名单', allow.length ? allow.join('、') : '未配置'],
        ['拒绝名单', deny.length ? deny.join('、') : '未配置'],
      ]
      note = '名单规则随副本一起复制（rules[].waf），由 WAF IP 阶段执行。'
    } else if (section === 'connect_proxy') {
      var proxy = source.connect_proxy
      rows = proxy && proxy.enabled
        ? [
            ['状态', '已启用'],
            ['上游', (proxy.upstream && proxy.upstream.name ? proxy.upstream.name + ':' + (proxy.upstream.port || 3128) : '—')],
            ['允许目标', (proxy.allowed_targets || []).join('、') || '未限制'],
          ]
        : [['状态', '未启用']]
      note = 'CONNECT / absolute-form 请求的独立上游与白名单会原样复制。'
    } else if (section === 'paths') {
      var paths = source.paths || []
      var rowsHtml = paths.length
        ? '<table class="mini-paths"><thead><tr><th>Path</th><th>Path 级 backend</th></tr></thead><tbody>' +
          paths.map(function (item) {
            return '<tr><td><code>' + esc(item.path) + '</code></td><td>' + esc(item.strip_prefix ? '继承 Host 级 backend · strip_prefix' : '继承 Host 级 backend') + '</td></tr>'
          }).join('') +
          '</tbody></table>'
        : '<div class="copy-block-note">源规则没有 Path 级规则（使用 Host 级 backend）。</div>'
      return '<div class="entity-form-panel-head"><h3>Path 规则</h3><p>按 path 前缀覆盖 Host 级 backend；' + paths.length + ' 条规则随副本一起复制。</p></div>' +
        '<div class="entity-form-panel-body">' + rowsHtml + '<div class="copy-block-note">' + esc(note) + '</div></div>'
    } else if (section === 'auth') {
      var auth = (source.backend && source.backend.service && source.backend.service.auth) || {}
      rows = [['认证方式', auth.type || '未配置'], ['启用状态', auth.enabled === false ? '已禁用' : '启用']]
      note = 'service.auth 随 backend 一起复制；如需新凭据，请创建后单独修改。'
    } else if (section === 'upstream') {
      rows = [['Host 改写', '跟随全局 / backend.mode'], ['路径改写', '未配置'], ['请求头改写', '未配置']]
      note = 'request / response 改写配置随 backend 复制。'
    } else if (section === 'cache') {
      var cacheCfg = (source.backend && source.backend.cache) || {}
      rows = [
        ['状态', cacheCfg.enabled ? '启用' : '关闭'],
        ['TTL', cacheCfg.enabled ? (cacheCfg.ttl || 300) + 's' : '—'],
        ['缓存方法', cacheCfg.enabled ? (cacheCfg.methods || ['GET', 'HEAD']).join(', ') : '—'],
      ]
      note = 'backend.cache 原样复制；副本与原规则的缓存键互不影响。'
    } else if (section === 'security') {
      var security = source.security || {}
      rows = [['Profile', security.profile || '继承全局'], ['CORS', security.cors ? '已配置' : '未配置']]
      note = 'Host 级 security 覆盖随副本复制；未配置时继承全局。'
    } else if (section === 'rate_limit') {
      var limit = source.rate_limit || {}
      rows = limit.requests
        ? [
            ['请求数', String(limit.requests)],
            ['周期', (limit.period || 60) + 's'],
            ['计数键', limit.key || 'global'],
          ]
        : [['状态', '未配置']]
      note = 'rules[].rate_limit 随副本复制；与原规则各自独立计数。'
    }

    return '<div class="entity-form-panel-head"><h3>' + esc(sectionTitle(section)) + '</h3><p>' + esc(sectionDesc(section)) + '</p></div>' +
      '<div class="entity-form-panel-body">' +
      '<dl class="kv-list">' + rows.map(function (row) {
        return '<dt>' + esc(row[0]) + '</dt><dd>' + esc(row[1]) + '</dd>'
      }).join('') + '</dl>' +
      '<div class="copy-block-note">' + esc(note) + '</div>' +
      '</div>'
  }

  function sectionTitle(id) {
    var titles = {
      backend: 'Backend',
      ip_allowlist: 'IP 白名单',
      connect_proxy: '正向代理',
      paths: 'Path 规则',
      auth: '认证授权',
      upstream: '上游转发',
      cache: '响应缓存',
      security: '安全',
      rate_limit: '路由限流',
    }
    return titles[id] || id
  }

  function sectionDesc(id) {
    var descs = {
      backend: 'Host 级默认 upstream / handler / redirect（副本原样保留）。',
      ip_allowlist: '仅名单内来源 IP 可访问（rules[].waf.allow，WAF IP 阶段执行）。',
      connect_proxy: 'CONNECT / absolute-form 请求的独立上游与目标白名单。',
      paths: '按 path 前缀覆盖 Host 级 backend。',
      auth: '上游 service 访问认证。',
      upstream: 'request / response 改写。',
      cache: 'HTTP 响应缓存 backend.cache。',
      security: '覆盖全局 security: 的响应头与 CORS。',
      rate_limit: '按 Host 规则限流 rules[].rate_limit。',
    }
    return descs[id] || ''
  }

  function renderCopyDrawer() {
    var draft = state.draft
    var source = draft.source
    var sourceIndex = indexOfRule(source.id) + 1
    var sections = copySections()
    var active = draft.activeSection || 'basic'

    var panelsHtml = sections.map(function (section) {
      var body = section.id === 'basic' ? basicPanelHTML() : readOnlyPanel(section.id)
      return '<div class="copy-panel" data-panel="' + section.id + '"' + (section.id === active ? '' : ' hidden') + '>' +
        (section.id === 'basic'
          ? '<div class="entity-form-panel-head"><h3>基础信息</h3><p>复制只改 Host；其余配置原样带走，创建后可继续在列表中编辑。</p></div><div class="entity-form-panel-body">' + body + '</div>'
          : body) +
        '</div>'
    }).join('')

    var overlay = document.createElement('div')
    overlay.className = 'drawer-overlay'
    overlay.setAttribute('data-overlay', 'copy')
    overlay.innerHTML = [
      '<aside class="drawer" role="dialog" aria-modal="true" aria-labelledby="copy-title">',
      '  <header class="drawer-head">',
      '    <h2 id="copy-title">复制路由 <span class="chip chip-new">副本</span><span class="head-sub">源规则 #' + sourceIndex + ' · ' + esc(source.host) + '</span></h2>',
      '    <button type="button" class="drawer-close" data-close aria-label="关闭">×</button>',
      '  </header>',
      '  <div class="drawer-body">',
      '    <div class="source-strip">',
      '      <div class="item"><div class="label">源 Host</div><div class="value"><code>' + esc(source.host) + '</code></div></div>',
      '      <div class="item"><div class="label">状态</div><div class="value">' + (source.enabled !== false ? '启用' : '禁用') + '</div></div>',
      '      <div class="item"><div class="label">Paths</div><div class="value">' + ((source.paths || []).length || 0) + ' 条</div></div>',
      '      <div class="item"><div class="label">Backend</div><div class="value" title="' + esc(ruleSummary(source)) + '">' + esc(ruleSummary(source)) + '</div></div>',
      '    </div>',
      '    <div class="copy-notice"><span class="dot"></span><span>复制会深拷贝整条规则（backend / paths / WAF / 限流 / 缓存 / 安全 / 正向代理…），通常只需要确认新 Host；未保存或未发布前不会影响线上流量。</span></div>',
      '    <div class="entity-form-layout">',
      '      <aside class="entity-form-nav" aria-label="表单分区">',
      sections.map(function (section) {
        return '<button type="button" class="entity-form-tab' + (section.id === active ? ' active' : '') + '" data-section="' + section.id + '">' +
          '<span>' + esc(section.label) + '</span>' +
          (section.badge ? '<em class="entity-form-badge">' + esc(section.badge) + '</em>' : '') +
          '</button>'
      }).join(''),
      '      </aside>',
      '      <div class="entity-form-panel">' + panelsHtml + '</div>',
      '    </div>',
      '  </div>',
      '  <footer class="drawer-foot">',
      '    <span class="foot-note">创建后仅改草稿，需「保存与发布」才生效</span>',
      '    <button type="button" class="btn" data-close>取消</button>',
      '    <button type="button" class="btn btn-primary" id="copy-submit">创建副本</button>',
      '  </footer>',
      '</aside>',
    ].join('\n')

    $('#drawer-root').appendChild(overlay)
    drawers.push(overlay)
    document.body.style.overflow = 'hidden'

    overlay.addEventListener('click', function (event) {
      if (event.target === overlay || event.target.closest('[data-close]')) closeTopDrawer()
    })

    $$('.entity-form-tab', overlay).forEach(function (tab) {
      tab.addEventListener('click', function () {
        var section = tab.getAttribute('data-section')
        state.draft.activeSection = section
        $$('.entity-form-tab', overlay).forEach(function (item) {
          item.classList.toggle('active', item === tab)
        })
        $$('.copy-panel', overlay).forEach(function (panel) {
          panel.hidden = panel.getAttribute('data-panel') !== section
        })
      })
    })

    var hostInput = $('#copy-host', overlay)
    var typeSelect = $('#copy-host-type', overlay)
    var enabledInput = $('#copy-enabled', overlay)
    var enabledLabel = $('#copy-enabled-label', overlay)
    var positionSelect = $('#copy-position', overlay)
    var submit = $('#copy-submit', overlay)

    hostInput.addEventListener('input', function () {
      state.draft.host = hostInput.value
      var noteWrapper = $('#copy-host-note-wrapper', overlay)
      if (noteWrapper) {
        noteWrapper.hidden = hostInput.value.trim() !== state.draft.suggestion.host
      }
      refreshCopyValidation()
    })
    typeSelect.addEventListener('change', function () {
      state.draft.hostType = typeSelect.value
      $('#copy-type-hint', overlay).textContent = typeHint(state.draft.hostType, state.draft.host)
      refreshCopyValidation()
    })
    enabledInput.addEventListener('change', function () {
      state.draft.enabled = enabledInput.checked
      enabledLabel.textContent = enabledInput.checked ? '启用' : '禁用'
    })
    positionSelect.addEventListener('change', function () {
      state.draft.position = positionSelect.value
    })
    submit.addEventListener('click', function () {
      if (submit.disabled) return
      createCopy()
    })

    window.setTimeout(function () {
      hostInput.focus()
      hostInput.select()
    }, 60)

    refreshCopyValidation()
  }

  function refreshCopyValidation() {
    var overlay = drawers[drawers.length - 1]
    if (!overlay) return
    var draft = state.draft
    var host = (draft.host || '').trim()
    var submit = $('#copy-submit', overlay)
    var input = $('#copy-host', overlay)
    var errorEl = $('#copy-host-error', overlay)
    var warnEl = $('#copy-host-warn', overlay)
    var yamlEl = $('#copy-yaml', overlay)

    var error = U.validateHost(host)
    var warn = ''
    if (!error) {
      var conflict = U.findHostConflict(host, state.rules)
      if (conflict) {
        error = 'Host 与规则 #' + (conflict.index + 1) + '（' + conflict.rule.host + '）重复；列表顺序在前的先匹配，副本通常不会命中。'
      } else if (U.effectiveHostType(host) === 'regex') {
        warn = '正则 Host 会按正则匹配，请确认副本的匹配范围与源规则不同。'
      }
    }

    errorEl.textContent = error || ''
    errorEl.hidden = !error
    warnEl.textContent = warn || ''
    warnEl.hidden = !warn
    input.classList.toggle('has-error', Boolean(error))
    submit.disabled = Boolean(error)

    var typeHintEl = $('#copy-type-hint', overlay)
    if (typeHintEl) typeHintEl.textContent = typeHint(draft.hostType, draft.host)

    if (yamlEl) {
      try {
        yamlEl.textContent = U.ruleYAML(draftRule())
      } catch (e) {
        yamlEl.textContent = '# YAML 预览失败：' + e.message
      }
    }
  }

  function closeTopDrawer() {
    var overlay = drawers.pop()
    if (!overlay) return
    overlay.remove()
    if (drawers.length === 0) {
      document.body.style.overflow = ''
      state.draft = null
    }
  }

  function openCopyDrawer(sourceId, options) {
    var source = findRule(sourceId)
    if (!source) return
    var existingHosts = state.rules.map(function (rule) { return rule.host })
    var suggestion = U.suggestCopyHost(source.host, existingHosts)
    state.draft = {
      sourceId: sourceId,
      source: cloneDeep(source),
      host: suggestion.host,
      hostType: source.host_type || 'auto',
      enabled: source.enabled !== false,
      position: 'after',
      suggestion: suggestion,
      fromDetail: Boolean(options && options.fromDetail),
      activeSection: 'basic',
    }
    renderCopyDrawer()
  }

  function createCopy() {
    var draft = state.draft
    if (!draft) return
    var sourceIndex = indexOfRule(draft.sourceId)
    if (sourceIndex < 0) return

    var copy = cloneDeep(draft.source)
    delete copy.origin
    delete copy.published
    delete copy.savedOnly
    delete copy.isNew
    copy.id = 'copy-' + Date.now() + '-' + (++state.copySeq)
    copy.host = draft.host.trim()
    if (draft.hostType === 'auto') delete copy.host_type
    else copy.host_type = draft.hostType
    copy.enabled = draft.enabled
    copy.isNew = true
    copy.published = false
    copy.savedOnly = false

    var insertAt = draft.position === 'end' ? state.rules.length : sourceIndex + 1
    state.rules.splice(insertAt, 0, copy)
    state.dirty = true
    state.highlightId = copy.id

    closeTopDrawer()
    if (state.view === 'detail') showView('routes')
    renderRules()

    var placement = draft.position === 'end' ? '已添加到列表末尾' : '紧随 ' + draft.source.host + ' 之后'
    toast('已创建副本 ' + copy.host + '（第 ' + (insertAt + 1) + ' 条，' + placement + '，未保存）', {
      actionLabel: '撤销',
      onAction: function () { undoCopy(copy.id) },
    })
  }

  function undoCopy(ruleId) {
    state.rules = state.rules.filter(function (rule) { return rule.id !== ruleId })
    state.highlightId = null
    state.dirty = state.rules.some(function (rule) { return rule.isNew && !rule.published })
    renderRules()
    toast('已撤销副本')
  }

  // ---------------------------------------------------------------- save & publish

  function yamlListItemLines(rule, baseIndent) {
    var lines = U.ruleYAML(rule).split('\n')
    var base = new Array(baseIndent + 1).join('  ')
    return lines.map(function (line, index) {
      return (index === 0 ? base + '- ' : base + '  ') + line
    })
  }

  function buildDiffHTML() {
    var pending = state.rules.filter(function (rule) { return rule.isNew && !rule.published })
    var lines = []
    lines.push({ cls: 'ctx', text: '  rules:' })

    var contextRule = state.rules.filter(function (rule) { return !pending.some(function (item) { return item.id === rule.id }) })[0]
    if (contextRule) {
      yamlListItemLines(contextRule, 2).forEach(function (line) {
        lines.push({ cls: 'ctx', text: ' ' + line })
      })
      lines.push({ cls: 'ctx', text: '  # … 其余规则未变更 …' })
    }

    pending.forEach(function (rule) {
      yamlListItemLines(rule, 2).forEach(function (line) {
        lines.push({ cls: 'add', text: '+' + line })
      })
    })

    if (!pending.length) {
      lines.push({ cls: 'ctx', text: '  # 本次仅调整规则顺序，无新增规则' })
    }

    return lines.map(function (line) {
      return '<span class="diff-line ' + line.cls + '">' + esc(line.text) + '</span>'
    }).join('')
  }

  function openSaveDrawer() {
    var pending = state.rules.filter(function (rule) { return rule.isNew && !rule.published })
    var savedOnlyCount = pending.filter(function (rule) { return rule.savedOnly }).length
    var newCount = pending.length - savedOnlyCount

    var overlay = document.createElement('div')
    overlay.className = 'drawer-overlay'
    overlay.setAttribute('data-overlay', 'save')
    overlay.innerHTML = [
      '<aside class="drawer drawer-sm" role="dialog" aria-modal="true" aria-labelledby="save-title">',
      '  <header class="drawer-head">',
      '    <h2 id="save-title">保存与发布</h2>',
      '    <button type="button" class="drawer-close" data-close aria-label="关闭">×</button>',
      '  </header>',
      '  <div class="drawer-body">',
      '    <p class="form-hint" style="padding:0;margin-bottom:14px">以下为已发布配置与即将写入内容的差异。确认后可仅保存到 YAML，或保存并热加载发布。</p>',
      '    <div class="publish-steps" id="publish-steps">',
      '      <span class="publish-step" data-step="validate">校验配置</span>',
      '      <span class="publish-step" data-step="write">写入 YAML</span>',
      '      <span class="publish-step" data-step="reload">reload 热加载</span>',
      '    </div>',
      '    <pre class="diff">' + buildDiffHTML() + '</pre>',
      pending.length
        ? '    <p class="field-hint" style="margin-top:10px">' + (newCount ? '新增 ' + newCount + ' 条规则' : '') + (newCount && savedOnlyCount ? '，' : '') + (savedOnlyCount ? '已保存草稿 ' + savedOnlyCount + ' 条待发布' : '') + '；发布前会先执行 ingress validate。</p>'
        : '',
      '  </div>',
      '  <footer class="drawer-foot">',
      '    <button type="button" class="btn" data-close>取消</button>',
      '    <button type="button" class="btn" id="save-only">仅保存</button>',
      '    <button type="button" class="btn btn-primary" id="save-publish">保存并发布</button>',
      '  </footer>',
      '</aside>',
    ].join('\n')

    $('#drawer-root').appendChild(overlay)
    drawers.push(overlay)
    document.body.style.overflow = 'hidden'

    overlay.addEventListener('click', function (event) {
      if (event.target === overlay || event.target.closest('[data-close]')) closeTopDrawer()
    })

    $('#save-only', overlay).addEventListener('click', function () {
      pending.forEach(function (rule) { rule.savedOnly = true })
      state.dirty = false
      closeTopDrawer()
      renderRules()
      toast('已写入 YAML 草稿（未发布，reload 后生效）')
    })

    $('#save-publish', overlay).addEventListener('click', function () {
      runPublish(overlay, pending)
    })
  }

  function runPublish(overlay, pending) {
    var steps = $$('.publish-step', overlay)
    var saveOnly = $('#save-only', overlay)
    var savePublish = $('#save-publish', overlay)
    saveOnly.disabled = true
    savePublish.disabled = true

    var order = ['validate', 'write', 'reload']
    order.forEach(function (key, index) {
      window.setTimeout(function () {
        steps.forEach(function (step) {
          var id = step.getAttribute('data-step')
          var pos = order.indexOf(id)
          step.classList.toggle('done', pos < index)
          step.classList.toggle('active', pos === index)
        })
      }, index * 420)
    })

    window.setTimeout(function () {
      steps.forEach(function (step) { step.classList.add('done'); step.classList.remove('active') })
      pending.forEach(function (rule) {
        rule.published = true
        rule.savedOnly = false
      })
      state.dirty = false
      closeTopDrawer()
      renderRules()
      toast('已发布：配置写入 YAML 并 reload 成功')
    }, order.length * 420 + 260)
  }

  // ---------------------------------------------------------------- detail page

  function showDetail(ruleId) {
    state.detailRuleId = ruleId
    state.detailTab = 'logs'
    renderDetail()
    showView('detail')
  }

  function renderDetail() {
    var rule = findRule(state.detailRuleId)
    if (!rule) {
      showView('routes')
      return
    }
    var index = indexOfRule(rule.id) + 1
    var enabled = rule.enabled !== false
    var pathCount = (rule.paths || []).length
    var view = $('#view-detail')

    view.innerHTML = [
      '<div class="detail-back"><button type="button" class="btn btn-sm btn-ghost" data-back>← 返回路由列表</button></div>',
      '<header class="page-header">',
      '  <div>',
      '    <h1>路由详情 — ' + esc(rule.host) + '</h1>',
      '    <p>规则 #' + index + ' · 类型 ' + esc(rule.host_type || 'auto') + ' · 观测数据为原型模拟</p>',
      '  </div>',
      '  <div class="actions">',
      '    <button type="button" class="btn btn-sm btn-primary" data-copy-detail="' + esc(rule.id) + '">复制路由</button>',
      '    <span class="btn btn-sm btn-ghost" aria-disabled="true">调查此路由</span>',
      '    <span class="btn btn-sm btn-ghost" aria-disabled="true">日志</span>',
      '    <span class="btn btn-sm btn-ghost" aria-disabled="true">WAF</span>',
      '  </div>',
      '</header>',
      '<div class="scope-bar">',
      '  <span>观测范围</span>',
      '  <code>' + esc(rule.host) + '</code>',
      '  <span>规则 #' + index + '</span>',
      '  <span class="spacer"></span>',
      '  <span class="hint-inline">复制入口二：详情页右上角「复制路由」</span>',
      '</div>',
      '<div class="detail-cards">',
      '  <div class="detail-card"><div class="label">状态</div><div class="value">' + (enabled ? '启用' : '禁用') + '</div><div class="sub">' + (rule.isNew && !rule.published ? '副本 · 未发布' : '已发布配置') + '</div></div>',
      '  <div class="detail-card"><div class="label">Host 类型</div><div class="value">' + esc(rule.host_type || 'auto') + '</div><div class="sub">实际匹配 ' + U.effectiveHostType(rule.host) + '</div></div>',
      '  <div class="detail-card"><div class="label">Backend</div><div class="value">' + esc(backendSummary(rule.backend)) + '</div><div class="sub">' + (rule.connect_proxy && rule.connect_proxy.enabled ? '正向代理已启用' : '直连上游') + '</div></div>',
      '  <div class="detail-card"><div class="label">Paths</div><div class="value">' + pathCount + ' 条</div><div class="sub">Host 级路径规则</div></div>',
      '  <div class="detail-card"><div class="label">IP 白名单</div><div class="value">' + (((rule.waf && rule.waf.allow) || []).length || '未限制') + '</div><div class="sub">WAF IP 阶段</div></div>',
      '</div>',
      '<div class="panel">',
      '  <div class="detail-tabs" role="tablist">',
      '    <button type="button" role="tab" class="detail-tab ' + (state.detailTab === 'logs' ? 'active' : '') + '" data-detail-tab="logs">访问日志</button>',
      '    <button type="button" role="tab" class="detail-tab ' + (state.detailTab === 'waf' ? 'active' : '') + '" data-detail-tab="waf">WAF 事件</button>',
      '    <button type="button" role="tab" class="detail-tab ' + (state.detailTab === 'cache' ? 'active' : '') + '" data-detail-tab="cache">缓存统计</button>',
      '  </div>',
      '  <div class="detail-tab-body" id="detail-tab-body">' + detailTabBody() + '</div>',
      '</div>',
    ].join('\n')
  }

  function detailTabBody() {
    var detail = MOCK.detail
    if (state.detailTab === 'logs') {
      return [
        '<div class="metrics-row">',
        '  <div class="metric-box"><div class="label">近 5 分钟请求</div><div class="value">1,284</div><div class="delta">+8.2%</div></div>',
        '  <div class="metric-box"><div class="label">P95 延迟</div><div class="value">38ms</div><div class="delta">-3ms</div></div>',
        '  <div class="metric-box"><div class="label">5xx 比例</div><div class="value">0.04%</div><div class="delta">-0.01%</div></div>',
        '  <div class="metric-box"><div class="label">QPS</div><div class="value">4.3</div><div class="delta">+0.2</div></div>',
        '</div>',
        '<table class="data" style="min-width:0"><thead><tr><th>时间</th><th>客户端</th><th>方法</th><th>路径</th><th>状态</th><th>耗时</th><th>缓存</th></tr></thead><tbody>',
        detail.logs.map(function (row) {
          return '<tr><td>' + esc(row.time) + '</td><td>' + esc(row.ip) + '</td><td>' + esc(row.method) + '</td><td><code>' + esc(row.path) + '</code></td><td>' + row.status + '</td><td>' + row.ms + 'ms</td><td>' + (row.cache ? '<span class="badge badge-wildcard">hit</span>' : '—') + '</td></tr>'
        }).join(''),
        '</tbody></table>',
      ].join('\n')
    }
    if (state.detailTab === 'waf') {
      return [
        '<table class="data" style="min-width:0"><thead><tr><th>时间</th><th>来源 IP</th><th>规则</th><th>动作</th><th>请求路径</th></tr></thead><tbody>',
        detail.waf.map(function (row) {
          return '<tr><td>' + esc(row.time) + '</td><td>' + esc(row.ip) + '</td><td><code>' + esc(row.rule) + '</code></td><td><span class="badge ' + (row.action === 'block' ? 'badge-block' : 'badge-audit') + '">' + esc(row.action) + '</span></td><td><code>' + esc(row.path) + '</code></td></tr>'
        }).join(''),
        '</tbody></table>',
      ].join('\n')
    }
    return [
      '<div class="metrics-row">',
      '  <div class="metric-box"><div class="label">命中率</div><div class="value">' + detail.cache.hitRate + '</div><div class="delta">近 15 分钟</div></div>',
      '  <div class="metric-box"><div class="label">命中次数</div><div class="value">' + detail.cache.hits.toLocaleString() + '</div><div class="delta">cache_hit=1</div></div>',
      '  <div class="metric-box"><div class="label">未命中</div><div class="value">' + detail.cache.misses.toLocaleString() + '</div><div class="delta">回源</div></div>',
      '  <div class="metric-box"><div class="label">平均 TTL</div><div class="value">' + detail.cache.avgTtl + '</div><div class="delta">backend.cache.ttl</div></div>',
      '</div>',
      '<table class="data" style="min-width:0"><thead><tr><th>缓存键路径</th><th>命中次数</th></tr></thead><tbody>',
      detail.cache.topPaths.map(function (row) {
        return '<tr><td><code>' + esc(row.path) + '</code></td><td>' + row.hits.toLocaleString() + '</td></tr>'
      }).join(''),
      '</tbody></table>',
    ].join('\n')
  }

  // ---------------------------------------------------------------- topology / match

  function renderTopology() {
    $('#topology-body').innerHTML = '<div class="topo-list">' + state.rules.map(function (rule, index) {
      var paths = (rule.paths || []).map(function (item) { return item.path })
      return [
        '<div class="topo-row">',
        '  <div>',
        '    <code>' + esc(rule.host) + '</code>',
        '    <div class="topo-meta">#' + (index + 1) + ' · ' + (rule.host_type || 'auto') + ' · ' + (rule.enabled !== false ? '启用' : '禁用') + '</div>',
        '  </div>',
        '  <div class="topo-meta">',
        '    <span class="arrow">→</span>' + esc(ruleSummary(rule)),
        paths.length ? '<br /><span class="arrow">↳</span>paths: ' + paths.map(function (path) { return '<code>' + esc(path) + '</code>' }).join('、') : '',
        '  </div>',
        '</div>',
      ].join('\n')
    }).join('') + '</div>'
  }

  function matchHost(ruleHost, host) {
    var type = U.effectiveHostType(ruleHost)
    var normalized = String(host || '').toLowerCase()
    var pattern = String(ruleHost || '').toLowerCase()
    if (type === 'exact') return normalized === pattern
    if (type === 'wildcard') {
      try {
        var escaped = pattern.split('*').map(function (part) {
          return part.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
        }).join('.*')
        return new RegExp('^' + escaped + '$').test(normalized)
      } catch (e) {
        return false
      }
    }
    try {
      return new RegExp(pattern).test(normalized)
    } catch (e) {
      return false
    }
  }

  function runMatch() {
    var host = $('#match-host').value.trim()
    var path = $('#match-path').value.trim() || '/'
    var resultBox = $('#match-result')
    var hit = null

    for (var i = 0; i < state.rules.length; i++) {
      var rule = state.rules[i]
      if (!matchHost(rule.host, host)) continue
      var paths = rule.paths || []
      if (!paths.length) {
        hit = { rule: rule, index: i, path: null }
        break
      }
      var pathHit = paths.filter(function (item) {
        var p = item.path || '/'
        return p === '/' || path === p || path.indexOf(p.replace(/\/$/, '') + '/') === 0
      })[0]
      if (pathHit) {
        hit = { rule: rule, index: i, path: pathHit }
        break
      }
    }

    resultBox.hidden = false
    if (!hit) {
      resultBox.innerHTML = '<span class="miss">未命中任何路由规则</span> · 将返回 404（未配置 fallback 时）'
      return
    }
    var rule = hit.rule
    resultBox.innerHTML = [
      '命中规则 <code>#' + (hit.index + 1) + '</code> <code>' + esc(rule.host) + '</code>（' + (rule.enabled !== false ? '启用' : '<span class="miss">禁用，会被跳过</span>') + '）',
      '<br />匹配后端：' + esc(ruleSummary(rule)),
      hit.path ? '<br />Path 规则：<code>' + esc(hit.path.path) + '</code>' : '<br />Path 规则：使用 Host 级 backend',
      '<br /><button type="button" class="btn btn-sm btn-ghost" data-detail="' + esc(rule.id) + '">查看路由详情</button>',
    ].join('')
  }

  // ---------------------------------------------------------------- wiring

  function init() {
    $('#rules-tbody').addEventListener('click', function (event) {
      var detail = event.target.closest('[data-detail]')
      if (detail) { showDetail(detail.getAttribute('data-detail')); return }
      var menu = event.target.closest('[data-menu]')
      if (menu) { openRowMenu(menu, menu.getAttribute('data-menu')); return }
      var paths = event.target.closest('[data-paths]')
      if (paths) { toast('Paths 编辑沿用现有弹窗（原型聚焦复制流程）'); return }
      var edit = event.target.closest('[data-edit]')
      if (edit) { toast('编辑沿用现有规则弹窗（原型聚焦复制流程）'); return }
    })

    $$('[data-route-tab]').forEach(function (tab) {
      tab.addEventListener('click', function () {
        var id = tab.getAttribute('data-route-tab')
        $$('[data-route-tab]').forEach(function (item) {
          var active = item === tab
          item.classList.toggle('active', active)
          item.setAttribute('aria-selected', active ? 'true' : 'false')
        })
        $$('.route-panel').forEach(function (panel) {
          panel.hidden = panel.getAttribute('data-route-panel') !== id
        })
        if (id === 'topology') renderTopology()
      })
    })

    $('#btn-add').addEventListener('click', function () {
      toast('「添加」沿用现有规则弹窗（原型聚焦复制流程）')
    })

    $('#btn-save').addEventListener('click', openSaveDrawer)

    $('#btn-match').addEventListener('click', runMatch)
    $('#match-result').addEventListener('click', function (event) {
      var detail = event.target.closest('[data-detail]')
      if (detail) showDetail(detail.getAttribute('data-detail'))
    })

    $('#view-detail').addEventListener('click', function (event) {
      var back = event.target.closest('[data-back]')
      if (back) { showView('routes'); renderRules(); return }
      var copyBtn = event.target.closest('[data-copy-detail]')
      if (copyBtn) { openCopyDrawer(copyBtn.getAttribute('data-copy-detail'), { fromDetail: true }); return }
      var tab = event.target.closest('[data-detail-tab]')
      if (tab) {
        state.detailTab = tab.getAttribute('data-detail-tab')
        renderDetail()
      }
    })

    document.addEventListener('keydown', function (event) {
      if (event.key !== 'Escape') return
      if (state.menuEl) { closeRowMenu(); return }
      if (drawers.length) closeTopDrawer()
    })

    renderRules()
    handleDeepLink()
  }

  function handleDeepLink() {
    var hash = window.location.hash || ''
    var copyMatch = hash.match(/copy=([\w-]+)/)
    if (copyMatch && findRule(copyMatch[1])) {
      openCopyDrawer(copyMatch[1])
      return
    }
    var detailMatch = hash.match(/detail=([\w-]+)/)
    if (detailMatch && findRule(detailMatch[1])) {
      showDetail(detailMatch[1])
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init)
  } else {
    init()
  }
})()
