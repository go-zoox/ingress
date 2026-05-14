# Configuration Reference

Ingress uses YAML configuration files to define routing rules, authentication, SSL certificates, and other settings.

## Configuration Structure

```yaml
version: v1                    # Configuration version
port: 8080                     # HTTP port (default: 8080)
# enable_h2c: false            # Optional: cleartext HTTP/2 (h2c) on port; unsafe on public networks

# Cache configuration
cache:
  ttl: 30                      # Cache TTL in seconds
  # engine: redis              # Optional: use Redis cache
  # host: 127.0.0.1
  # port: 6379
  # password: '123456'
  # db: 2

# HTTPS configuration
https:
  port: 8443                   # HTTPS port
  # enable_http3: false        # Optional: HTTP/3 (QUIC) on UDP; needs TLS
  # http3_port: 8443           # Optional: UDP port (default: same as https.port)
  # http3_altsvc_max_age: 86400 # Optional: Alt-Svc ma= (seconds); negative disables header
  # redirect_from_http:
  #   disabled: false          # Optional: false by default; when false and https.port is set, force HTTP -> HTTPS
  #   permanent: true          # Optional: true=301, false=302
  #   exclude_paths:           # Optional: exact paths to skip redirect
  #     - /healthz
  ssl:
    - domain: example.com
      cert:
        certificate: /path/to/cert.pem
        certificate_key: /path/to/key.pem

# Health check configuration
healthcheck:
  outer:
    enable: true               # Enable outer health check
    path: /healthz             # Health check endpoint path
    ok: true                   # Always return OK
  inner:
    enable: true               # Enable inner service health checks
    interval: 30               # Check interval in seconds
    timeout: 5                 # Check timeout in seconds

# Fallback service (used when no rule matches)
fallback:
  service:
    name: httpbin.org
    port: 443

# Routing rules
rules:
  - host: example.com
    backend:
      # Omit backend.type when only one of service | handler | redirect applies — runnable twin spelling:
      # examples/basic/ingress.yaml (explicit type: service vs omission).
      service:
        name: backend-service
        port: 8080
```

## Configuration Fields

### Top-Level Fields

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `version` | string | Configuration version | `v1` |
| `port` | int | HTTP port to listen on | `8080` |
| `enable_h2c` | bool | Cleartext HTTP/2 (h2c) on the HTTP port | `false` |
| `cache` | object | Cache configuration | - |
| `https` | object | HTTPS configuration | - |
| `healthcheck` | object | Health check configuration | - |
| `fallback` | object | Fallback backend | - |
| `rules` | array | Routing rules | `[]` |
| `waf` | object | Optional WAF baseline; route patches use **`rules[].waf`** YAML maps ([WAF guide](waf.md)) | _(inactive when omitted or `enabled: false`)_ |

### WAF (`waf` / `rules[].waf`)

