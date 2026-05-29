# Security response headers

Ingress can attach **profile-based HTTP security headers** after a route match: HSTS, `X-Frame-Options`, `X-Content-Type-Options`, `Referrer-Policy`, `Content-Security-Policy`, and **CORS** (including **OPTIONS preflight**).

Configure a global baseline under **`security:`**, Host-level overrides under **`rules[].security`**, and Path-level overrides under **`paths[].security`**. Fields merge in order: global → Host → Path.

## Profiles

| Profile | Typical use | HSTS | Frame | CORS |
|---------|-------------|------|-------|------|
| `strict` | Web apps, admin UI | auto | DENY | off |
| `api` | REST / JSON APIs | auto | DENY | on (requires `cors.origins`) |
| `embeddable` | Pages allowed in iframes | auto | SAMEORIGIN | off |
| `off` | Disabled | — | — | — |

**HSTS `auto`** sends `Strict-Transport-Security` only when the request is HTTPS (direct TLS or `X-Forwarded-Proto: https`).

## Example

```yaml
security:
  profile: strict

rules:
  - host: api.example.com
    security:
      profile: api
      cors:
        origins:
          - https://portal.example.com
        credentials: true
    backend:
      service:
        name: api
        port: 8080
```

Runnable sample: [`examples/security/profiles.yaml`](../../examples/security/profiles.yaml).

## Fields

| Field | Description |
|-------|-------------|
| `profile` | `strict`, `api`, `embeddable`, or `off` |
| `hsts` | `auto` (default), `on`, or `off` |
| `frame` | `inherit`, `deny`, `sameorigin`, or `off` |
| `content_type_options` | `true` / `false` (nosniff) |
| `referrer_policy` | Header value; `off` disables |
| `csp` | CSP policy string; `off` disables |
| `cors.enabled` | Explicit on/off |
| `cors.origins` | Allowed origins (required when CORS is enabled) |
| `cors.methods` | Default: GET, POST, PUT, PATCH, DELETE, OPTIONS |
| `cors.headers` | Default: Authorization, Content-Type, Accept, X-Requested-With |
| `cors.credentials` | `Access-Control-Allow-Credentials` |
| `cors.max_age` | Preflight cache seconds (default 86400) |

## Precedence

- Security headers apply on **service**, **handler**, **redirect**, WAF block, rate-limit, and error responses for matched routes.
- **`backend.service.response.headers`** and **handler headers** are applied first; security headers are added unless the same key was already set.
- Unmatched routes (404) use the **global** `security:` profile only.

## Admin console

Edit global **安全** in the config modules panel; Host / Path overrides in the rule editor **安全** sidebar, or set `rules[].security` / `paths[].security` in YAML mode.

See also [Configuration reference](./configuration.md) and [Rewriting](./rewriting.md) for manual `response.headers`.
