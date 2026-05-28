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
- **Upstream Host mode**: Prefer **`backend.service.mode`** (`internal` / `external`). Legacy **`backend.mode`** is an alias when `service.mode` is empty; both must not disagree. `internal` (default) keeps the client `Host` unless `service.request.host.rewrite` is set. `external` sets `Host` to the upstream (`service.Host()`). Explicit `request.host.rewrite` wins. Fallback uses `host: @@fallback` (`core/host_rewrite.go`).
- **Path strip prefix**: On **`paths[].backend.service` only**, `strip_prefix: true` expands at load time to `request.path.rewrites` using that path’s `paths[].path` (see `core/strip_prefix.go`). Cannot combine with explicit `request.path.rewrites` on the same backend.
- **Default upstream port**: `backend.service.port` may be omitted (`0`). `core/service/host.go` then uses **443** when `protocol` is **`https`** and **80** when **`http`** (or default/empty). First `Host()` / `Target()` call may write the chosen port back onto the loaded `Service` value.
- **`service.protocol` omission**: Unset or empty defaults to **`http`** (`protocol,default=http` on `core/service.Service` and the same default in `Host()` / `Target()`).

## Common pitfalls

- **`\w` in Go regex** matches `[0-9A-Za-z_]`, not `-`. Subdomains such as `custom-domain.inlets.example.com` will not match `(\w+).inlets.example.com`; use something like `([a-zA-Z0-9-]+)` or a wildcard host if appropriate.
- **Omitting `host_type` is not “always exact”**: It is “auto”. Plain hostnames (letters, digits, dots, hyphens only, no `*` or regex metacharacters) still resolve to **exact**.

## HTTP/2 and HTTP/3 (zoox)

Ingress runs on [github.com/go-zoox/zoox](https://github.com/go-zoox/zoox). Protocol features are implemented there; ingress **maps YAML → `zoox.Application.Config`** in `core/build.go`.

- **HTTP/2 over TLS**: When `https.port` is set and TLS is available (files or SNI loader), zoox configures the HTTPS server with HTTP/2 (ALPN `h2`). No separate ingress flag.
- **`enable_h2c`**: Cleartext HTTP/2 on the plaintext HTTP listener (`port`). Unsafe on untrusted networks; use only behind a trusted LB or for local testing.
- **HTTP/3**: Under `https:` — `enable_http3`, optional `http3_port` (default same as `https.port`), optional `http3_altsvc_max_age` (`Alt-Svc` `ma=` seconds; `0` uses framework default; negative disables the header). Requires TCP HTTPS and TLS; opens a UDP listener for QUIC.

Zoox may also honor env overrides when unset in config: `ENABLE_H2C`, `ENABLE_HTTP3`, `HTTP3_PORT`, `HTTP3_ALTSVC_MAX_AGE` (see zoox `BuiltInEnv*` in `application.go` / `constants.go`).

## HTTP -> HTTPS redirect behavior

- Global redirect is configured under `https.redirect_from_http` in `core/config.go`.
- **Default behavior**: when `https.port` is set, forced HTTP -> HTTPS redirect is **not** active unless `https.redirect_from_http.enabled: true`.
- `permanent: true` returns 301; `permanent: false` returns 302, unless **`with_origin_method_and_body: true`** uses **308**/**307** instead (**302**/**301** when false).
- `exclude_paths` uses exact path matching and skips forced redirect for matched paths.
- Redirect is decided before route matching in `core/build.go` (`shouldRedirectFromHTTP`), while route-level redirect (`rules[].backend.redirect`) still applies in normal route flow.
- HTTPS detection checks both TLS and `X-Forwarded-Proto: https` to avoid redirect loops behind trusted proxies/LBs.

## Logging (`logging` in ingress.yaml = zoox `Config.Logger`)

- Ingress **`logging`** decodes as **`zcfg.Logger`** and is copied to **`app.Config.Logger`** in `prepare()` when `level` or `transports` is set. Zoox **`Logger()`** builds console + transports (`components/application/logger`).
- Use **`logging.transports`** with `type: file`, `path`, `levels` (same as zoox). No separate ingress-only logging types.
- Relative **`path`** / **`levels.*`** values resolve against the **ingress config file directory** (`core.ResolveConfigPaths` in `run` / `validate`), not the process cwd.

## Access logging