| Field | Type | Description |
|-------|------|-------------|
| `enabled` | bool | Master switch (`false` or omitted baseline keeps WAF off unless a route sets `enabled: true`). |
| `trust_proxy` | bool | Use `X-Forwarded-For` for client IP (default direct `RemoteAddr`). |
| `xff_index` | int | Segment index (`0` = leftmost IP; negatives count from the right). |
| `log_only` | bool | Global audit — `[waf audit]` logs, no blocking. |
| `block_status_code` | int | Status on block (unset/`0` → `403`). |
| `block_content_type` | string | Response header when blocking (`text/plain; charset=utf-8` default). |
| `block_body` | string | Response body when blocking. |
| `disable_builtin` | bool | Omit embedded starters when `true` (see [built-in rules](waf.md#built-in-starter-rules)). |
| `deny` | string array | IPs / CIDRs (deny first). |
| `allow` | string array | Non-empty ⇒ only listed nets survive IP phase. |
| `rules` | array | Custom signatures (`id`, `pattern`, `type`, `targets`, optional per-rule `log_only`). Same `id` in a route map replaces inherited rule metadata. |

### Cache Configuration

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `ttl` | int | Cache TTL in seconds | `60` |
| `host` | string | Redis host (if using Redis) | - |
| `port` | int | Redis port | `6379` |
| `password` | string | Redis password | - |
| `db` | int | Redis database number | `0` |
| `prefix` | string | Cache key prefix | - |

### HTTPS Configuration

| Field | Type | Description |
|-------|------|-------------|
| `port` | int | HTTPS port to listen on |
| `enable_http3` | bool | Enable HTTP/3 (QUIC) on UDP when TLS is configured |
| `http3_port` | int | UDP port for HTTP/3; `0` means same as `https.port` |
| `http3_altsvc_max_age` | int | `Alt-Svc` `ma=` in seconds; `0` uses server default; negative disables `Alt-Svc` |
| `redirect_from_http.disabled` | bool | Disable forced HTTP -> HTTPS redirect (`false` by default, which means enabled when `https.port` is set) |
| `redirect_from_http.permanent` | bool | Use `301` when true, `302` when false |
| `redirect_from_http.with_origin_method_and_body` | bool | When true, use `308`/`307` instead of `301`/`302` so method and body are preserved (default `false`) |
| `redirect_from_http.exclude_paths` | array | Exact request paths that skip forced redirect |
| `ssl` | array | SSL certificate configurations |

HTTP/2 over TLS is negotiated automatically when HTTPS is enabled (no extra field).

#### SSL Certificate

| Field | Type | Description |
|-------|------|-------------|
| `domain` | string | Domain name for the certificate |
| `cert.certificate` | string | Path to certificate file |
| `cert.certificate_key` | string | Path to private key file |

### Health Check Configuration

#### Outer Health Check

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `enable` | bool | Enable outer health check | `false` |
| `path` | string | Health check endpoint path | `/healthz` |
| `ok` | bool | Always return OK | `false` |

#### Inner Health Check

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `enable` | bool | Enable inner health checks | `false` |
| `interval` | int | Check interval in seconds | `30` |
| `timeout` | int | Check timeout in seconds | `5` |

### Fallback Configuration

The fallback backend is used when no routing rule matches the request.

If `service.request.host.rewrite` is **omitted**, `Host` is still aligned to the fallback upstream. Set `service.request.host.rewrite: false` only when you must preserve the client `Host`. Optional `mode: external` documents the same default Host behavior.

```yaml
fallback:
  # mode: internal            # optional — internal (default) | external
  service:
    name: fallback-service
    port: 8080
    protocol: http              # http or https
    # request:
    #   host:
    #     rewrite: false        # optional: preserve client Host
```

### Rules Configuration

Rules define how requests are routed to backend services. See the [Routing Guide](/guide/routing) for detailed information.

For **`backend.type`**, this snippet **mixes styles**: the rule `backend` sets **`type: service`** explicitly, while **`paths[].backend`** blocks omit **`type`** (they infer **`service`** or **`handler`** from their nested blocks).

```yaml
rules:
  - host: example.com           # Host to match
    # host_type: optional — omit or `auto` to infer exact vs regex vs wildcard from `host` at compile time
    # explicit values: exact | regex | wildcard
    backend:
      type: service             # optional — omit when only service applies (see examples/basic/ingress.yaml)
      mode: internal            # optional — internal (default) | external (default Host to upstream when rewrite omitted)
      service:
        name: backend-service
        port: 8080
        protocol: http          # http or https
        auth:                   # Authentication (optional)
          type: basic
          basic:
            users:
              - username: admin
                password: admin123
        healthcheck:            # Service health check (optional)
          enable: true
          method: GET
          path: /health
          status: [200]
        request:
          host:
            rewrite: true       # optional explicit override; often omit when mode: external
          path:
            rewrites:          # Path rewrite rules
              - ^/api/v1:/api/v2
          headers:             # Additional headers
            X-Custom-Header: value
          query:               # Query parameters
            key: value
          delay: 0              # Delay in milliseconds
          timeout: 30           # Timeout in seconds
      # redirect: ...           # Only redirect block — see Routing guide (backend.type optional when unique)
    paths:                      # Path-based routing (optional)
      - path: /api
        backend:
          mode: internal          # optional on path backends
          service:
            name: api-service
            port: 8080
      - path: /healthz
        backend:
          handler:
            status_code: 200
            headers:
              Content-Type: application/json
            body: |
              {"ok": true}
```

### `rules[].backend` and `paths[].backend` fields

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `type` | string | `service`, `handler`, or `redirect` (often **omitted** and inferred) | inferred |
| `mode` | string | `internal` keeps client `Host` unless `service.request.host.rewrite` is set; `external` defaults `Host` to the upstream | `internal` |
| `service` | object | Upstream when type is `service` | - |
| `handler` | object | Handler when type is `handler` | - |
| `redirect` | object | Redirect when type is `redirect` | - |

`mode` applies per backend block (host-level or path-level). It affects **proxy** backends only; **`handler`** / **`redirect`** ignore it for behavior, but **`ingress validate`** still accepts `internal` / `external`.

## Access Log Fields

Ingress access logs use an application-level fixed format (not Nginx `log_format`). The original core fields are kept, with the following extra fields appended:

- `referer`: value from `Referer`; `-` when empty
- `ua`: value from `User-Agent`; `-` when empty
- `xff`: value from `X-Forwarded-For`; `-` when empty
- `real_ip`: value from `X-Real-IP`; falls back to request remote address; `-` when unavailable
- `tls_protocol`: TLS version (for example `TLS 1.3`); `-` for non-TLS requests
- `tls_cipher`: TLS cipher suite name; `-` for non-TLS requests
- `upstream_status`: upstream response status (handler branch uses handler status)
- `upstream_response_length`: upstream response length (`-1` when unknown)
- `upstream_response_time`: upstream response duration in Go `time.Duration` text form

Example:

```text
[host: example.com, target: http://backend:8080] "GET /api HTTP/1.1" 200 12.3ms real_ip="10.0.0.9" referer="https://portal.example.com/" ua="curl/8.7.1" xff="10.0.0.1" tls_protocol="TLS 1.3" tls_cipher="TLS_AES_128_GCM_SHA256" upstream_status=200 upstream_response_length=512 upstream_response_time=12.3ms
```

Note: there is currently no standalone field exactly equivalent to Nginx `$body_bytes_sent`; if needed, derive it via downstream log/metrics aggregation.

## Environment Variables

You can override some configuration using environment variables:

- `CONFIG`: Path to configuration file
- `PORT`: HTTP port number

## Configuration Validation

Ingress validates configuration when the process starts and when you run **`ingress validate`**. Errors prevent startup or a successful reload.

Static checks include router compilation (host/path regex), HTTPS/TLS shape, **per-backend `mode` (`internal` / `external`)**, and **per-backend** consistency (**`backend.type`** is usually omitted and inferred as **`service`**, **`handler`**, or **`redirect`**):

- **`backend.type` is optional.** With **`type` omitted**, Ingress **infers** the mode when exactly one of `service`, `handler`, or `redirect` looks configured; otherwise validation fails and asks for an explicit **`backend.type`**.
- With an explicit type, only the matching block is allowed (for example **`type: redirect`** requires **`redirect.url`** and forbids populated **`service`** / **`handler`** fields).

Validation errors cite the rule index, configured host pattern, and routing path: **`rules[N] host="..." path="..."`**. Rule-level backends use **`path="/"`**; path backends use the configured **`paths[].path`** pattern (if that pattern string is empty, the message falls back to `paths[index]`). **Fallback** backends use **`fallback path="/"`**.

## Reloading Configuration

You can reload the configuration without restarting the server:

1. Send a SIGHUP signal: `kill -HUP $(cat /tmp/gozoox.ingress.pid)`
2. Use the reload command: `ingress reload`

The server will reload the configuration file and apply changes without dropping connections.
