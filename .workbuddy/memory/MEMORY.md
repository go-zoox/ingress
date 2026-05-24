# go-zoox/ingress — Long-term Memory

## Project Overview
Go-based reverse proxy/ingress controller with rule-based routing, auth, and admin console.

## Key Architecture Patterns
- **Auth struct**: Uses `*bool` with `IsEnabled()` for tri-state (nil=auto by Type, true=explicit enable, false=explicit disable)
- **OAuth2 flow**: Must follow go-zoox/oauth2 library pattern — all logic (session, redirect, cleanup) INSIDE `Authorize`/`Callback` closures (go-zoox/connect pattern). Use Go named returns for closure capture.
- **ValidateAuth chain**: OAuth2 validated first in `build.go` → then `ValidateAuth()` for basic/bearer. OAuth2 case returns nil (already validated upstream).
- **match.go**: All `service.Service{}` literals must include Auth, Mode, StripPrefix, HealthCheck fields (past bug source).
- **zoox context**: `ctx.Host()` (not `ctx.Hostname()` which drops port), `ctx.Query().Get("code").String()`, `Session().Get(key)` returns string.
- **JWT signing**: `go-zoox/jwt`, algorithm stored uppercase, normalize with `strings.ToUpper()`.

## Admin Console
- React + TypeScript frontend at `core/admin/web/`
- `configEntities.ts` handles form <-> YAML conversion
- `BackendFormFields.tsx` for form components
- `AuthFormFields.tsx` for auth-specific UI
- `HealthCheckFormFields.tsx` for healthcheck config UI
- `admininspect.go` provides route/match preview data with auth + healthcheck labels

## Auth Config Schema (YAML)
```yaml
auth:
  enabled: true          # *bool tri-state
  type: basic|bearer|oauth2
  basic:
    users:
      - username: x
        password: y
  bearer:
    tokens: ["t1", "t2"]
  oauth2:
    provider: github|gitlab|google|microsoft|feishu|slack|kakao|auth0|okta|doreamon
    client_id: "xxx"
    client_secret: "yyy"
    scopes: ["user:email"]  # optional, has provider defaults
    connect:
      enabled: true
      jwt:
        secret: "shared-secret"
        algorithm: hs256
        expires_in: "5m"
```

## User Preferences
- Prefers phased execution (plan first, implement after)
- Wants AI to provide options when uncertain, then seek confirmation
- Values following the plan exactly — gets frustrated when implementation deviates
- Reads Chinese, communicates in Chinese for this project

## Session History
- 2026-05-24: Auth bug fix, OAuth2 feature, auth.enabled, admin console auth UI, healthcheck UI, mobile responsive, ANSI log parse fix, LogPath->AccessLogPath rename, Overview flicker fix (11 commits)
