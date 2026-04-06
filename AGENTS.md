# Agent notes (go-zoox/ingress)

Context for humans and coding agents working on this repository.

## Host routing

- **Compile path**: `core/prepare.go` calls `compileRouterIndex` in `core/compile.go` on startup and on config reload. Host and path patterns are compiled once (Go `regexp`); invalid patterns fail startup/reload.
- **`host_type` resolution**: If `host_type` is omitted or set to `auto`, the effective type is chosen from the `host` string at compile time and **written back** to `rule.Rule.HostType`. Downstream code (`core/match.go`, `core/build.go`, service name templates) relies on that final value, not only on the YAML omission.
- **Inference order** (when auto):
  1. If `host` contains regexp metacharacters `( ) [ ] ^ $ | + ? \` → **regex**
  2. Else if `host` contains `*` → **wildcard**
  3. Else → **exact**
- Regex is checked before `*` so patterns like `^.*\.example\.com$` are not misclassified as wildcard.
- **Explicit `host_type: exact`**: Disables inference; `host` is matched as a literal string even if it looks like a pattern (rare).

## Common pitfalls

- **`\w` in Go regex** matches `[0-9A-Za-z_]`, not `-`. Subdomains such as `custom-domain.inlets.example.com` will not match `(\w+).inlets.example.com`; use something like `([a-zA-Z0-9-]+)` or a wildcard host if appropriate.
- **Omitting `host_type` is not “always exact”**: It is “auto”. Plain hostnames (letters, digits, dots, hyphens only, no `*` or regex metacharacters) still resolve to **exact**.

## HTTP/2 and HTTP/3 (zoox)

Ingress runs on [github.com/go-zoox/zoox](https://github.com/go-zoox/zoox). Protocol features are implemented there; ingress **maps YAML → `zoox.Application.Config`** in `core/build.go`.

- **HTTP/2 over TLS**: When `https.port` is set and TLS is available (files or SNI loader), zoox configures the HTTPS server with HTTP/2 (ALPN `h2`). No separate ingress flag.
- **`enable_h2c`**: Cleartext HTTP/2 on the plaintext HTTP listener (`port`). Unsafe on untrusted networks; use only behind a trusted LB or for local testing.
- **HTTP/3**: Under `https:` — `enable_http3`, optional `http3_port` (default same as `https.port`), optional `http3_altsvc_max_age` (`Alt-Svc` `ma=` seconds; `0` uses framework default; negative disables the header). Requires TCP HTTPS and TLS; opens a UDP listener for QUIC.

Zoox may also honor env overrides when unset in config: `ENABLE_H2C`, `ENABLE_HTTP3`, `HTTP3_PORT`, `HTTP3_ALTSVC_MAX_AGE` (see zoox `BuiltInEnv*` in `application.go` / `constants.go`).

## Access logging

- Access logs are emitted in `core/build.go` (handler branch and upstream proxy branch), and now share `buildAccessLogExtraFields`.
- Keep existing leading fields stable (`host`, `target`, request line, status, duration) for backwards-compatible log parsing; append new fields at the end.
- Added extra fields map roughly to common Nginx variables: `referer`, `ua`, `xff`, `tls_protocol`, `tls_cipher`, `upstream_status`, `upstream_response_length`, `upstream_response_time`.
- For missing values, use `-` (or `-1` for unknown upstream content length) to keep logs structurally predictable.
- TLS names are sourced from Go stdlib (`tls.VersionName`, `tls.CipherSuiteName`), so expected protocol strings are like `TLS 1.3` (not `TLSv1.3`).

## Docs and tests

- User-facing behavior: `docs/guide/routing.md` (EN), `docs/zh/guide/routing.md` (ZH), TLS and HTTP/2–3 in `docs/guide/ssl-tls.md` / `docs/zh/guide/ssl-tls.md`, routing/config snippets in `docs/guide/configuration.md` / `docs/zh/guide/configuration.md`, and access-log field notes in those same configuration docs.
- Inference and compile behavior: `core/compile_test.go`, `core/compile.go` (`effectiveHostType`, `hostLooksLikeRegexp`).
- Protocol wiring and logging: `core/build.go`, `core/build_test.go` (`TestBuild_HTTP2HTTP3ZooxConfig`, `TestBuild_AccessLogExtraFields_WithTLS`, `TestBuild_AccessLogExtraFields_WithoutTLS`).

## Verification

- From repo root: `go test ./core/...` (or narrow with `-run`). If the environment cannot reach the module proxy, try `GOPROXY=off` when modules are already cached.
