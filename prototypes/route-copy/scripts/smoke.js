/**
 * 纯逻辑冒烟测试：node scripts/smoke.js
 * 覆盖副本 Host 生成、冲突检测、YAML 片段序列化。
 */
'use strict'

const assert = require('node:assert/strict')
const U = require('../js/copy-utils.js')

// --- host 类型推断 -------------------------------------------------------
assert.equal(U.effectiveHostType('api.example.com'), 'exact')
assert.equal(U.effectiveHostType('*.example.com'), 'wildcard')
assert.equal(U.effectiveHostType('^legacy\\.example\\.com$'), 'regex')
assert.equal(U.effectiveHostType(''), 'exact')

// --- 自动生成区分名 ------------------------------------------------------
assert.equal(U.suggestCopyHost('api.example.com', []).host, 'api-copy.example.com')
assert.equal(U.suggestCopyHost('www.example.com', []).host, 'www-copy.example.com')
assert.equal(U.suggestCopyHost('localhost', []).host, 'localhost-copy')

// 已存在同名时顺延
const occupied = ['admin-copy.example.com']
const avoided = U.suggestCopyHost('admin.example.com', occupied)
assert.equal(avoided.host, 'admin-copy-2.example.com')
assert.equal(avoided.collisionAvoided, true)
assert.equal(avoided.firstSuggestion, 'admin-copy.example.com')

// 通配符保留 *. 前缀，改写后面的 label
assert.equal(U.suggestCopyHost('*.shop.example.com', []).host, '*.shop-copy.example.com')

// 正则 Host 不自动改名，要求人工确认
const regex = U.suggestCopyHost('^legacy\\.example\\.com$', [])
assert.equal(regex.host, '^legacy\\.example\\.com$')
assert.equal(regex.auto, false)
assert.equal(regex.reason, 'regex')

// 复制副本的副本：在已有 -copy 基础上继续顺延
assert.equal(U.suggestCopyHost('api-copy.example.com', []).host, 'api-copy-2.example.com')

// --- 冲突检测 ------------------------------------------------------------
const rules = [
  { id: 'a', host: 'api.example.com' },
  { id: 'b', host: 'API-Copy.Example.com' },
]
assert.equal(U.findHostConflict('api.example.com', rules).index, 0)
assert.equal(U.findHostConflict('api-copy.example.com', rules).index, 1) // 大小写不敏感
assert.equal(U.findHostConflict('nope.example.com', rules), null)
assert.equal(U.findHostConflict('  ', rules), null)

// --- Host 校验 -----------------------------------------------------------
assert.equal(U.validateHost(''), 'Host 不能为空')
assert.ok(U.validateHost('bad host.example.com'))
assert.ok(U.validateHost('*.'))
assert.equal(U.validateHost('api.example.com'), '')
assert.equal(U.validateHost('*.example.com'), '')

// --- YAML 序列化 ---------------------------------------------------------
const yaml = U.ruleYAML({
  id: 'copy-1',
  host: '*.shop-copy.example.com',
  host_type: 'auto',
  enabled: true,
  backend: { type: 'service', service: { name: 'orders', port: 8080 }, cache: { enabled: true, ttl: 300 } },
  paths: [{ path: '/v2/orders' }, { path: '/v2/users', strip_prefix: true }],
  rate_limit: { requests: 100, period: 60, key: 'ip' },
  disabled_meta: 'ignored',
})
assert.match(yaml, /host: '\*\.shop-copy\.example\.com'/) // YAML 里 * 开头必须加引号
assert.match(yaml, /backend:\n {2}type: service/)
assert.match(yaml, /paths:\n {2}- path: \/v2\/orders/)
assert.match(yaml, /- path: \/v2\/users/)
assert.match(yaml, /ttl: 300/)
assert.doesNotMatch(yaml, /host_type/) // auto 省略
assert.doesNotMatch(yaml, /disabled_meta/)

const disabledYaml = U.ruleYAML({
  host: '^legacy\\.example\\.com$',
  host_type: 'regex',
  enabled: false,
  backend: { type: 'redirect', redirect: { url: 'https://www.example.com', status: 'permanent' } },
})
assert.ok(disabledYaml.includes("host: '^legacy\\.example\\.com$'"), disabledYaml)
assert.match(disabledYaml, /host_type: regex/)
assert.match(disabledYaml, /enabled: false/)

// --- 与模拟数据一致性（host 全部可推断、id 唯一） ------------------------
global.window = {}
require('../js/mock-data.js')
const mock = global.window.MOCK
const ids = new Set()
for (const rule of mock.rules) {
  assert.ok(rule.host, 'rule host missing')
  assert.equal(U.validateHost(rule.host), '')
  assert.equal(U.effectiveHostType(rule.host), U.effectiveHostType(rule.host))
  assert.ok(!ids.has(rule.id), 'duplicate rule id: ' + rule.id)
  ids.add(rule.id)
}
const source = mock.rules.find((rule) => rule.host === 'api.example.com')
assert.ok(source)
assert.equal(U.suggestCopyHost(source.host, mock.rules.map((r) => r.host)).host, 'api-copy.example.com')
assert.equal(U.suggestCopyHost('admin.example.com', mock.rules.map((r) => r.host)).host, 'admin-copy-2.example.com')

console.log('copy-utils smoke: all assertions passed')
