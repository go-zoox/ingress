/* eslint-disable no-unused-vars */
const MOCK = {
  instance: {
    version: "v1.16.2",
    pid: 48291,
    configPath: "/etc/ingress/ingress.yaml",
    configHash: "a3f2c91e",
    listenHTTP: 8080,
    listenHTTPS: 8443,
    uptime: "2d 14h 32m",
    lastReload: "2026-05-20 09:41:03",
    lastReloadOK: true,
    rulesCount: 12,
    wafEnabled: true,
    wafLogOnly: false,
  },
  routes: [
    {
      id: 1,
      host: "api.example.com",
      hostType: "exact",
      path: "/",
      pathType: "prefix",
      backendType: "service",
      target: "api.internal:8080",
      waf: "inherit",
      cache: false,
    },
    {
      id: 2,
      host: "api.example.com",
      hostType: "exact",
      path: "/v2",
      pathType: "prefix",
      backendType: "service",
      target: "api-v2.internal:8080",
      waf: "inherit",
      cache: true,
    },
    {
      id: 3,
      host: "*.cdn.example.com",
      hostType: "wildcard",
      path: "/",
      pathType: "prefix",
      backendType: "service",
      target: "minio.internal:9000",
      waf: "inherit",
      cache: true,
    },
    {
      id: 4,
      host: "^([a-z0-9-]+)\\.inlets\\.example\\.com$",
      hostType: "regex",
      path: "/",
      pathType: "prefix",
      backendType: "service",
      target: "${1}.tunnel:443",
      waf: "patched",
      cache: false,
    },
    {
      id: 5,
      host: "admin.internal",
      hostType: "exact",
      path: "/healthz",
      pathType: "exact",
      backendType: "handler",
      target: "static:/var/www/status",
      waf: "off",
      cache: false,
    },
    {
      id: 6,
      host: "legacy.example.com",
      hostType: "exact",
      path: "/",
      pathType: "prefix",
      backendType: "redirect",
      target: "https://www.example.com$request_uri",
      waf: "inherit",
      cache: false,
    },
  ],
  wafEvents: [
    {
      time: "09:44:12",
      action: "block",
      rule: "sql-injection-uri",
      host: "api.example.com",
      path: "/search?q=1' OR '1'='1",
      client: "203.0.113.44",
    },
    {
      time: "09:42:01",
      action: "audit",
      rule: "scanner-ua",
      host: "api.example.com",
      path: "/",
      client: "198.51.100.8",
    },
    {
      time: "09:38:55",
      action: "block",
      rule: "ip-deny",
      host: "waf-demo.example.com",
      path: "/admin",
      client: "192.0.2.99",
    },
    {
      time: "09:35:20",
      action: "audit",
      rule: "path-traversal",
      host: "*.cdn.example.com",
      path: "/../../etc/passwd",
      client: "203.0.113.12",
    },
  ],
  certs: [
    {
      domain: "api.example.com",
      issuer: "Let's Encrypt R3",
      notAfter: "2026-08-14",
      daysLeft: 86,
      status: "ok",
    },
    {
      domain: "*.cdn.example.com",
      issuer: "Internal CA",
      notAfter: "2026-06-02",
      daysLeft: 13,
      status: "warn",
    },
    {
      domain: "legacy.example.com",
      issuer: "Let's Encrypt R3",
      notAfter: "2026-04-28",
      daysLeft: -22,
      status: "expired",
    },
  ],
  accessLogs: [
    "203.0.113.44 api.example.com -> api.internal:8080 \"GET /api/users HTTP/1.1\" 200 12ms cache_hit=0 real_ip=203.0.113.44 upstream_status=200",
    "198.51.100.8 api.example.com -> api.internal:8080 \"POST /api/login HTTP/1.1\" 401 8ms cache_hit=0 real_ip=198.51.100.8 upstream_status=401",
    "203.0.113.12 cdn.example.com -> minio.internal:9000 \"GET /assets/app.js HTTP/1.1\" 200 3ms cache_hit=1 real_ip=203.0.113.12 upstream_status=200",
    "192.0.2.99 waf-demo.example.com -> httpbin.org:443 \"GET /admin HTTP/1.1\" 403 2ms cache_hit=0 real_ip=192.0.2.99 waf_block=1",
    "10.0.0.5 admin.internal -> static:/var/www/status \"GET /healthz HTTP/1.1\" 200 1ms cache_hit=0 real_ip=10.0.0.5",
    "203.0.113.44 api.example.com -> api.internal:8080 \"GET /search?q=test HTTP/1.1\" 200 45ms cache_hit=0 real_ip=203.0.113.44 upstream_status=200",
    "203.0.113.44 legacy.example.com -> redirect \"GET / HTTP/1.1\" 301 0ms cache_hit=0",
    "198.51.100.8 tunnel-a.inlets.example.com -> tunnel-a.tunnel:443 \"GET / HTTP/1.1\" 502 12003ms upstream_status=502",
  ],
  yamlBaseline: `version: v1
port: 8080

https:
  port: 8443
  redirect_from_http:
    permanent: true
  ssl:
    - domain: api.example.com
      cert:
        certificate: /etc/ingress/certs/api.pem
        certificate_key: /etc/ingress/certs/api-key.pem

waf:
  enabled: true
  log_only: false
  builtin: true

healthcheck:
  outer:
    enable: true
    path: /healthz
  inner:
    enable: true
    interval: 30
    timeout: 5

rules:
  - host: api.example.com
    backend:
      service:
        name: api.internal
        port: 8080
    paths:
      - path: /v2
        backend:
          service:
            name: api-v2.internal
            port: 8080
            cache:
              enabled: true

  - host: "*.cdn.example.com"
    host_type: wildcard
    backend:
      service:
        name: minio.internal
        port: 9000
`,
};
