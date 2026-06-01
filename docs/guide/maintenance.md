# Maintenance mode

Ingress can return **503 Service Unavailable** for matched traffic during planned downtime. Maintenance is evaluated **after route match and WAF**, and **before** redirects, handlers, or upstream proxies.

Two layers work together:

1. **Global `maintenance:`** — host registry with optional per-host time windows and default 503 copy / bypass rules.
2. **Route `rules[].backend.service.maintenance`** — host-level service backend only; `scope: all` (every host on the rule) or `scope: listed` (only hosts in `maintenance.hosts` on that rule).

Either layer can trigger maintenance for a request. When both apply, **route-level `title` / `subtitle` / `retry_after` override** the global defaults; **bypass rules are merged** (global + route).

## Global maintenance

```yaml
maintenance:
  hosts:
    - host: app.example.com
    - host: staging-*.example.com
      window:
        start: "2026-05-30T02:00:00+08:00"
        end: "2026-05-30T06:00:00+08:00"
  retry_after: 3600
  title: Planned maintenance
  subtitle: We will be back shortly.
  bypass:
    allow_ips:
      - 10.0.0.0/8
    paths:
      - /healthz
      - /metrics/*
    header:
      name: X-Maintenance-Bypass
      value: secret-token
```

### `maintenance.hosts[]`

Each entry is a `{ host, window }` object (`host` and **`window`** with **`start`** and **`end`** are required). Maintenance applies only while `now` is inside that window.

Times use **RFC3339** (e.g. `2026-05-30T02:00:00+08:00`). Validation rejects `end` before `start`.

| Field | Description |
|-------|-------------|
| `host` | Host pattern (exact, `*` wildcard, or Go regex — same inference as route `host`) |
| `window.start` | Required RFC3339 maintenance start |
| `window.end` | Required RFC3339 maintenance end |

## Route-level maintenance

Only on **`rules[].backend.service`** (not path backends or fallback). Requires **`backend.type: service`** (or inferred service backend).

```yaml
rules:
  - host: "*.example.com"
    backend:
      type: service
      service:
        name: backend.internal
        port: 8080
        maintenance:
          enabled: true
          scope: listed          # all | listed (default: all)
          hosts:
            - host: legacy.example.com
              window:
                start: "2026-05-31T00:00:00+08:00"
                end: "2026-05-31T01:00:00+08:00"
          title: Legacy stack maintenance
          retry_after: 1800
          bypass:
            paths:
              - /healthz
```

| Field | Description | Default |
|-------|-------------|---------|
| `enabled` | Turn on route maintenance | `false` |
| `scope` | `all` — every host matched by the rule; `listed` — only `hosts[]` | `all` |
| `hosts` | Required when `scope: listed`; same shape as global `maintenance.hosts` (each with `window`) | — |
| `window` | Required when `scope: all` and `enabled: true` (rule-level maintenance window) | — |
| `retry_after` | `Retry-After` response header (seconds) | `0` (omit header) |
| `title` / `subtitle` | Override built-in 503 heading / message | built-in copy |
| `bypass` | Same as global bypass | — |

`scope: all` must **not** include `hosts`. `scope: listed` **requires** at least one host entry.

## Bypass

While maintenance is active for a request, bypass allows it through to the normal backend flow:

| Bypass | Semantics |
|--------|-----------|
| `allow_ips` | Client IP or CIDR (uses `RemoteAddr`; `X-Forwarded-For` leftmost when resolving IP for bypass) |
| `paths` | Exact path, or trailing `*` prefix match (e.g. `/metrics/*` → prefix `/metrics/`) |
| `header` | Exact header name/value match |

Global and route bypass entries are **unioned**.

### Maintenance response header (`response_header`)

Sent on maintenance **503** responses and on **`GET /_/ingress/status`** when the Host is in maintenance.

| Field | Description | Default |
|-------|-------------|---------|
| `name` | Response header name | `X-Ingress-Maintenance` |
| `value` | Response header value | `1` |