- Access logs use `formatAccessLog` in `core/accesslog.go`, emitted from `core/build.go`.
- Format: `{client_ip} {host} -> {target} "{method} {path} {proto}" {status} {duration_ms} cache_hit=… waf_block=… real_ip=… referer=… ua=… xff=… tls_* upstream_*`.
- WAF blocks, redirects, handler, and upstream proxy share the same line shape.

## HTTP response cache (`backend.cache`)

- **Where**: `core/build.go` — **service** after auth/DNS (cache hit short-circuits before `go-zoox/proxy`); **handler** wraps `ctx.Writer` with `zooxHTTPCacheCaptureRW` when storing; **redirect** tries cache before `applyRedirect` and stores the final `Location` after expansion (GET only). Validate accepts `cache.enabled` for **service**, **handler**, and **redirect** (`core/validate.go`).
- **Defaults**: off without `cache.enabled: true`. Applied in `normalizeHTTPCache` (`core/http_cache.go`): TTL 300s, max body 2MiB, fingerprint `md5`, methods `GET`+`HEAD`, key headers `Authorization` + `Cookie` + `Accept-Encoding`, client bypass tokens `no-cache` / `no-store` / `max-age=0`, `Pragma: no-cache` honored by default, skip responses with **`Set-Cookie`** by default (`skip_when_set_cookie`). **`skip_vary`** default false: non-empty origin **`Vary`** ⇒ do not store; when **`skip_vary`**, **`Vary` is omitted** from stored JSON and stripped on hits.
- **Per-path rules (`backend.cache.paths`)**: Optional ordered list; first match wins (`action: cache` or `bypass`). Unmatched paths use **`default`** (`cache` or `bypass`; default `cache`). Per-rule **`ttl`** / **`max_body_bytes`** override backend defaults. Compiled at validate/prepare (`compileBackendCachePathRules`); gate via `httpCachePolicyForRequest` in `core/build.go`. **`match_type: auto`** mirrors host inference (regex metacharacters → regex; trailing `/` → prefix; else exact).
- **POST + JSON cache keys**: On a path rule, `methods: [POST]` and `key_json: [dot.paths]` (e.g. `product.id`) fingerprint selected JSON fields; request body is cloned and replayed to upstream (`readAndReplayRequestBody`). Global `cache.methods` must not include POST. Keys use prefix **`httpcache:v2:`** when `key_json` is set; missing fields / non-JSON / `{}` skip cache. Example: `examples/advanced/http-response-cache-post-json.yaml`.
- **Upstream Host mode**: Prefer **`backend.service.mode`** (`internal` / `external`); **`backend.mode`** is a legacy alias. Both may not be set to different values. See `effectiveBackendMode` / `effectiveHostRewrite` (`core/host_rewrite.go`).
- **Keys**: prefix `httpcache:v1:` (under the global `cache.prefix` when using Redis). Canonical string treats **HEAD method as GET** for fingerprinting so both can share an entry.
- **Store**: **GET** only for population: proxy upstream in `OnResponse`; handler after executing handler; redirect after final URL is known. Handler uses an optional body capture buffer; other methods may still **hit** cache (e.g. HEAD shares GET key).
- **Logs**: hits append `cache_hit=1` to the access log line (service proxy, handler, and redirect).

Separate from matcher KV: top-level `cache` still configures the shared `ctx.Cache()` backend (`core/prepare.go`, `core/match.go` uses `match.host:v2:` keys).

## WAF (layer-7 guard, v1)

- **When**: After a route match in `core/build.go`, before `backend.redirect`, handler, or upstream proxy (`waf.CheckRequest` + `*waf.Profile`).
- **Package**: `core/waf/` — `CompileIngress`, `CheckRequest`, `MergePatch` / `MergeRules`, `StarterRules`, `ApplyRulePatchesFromFile` / `ApplyRulePatchesFromYAML`.
- **Config**: Typed global `waf` on `core.Config` (`rule.WAF` — no nested pointers; `go-zoox/config` cannot decode them). Per-route **`rules[].waf`** maps merge over the baseline via **`waf.ApplyRulePatchesFromFile`** (called from `cmd/ingress/run.go` and `validate.go` right after `config.Load`). In-memory `cfg` uses **`rule.Rule.WAFPatch`** (`config:"-"`).
- **Semantics**: IP deny list, optional allow gate, then regex/contains signatures (optional starters from `StarterRules()`; disable via `disable_builtin`). Global/per-rule **`log_only`** audits without blocking (`[waf audit]` vs `[waf block]` in logs). No HTTP body scanning in v1.
- **Tests / examples**: `core/waf/compile_test.go`, `eval_test.go`, `patch_test.go`, `yaml_test.go`; `examples/waf/`.

