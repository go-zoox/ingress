# 高级示例

此页面提供高级配置示例。

## 带路径重写的正则主机匹配

```yaml
version: v1
port: 8080

rules:
  - host: ^t-(\w+).example.work
    host_type: regex
    backend:
      service:
        name: task.$1.svc
        port: 8080
    paths:
      - path: /api/v1/([^/]+)
        backend:
          service:
            name: $1.example.work
            port: 8080
            request:
              path:
                rewrites:
                  - ^/api/v1/([^/]+):/api/v1/task/$1
```

此示例：
- 使用正则表达式匹配像 `t-myapp.example.work` 这样的主机
- 使用捕获组 `$1` 路由到 `task.myapp.svc`
- 为特定 API 路由重写路径

## 通配符主机匹配

```yaml
version: v1
port: 8080

rules:
  - host: '*.example.work'
    host_type: wildcard
    backend:
      service:
        name: wildcard-service
        port: 8080
```

这匹配 `example.work` 的任何子域。

## 复杂路径重写

```yaml
version: v1
port: 8080

rules:
  - host: httpbin.example.work
    backend:
      service:
        name: httpbin.zcorky.com
        port: 443
        protocol: https
        request:
          host:
            rewrite: true
          path:
            rewrites:
              - ^/ip3/(.*):/$1
              - ^/ip2:/ip
    paths:
      - path: /httpbin.org
        backend:
          service:
            name: httpbin.org
            port: 443
            protocol: https
            request:
              path:
                rewrites:
                  - ^/httpbin.org/(.*):/$1
```

## 带多个服务的健康检查

```yaml
version: v1
port: 8080

healthcheck:
  outer:
    enable: true
    path: /healthz
    ok: true
  inner:
    enable: true
    interval: 30
    timeout: 5

rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        healthcheck:
          enable: true
          method: GET
          path: /health
          status: [200]
```

## Redis 缓存

```yaml
version: v1
port: 8080

cache:
  ttl: 60
  engine: redis
  host: redis.example.com
  port: 6379
  password: your-password
  db: 0
  prefix: ingress:

rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
```

## 完整配置

包含所有功能的完整示例：

```yaml
version: v1
port: 8080

cache:
  ttl: 30

https:
  port: 8443
  ssl:
    - domain: example.com
      cert:
        certificate: /etc/ssl/example.com/fullchain.pem
        certificate_key: /etc/ssl/example.com/privkey.pem

healthcheck:
  outer:
    enable: true
    path: /healthz
    ok: true
  inner:
    enable: true
    interval: 30
    timeout: 5

fallback:
  service:
    name: httpbin.org
    port: 443

rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        auth:
          type: basic
          basic:
            users:
              - username: admin
                password: admin123
        healthcheck:
          enable: true
          method: GET
          path: /health
          status: [200]
        request:
          host:
            rewrite: true
          path:
            rewrites:
              - ^/api/v1/(.*):/api/v2/$1
          headers:
            X-Forwarded-Proto: https
          timeout: 30
    paths:
      - path: /api
        backend:
          service:
            name: api-service
            port: 8080
            auth:
              type: bearer
              bearer:
                tokens:
                  - api-token-123
```
