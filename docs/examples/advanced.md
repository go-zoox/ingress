# Advanced Examples

This page provides advanced configuration examples.

## Regex Host Matching with Path Rewriting

```yaml
version: v1
port: 8080

rules:
  - host: ^t-(\w+).example.work
    host_type: regex
    backend:
      service:
        name: task.${host.1}.svc
        port: 8080
    paths:
      - path: /api/v1/([^/]+)
        backend:
          service:
            name: ${path.1}.${host.1}.example.work
            port: 8080
            request:
              path:
                rewrites:
                  - ^/api/v1/([^/]+):/api/v1/task/$1
```

This example:
- Matches hosts like `t-myapp.example.work` using regex
- Routes using scoped captures in `service.name` (`${host.<index>}` and `${path.<index>}`)
- Rewrites paths for specific API routes

## Wildcard Host Matching

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

This matches any subdomain of `example.work`.

## Complex Path Rewriting

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

## Health Checks with Multiple Services

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

## Redis Caching

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

## Complete Configuration

A complete example with all features:

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

## Backend redirect (advanced)

`backend.redirect` and `backend.service` cannot both be set on the **same** backend. Redirect-only backends need no `service` block.

### Regex host with captures in `redirect.url`

Same templating as `service.name`: `$1`, `${host.1}`, etc.

```yaml
rules:
  - host: '^bigscreen-([^.]+)\.example\.com$'
    host_type: regex
    backend:
      redirect:
        url: https://bigscreen-$1.other.example.com
```

### Host fallback redirect + path services

Unmatched paths use the host-level `redirect`; matched paths use `service`.

```yaml
rules:
  - host: app.example.com
    backend:
      redirect:
        url: https://www.example.com/landing
    paths:
      - path: ^/api/
        backend:
          service:
            name: api-svc
            port: 8080
            protocol: http
```

### Path-only redirect with `${path.N}`

```yaml
rules:
  - host: api.example.com
    backend:
      service:
        name: default-upstream
        port: 8080
        protocol: http
    paths:
      - path: ^/go/([^/]+)$
        backend:
          redirect:
            url: https://seg.${path.1}.example.com/resource
```