Omit the block entirely to use defaults. Set only `name` or only `value` to override one side; the other keeps its default. Route-level `response_header` overrides global when route maintenance triggers.

### Maintenance window headers

When the matched `hosts[]` entry has a **`window`**, ingress also sends (on maintenance **503** and **`GET /_/ingress/status`**):

| Header | When set |
|--------|----------|
| `X-Ingress-Maintenance-From` | `window.start` is set (RFC3339) |
| `X-Ingress-Maintenance-Until` | `window.end` is set (RFC3339) |

Route-level listed hosts take precedence over global `hosts[]` for these values; `scope: all` with no per-host window falls back to the global matched entry’s window if any.

## Distinguishing maintenance 503 vs upstream 503

Both may return HTTP **503**, but they come from different stages:

| Signal | Maintenance 503 | Upstream 503 |
|--------|-----------------|--------------|
| Response header | **`X-Ingress-Maintenance: 1`** (+ optional **`X-Ingress-Maintenance-From` / `-Until`** when a host `window` is configured) | _(absent)_ |
| Access log | **`maintenance_block=1`**, `upstream_response_length=-1` | `maintenance_block=0`, real upstream length/RTT |
| Body | Ingress error page (custom `title` / `subtitle`) | Upstream response body |
| Upstream contacted | **No** (short-circuited before proxy) | **Yes** |

Use **`GET /_/ingress/status`** for load-balancer / monitoring probes (host-level maintenance state, ignores path bypass).

## Response and logging

- Status **503** with HTML error page (or JSON when `Accept` prefers JSON).
- Response header **`X-Ingress-Maintenance: 1`** by default (`maintenance.response_header` / `service.maintenance.response_header` to customize).
- Optional **`X-Ingress-Maintenance-From`** / **`X-Ingress-Maintenance-Until`** when the active host entry defines `window.start` / `window.end`.
- Optional **`Retry-After`** when `retry_after` > 0.
- Access log appends **`maintenance_block=1`**.

## Ingress status probe

`GET /_/ingress/status` by default — handled **before** routing, WAF, and maintenance bypass. Override with **`maintenance.status_path`** (must start with `/`).

| Condition | HTTP | JSON `status` | Maintenance header |
|-----------|------|---------------|----------------------|
| Host not in maintenance | `200` | `"ok"` | _(absent)_ |
| Host in maintenance | `503` | `"maintenance"` (+ optional fields) | configured (default `X-Ingress-Maintenance: 1`; window headers when applicable) |

JSON maintenance responses include `maintenance_header_name`, `maintenance_header_value`, and when a host `window` is active: `maintenance_from`, `maintenance_until` (RFC3339, same as the `X-Ingress-Maintenance-*` headers).

Customize the JSON body with **`maintenance.status_response`** (`ok` / `maintenance` templates). Placeholders: `${host}`, `${title}`, `${subtitle}`, `${retry_after}` (bare number), `${maintenance_header_name}`, `${maintenance_header_value}`, `${maintenance_from}`, `${maintenance_until}`, `${status}` (`ok` | `maintenance`). String placeholders expand inside JSON quotes; omit a template to keep the built-in body for that state.

```yaml
maintenance:
  status_response:
    ok: '{"ready":true,"host":"${host}"}'
    maintenance: '{"ready":false,"message":"${title}","retry_after":${retry_after}}'
    content_type: application/json; charset=utf-8
```

Example (default path):

```bash
curl -sS -D - http://app.example.com/_/ingress/status
```

Custom path:

```yaml
maintenance:
  status_path: /internal/ingress-status
```

## Admin console

When **`admin.enabled: true`**, use the **维护** section for global maintenance and the route editor for per-rule maintenance.

## Examples

Runnable samples: [`examples/maintenance/`](https://github.com/go-zoox/ingress/tree/master/examples/maintenance) (see [Maintenance examples](../examples/maintenance.md)). Field tables: [Configuration reference](./configuration.md#maintenance-maintenance).

```bash
ingress validate -c examples/maintenance/global-always-on.yaml
```
