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

## Docs and tests

- User-facing behavior: `docs/guide/routing.md` (EN), `docs/zh/guide/routing.md` (ZH), and rule snippets in `docs/guide/configuration.md` / `docs/zh/guide/configuration.md`.
- Inference and compile behavior: `core/compile_test.go`, `core/compile.go` (`effectiveHostType`, `hostLooksLikeRegexp`).

## Verification

- From repo root: `go test ./core/...` (or narrow with `-run`). If the environment cannot reach the module proxy, try `GOPROXY=off` when modules are already cached.
