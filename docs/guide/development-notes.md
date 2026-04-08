# Development Notes

This page records implementation notes from recent ingress changes, mainly for future maintainers.

## 1) Global HTTP -> HTTPS Redirect

The redirect behavior is configured under `https.redirect_from_http`.

```yaml
https:
  port: 443
  redirect_from_http:
    disabled: false
    permanent: true
    exclude_paths:
      - /healthz
```

Key decisions:

- Redirect is evaluated before route matching.
- Redirect is enabled by default when `https.port` is set.
- `disabled: true` is the only explicit opt-out.
- Redirect keeps host/path/query by default.
- Requests already identified as HTTPS (`TLS` or `X-Forwarded-Proto: https`) are not redirected.
- `exclude_paths` uses exact path matching.

Why this shape:

- It avoids duplicating redirect rules per host.
- It keeps route-level `rules[].backend.redirect` available for per-route behavior.

## 2) Constant Extraction Rules

During refactors, avoid inline string literals for protocol/type/header selectors that are reused in logic branches.

Examples already extracted in `core/constants.go`:

- Host type selectors: `hostTypeExact`, `hostTypeRegex`, `hostTypeWildcard`, `hostTypeAuto`
- Backend type selectors: `backendTypeService`, `backendTypeHandler`
- Auth selectors/challenges: `authTypeBasic`, `authTypeBearer`, `authChallengeBasic`, `authChallengeBearer`
- Header and scheme: `headerXForwardedProto`, `headerWWWAuthenticate`, `schemeHTTP`, `schemeHTTPS`

Expected benefits:

- Fewer typo-driven bugs in branch conditions.
- Safer mechanical refactors.
- Clearer review diffs (semantic change vs text replacement).

## 3) Validation and Reload Safety

- `ingress validate` now reports errors by category:
  - `yaml syntax error ...`
  - `invalid config format ...`
  - `unsupported configuration ...`
- `ingress reload` validates config first and sends `SIGHUP` only when validation passes.

This keeps reload behavior consistent with startup safety and prevents applying broken config during runtime.
