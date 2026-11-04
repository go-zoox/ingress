/**
 * 模拟数据：与 ingress 的 rules / services 结构保持同构（只保留原型用到的字段）。
 * 仅用于产品原型，不参与编译。
 */
window.MOCK = {
  configPath: 'examples/admin-console/ingress.yaml',
  services: [
    { name: 'web', port: 8080, protocol: 'http', mode: 'internal' },
    { name: 'orders', port: 8080, protocol: 'http', mode: 'internal' },
    { name: 'admin', port: 9090, protocol: 'http', mode: 'internal' },
  ],
  rules: [
    {
      id: 'r1',
      host: 'www.example.com',
      host_type: 'auto',
      enabled: true,
      backend: {
        type: 'service',
        service: { name: 'web', port: 8080, protocol: 'http', mode: 'internal' },
        cache: { enabled: true, ttl: 300 },
      },
      paths: [{ path: '/', strip_prefix: true }],
      waf: { allow: ['10.0.0.0/8'], deny: [] },
      security: { profile: 'strict' },
    },
    {
      id: 'r2',
      host: 'api.example.com',
      host_type: 'auto',
      enabled: true,
      backend: {
        type: 'service',
        service: { name: 'orders', port: 8080, protocol: 'http', mode: 'internal', auth: { type: 'basic' } },
      },
      paths: [{ path: '/v2/orders' }, { path: '/v2/users', strip_prefix: true }],
      rate_limit: { requests: 100, period: 60, key: 'ip' },
      security: { profile: 'api' },
    },
    {
      id: 'r3',
      host: 'admin.example.com',
      host_type: 'auto',
      enabled: true,
      backend: {
        type: 'service',
        service: { name: 'admin', port: 9090, protocol: 'http', mode: 'internal' },
      },
      paths: [],
      waf: { allow: ['10.10.0.0/16', '192.168.0.0/16'], deny: [] },
    },
    {
      id: 'r4',
      host: 'admin-copy.example.com',
      host_type: 'auto',
      enabled: false,
      backend: {
        type: 'service',
        service: { name: 'admin', port: 9090, protocol: 'http', mode: 'internal' },
      },
      paths: [],
      origin: 'seed-copy',
      savedOnly: true,
    },
    {
      id: 'r5',
      host: '*.shop.example.com',
      host_type: 'auto',
      enabled: true,
      backend: {
        type: 'handler',
        handler: { type: 'file_server', root_dir: '/srv/shop' },
        cache: { enabled: true, ttl: 60 },
      },
      paths: [{ path: '/assets' }],
    },
    {
      id: 'r6',
      host: '^legacy\\.example\\.com$',
      host_type: 'auto',
      enabled: false,
      backend: {
        type: 'redirect',
        redirect: { url: 'https://www.example.com', status: 'permanent', preserve_path: true },
      },
      paths: [],
    },
    {
      id: 'r7',
      host: 'orders-api.example.com',
      host_type: 'auto',
      enabled: true,
      backend: {
        type: 'service',
        service: { name: 'orders', port: 8080, protocol: 'http', mode: 'internal' },
      },
      paths: [{ path: '/api/v1' }],
      connect_proxy: {
        enabled: true,
        upstream: { name: 'egress-proxy', port: 3128 },
        allowed_targets: ['^api\\..*:443$'],
      },
    },
  ],
  detail: {
    logs: [
      { time: '21:01:12', ip: '10.20.3.41', method: 'GET', path: '/v2/orders?page=1', status: 200, ms: 23, cache: '' },
      { time: '21:01:09', ip: '10.20.3.18', method: 'GET', path: '/v2/orders/88213', status: 200, ms: 18, cache: '' },
      { time: '21:00:58', ip: '10.20.9.7', method: 'POST', path: '/v2/orders', status: 201, ms: 64, cache: '' },
      { time: '21:00:43', ip: '10.20.3.41', method: 'GET', path: '/v2/users/me', status: 401, ms: 4, cache: '' },
      { time: '21:00:31', ip: '10.20.4.2', method: 'GET', path: '/v2/orders/88210', status: 200, ms: 31, cache: 'hit' },
    ],
    waf: [
      { time: '20:58:12', ip: '203.0.113.24', rule: 'sqli-basic', action: 'block', path: '/v2/orders?q=1%27+or+1%3D1' },
      { time: '20:41:03', ip: '198.51.100.7', rule: 'path-traversal', action: 'audit', path: '/v2/orders/../../etc/passwd' },
      { time: '20:12:55', ip: '203.0.113.99', rule: 'scanner-ua', action: 'block', path: '/v2/orders' },
    ],
    cache: {
      hitRate: '92.4%',
      hits: 18234,
      misses: 1502,
      avgTtl: '300s',
      topPaths: [
        { path: '/v2/orders?page=1', hits: 8210 },
        { path: '/v2/orders/88213', hits: 4102 },
        { path: '/v2/users?limit=20', hits: 2603 },
      ],
    },
  },
}
