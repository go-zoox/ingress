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
      # type defaults to service, options: service | handler
      # type: service
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

```yaml
fallback:
  service:
    name: fallback-service
    port: 8080
    protocol: http              # http or https
    request:
      host:
        rewrite: true           # Rewrite Host header
```

### Rules Configuration

Rules define how requests are routed to backend services. See the [Routing Guide](/guide/routing) for detailed information.

```yaml
rules:
  - host: example.com           # Host to match
    # host_type: optional — omit or `auto` to infer exact vs regex vs wildcard from `host` at compile time
    # explicit values: exact | regex | wildcard
    backend:
      type: service             # Backend type: service (default) or handler
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
            rewrite: true       # Rewrite Host header
          path:
            rewrites:          # Path rewrite rules
              - ^/api/v1:/api/v2
          headers:             # Additional headers
            X-Custom-Header: value
          query:               # Query parameters
            key: value
          delay: 0              # Delay in milliseconds
          timeout: 30           # Timeout in seconds
      redirect:                 # Redirect configuration (alternative to service)
        url: https://example.com
        permanent: false
    paths:                      # Path-based routing (optional)
      - path: /api
        backend:
          service:
            name: api-service
            port: 8080
      - path: /healthz
        backend:
          type: handler
          handler:
            status_code: 200
            headers:
              Content-Type: application/json
            body: |
              {"ok": true}
```

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

Ingress validates the configuration file when it starts. If there are any errors, the server will not start and display the error message.

## Reloading Configuration

You can reload the configuration without restarting the server:

1. Send a SIGHUP signal: `kill -HUP $(cat /tmp/gozoox.ingress.pid)`
2. Use the reload command: `ingress reload`

The server will reload the configuration file and apply changes without dropping connections.