## Redirect and config validation

- Route redirect (`rules[].backend.redirect` and path backends): evaluated before proxy/handler in `core/build.go`. **`backend.type`** is **`service`**, **`handler`**, or **`redirect`** (`core/constants.go`). **`inferBackendTypes` / `inferRuleBackends`** (`core/backend_type.go`) run during **`prepare`** and **`ingress validate`**, inferring the type when `type` is omitted and exactly one of service/handler/redirect blocks looks configured; otherwise validation demands an explicit `backend.type`. Each typed backend permits only its matching block (`core/validate.go`). **`expandRedirectURL`** (`core/match.go`) applies `${host.N}`/`${path.N}`/`$1`-style templates in redirect URLs (aligned with service naming). Route **`redirect.with_origin_method_and_body`** mirrors global semantics (**307**/**308** vs **302**/**301**).

## Docs and tests

- Runnable YAML samples live under repo-root `examples/` (topic subdirs); `docs/examples/` and `docs/zh/examples/` embed them via VitePress 1 snippet lines `<<< @/../examples/...` (path only; optional `{yaml}` in braces—do not add a trailing space + `yaml`, it is parsed as part of the filename).
- User-facing behavior: `docs/guide/routing.md` (EN), `docs/zh/guide/routing.md` (ZH), WAF in `docs/guide/waf.md` / `docs/zh/guide/waf.md`, TLS and HTTP/2–3 in `docs/guide/ssl-tls.md` / `docs/zh/guide/ssl-tls.md`, routing/config snippets in `docs/guide/configuration.md` / `docs/zh/guide/configuration.md`, and access-log field notes in those same configuration docs.
- Inference and compile behavior: `core/compile_test.go`, `core/compile.go` (`effectiveHostType`, `hostLooksLikeRegexp`).
- Config validation (`ingress validate`): `core/validate.go`, `core/validate_test.go`.
- Redirect and auth/header constants behavior: `core/build.go`, `core/constants.go`, `core/build_test.go`.
- Protocol wiring and logging: `core/build.go`, `core/build_test.go` (`TestBuild_HTTP2HTTP3ZooxConfig`, `TestBuild_AccessLogExtraFields_WithTLS`, `TestBuild_AccessLogExtraFields_WithoutTLS`).

## Admin console (`core/admin/`)

- **Config**: `admin:` section in `ingress.yaml` (`admin.enabled`, `port`, `database`, `web`).
- **Stack**: `core/admin/web` — React + TypeScript + Vite + pnpm; `core/admin` — go-zoox HTTP API, gormx + SQLite (`audit_log`, `waf_event`, `config_revision`).
- **Ingress integration**: starts with `ingress run` when `admin.enabled: true`; reads/writes the same ingress config file; `POST /api/v1/reload` validates then in-process reload. Route list / dry-run match use `core.ListRouteRows` and `core.PreviewMatch` (`core/admininspect.go`).
- **Dev**: `ingress run -c examples/admin-console/ingress.yaml` + `cd core/admin/web && pnpm dev` (proxy `/api`). **Build**: `cd core/admin && make build`. Demo config: `examples/admin-console/`.
- **Logs API**: `GET /api/v1/logs` supports `offset` (byte tail), `cache_hit`, `waf_block` filters for live monitoring.
- **Cache / TLS API**: `GET /api/v1/cache/overview`, `GET /api/v1/tls/certs` (x509 metadata from cert files).
- **Static prototype** (no backend): `prototypes/admin-console/`.

## Verification

- From repo root: `go test ./core/...` (or narrow with `-run`). If the environment cannot reach the module proxy, try `GOPROXY=off` when modules are already cached.
- Admin UI: `make -C core/admin web` then `go build -tags adminui ./cmd/ingress` (GoReleaser/Dockerfile run `make web` automatically; `core/admin/static/dist` is gitignored). Plain `go build` embeds a small API-only stub.
