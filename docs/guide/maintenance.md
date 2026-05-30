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
    - app.example.com
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

Each entry is either a plain hostname / wildcard string or an object:

| Field | Description |
|-------|-------------|
| `host` | Host pattern (exact, `*` wildcard, or Go regex — same inference as route `host`) |
| `window.start` | Optional RFC3339; omit for no lower bound |
| `window.end` | Optional RFC3339; omit for no upper bound |

When **no `window`** is set (or both sides are empty), the host entry is **always active** once the hostname matches.

Times use **RFC3339** (e.g. `2026-05-30T02:00:00+08:00`). Validation rejects `end` before `start`.

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
| `hosts` | Required when `scope: listed`; same shape as global `maintenance.hosts` | — |
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

## Response and logging

- Status **503** with HTML error page (or JSON when `Accept` prefers JSON).
- Optional **`Retry-After`** when `retry_after` > 0.
- Access log appends **`maintenance_block=1`**.

## Admin console

When **`admin.enabled: true`**, use the **维护** section for global maintenance and the route editor for per-rule maintenance.

## Examples

Runnable sample: [`examples/maintenance/ingress.yaml`](https://github.com/go-zoox/ingress/tree/master/examples/maintenance). Field tables: [Configuration reference](./configuration.md#maintenance-maintenance).

```bash
ingress validate -c examples/maintenance/ingress.yaml
```
