# Ingress — An Easy, Powerful, Flexible Reverse Proxy

[![PkgGoDev](https://pkg.go.dev/badge/github.com/go-zoox/ingress)](https://pkg.go.dev/github.com/go-zoox/ingress)
[![Build Status](https://github.com/go-zoox/ingress/actions/workflows/release.yml/badge.svg?branch=master)](https://github.com/go-zoox/ingress/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/go-zoox/ingress)](https://goreportcard.com/report/github.com/go-zoox/ingress)
[![Coverage Status](https://coveralls.io/repos/github/go-zoox/ingress/badge.svg?branch=master)](https://coveralls.io/github/go-zoox/ingress?branch=master)
[![GitHub issues](https://img.shields.io/github/issues/go-zoox/ingress.svg)](https://github.com/go-zoox/ingress/issues)
[![Release](https://img.shields.io/github/tag/go-zoox/ingress.svg?label=Release)](https://github.com/go-zoox/ingress/tags)

## Installation

To install the package, run:
```bash
go install github.com/go-zoox/ingress@latest
```

## Quick Start

```bash
# start ingress with in-memory routing state; plaintext HTTP port comes from config / examples
ingress run

# validate a config before running
ingress validate -c examples/basic/ingress.yaml

# start with a repo example
ingress run -c examples/basic/ingress.yaml

# WAF audit-only sample (blocks nothing; logs hits)
ingress run -c examples/waf/log-only-audit.yaml

# reload running instance after editing config (validates then SIGHUP)
ingress reload -c /path/to/ingress.yaml

# operations admin console (API + web UI; see admin/README.md)
ingress admin -c examples/admin-console/admin.yaml
```

## Configuration

Runnable samples live under [`examples/`](examples/) (validate with `ingress validate -c examples/<topic>/...`). Several YAML files pair explicit **`backend.type`** with omission side by side—see **`examples/basic/ingress.yaml`**, **`examples/ssl-tls/route-redirect.yaml`**, **`examples/redirect/capture-and-mixed.yaml`**, **`examples/waf/`** for first-generation WAF. Field-level reference: [`docs/guide/configuration.md`](docs/guide/configuration.md) (Chinese: [`docs/zh/guide/configuration.md`](docs/zh/guide/configuration.md)).

## Features

### ✅ Currently Implemented

- **Flexible Routing**: Exact, regex, and wildcard host matching (with optional automatic `host_type` inference at compile time) and path-based routing
- **Request/Response Rewriting**: Path, header, and query parameter modification
- **Authentication**: Basic Auth, Bearer Token (JWT/OAuth2/OIDC in progress)
- **SSL/TLS**: HTTPS and certificate configuration; **HTTP/2** over TLS (ALPN `h2`); optional **HTTP/3 (QUIC)** and cleartext **h2c** — see [SSL/TLS guide](docs/guide/ssl-tls.md)
- **Health Checks**: Outer and inner service health monitoring
- **Caching**: In-memory and Redis caching support
- **Redirects**: Global HTTP→HTTPS (`https.redirect_from_http`), per-route **`backend.redirect`** with optional **307/308** (`with_origin_method_and_body`) and URL capture templates; **`backend.type`** (**`service`**, **`handler`**, **`redirect`**) is optional and inferred when unambiguous—set it explicitly only when validation reports ambiguity.
- **WAF (v1)**: Layer-7 guard after route match—IP deny/allow lists (optional `trust_proxy` + `X-Forwarded-For` / `xff_index`), regex/contains signatures on path, query, URI, headers, or single `header:Name`; optional built-in starters; global and per-route `rules[].waf` YAML map merge; `log_only` audit mode. No request-body scanning in v1.
- **Config reload**: `ingress reload` runs **`ingress validate`** on the config file, then signals the running process (**SIGHUP**) to reload; the server also reloads on **SIGHUP** when started with **`ingress run`** (same config path). Operational niceties still missing: guaranteed zero‑downtime handoff, rollback, REST dynamic config API.
- **Timeout Control**: Request timeout and delay configuration
- **Fallback Service**: Default backend for unmatched requests (ingress-level fallback; distinct from circuit-breaker fallback in roadmap)
- **Access logging**: Text access logs with extended fields (**`real_ip`**, **`referer`**, **`xff`**, TLS protocol/cipher, upstream status/time, etc.) — see [configuration](docs/guide/configuration.md)

### 🚧 Roadmap

We have identified key features needed to make Ingress a production-ready reverse proxy. See our [TODO List](docs/TODO.md) for detailed roadmap. Some items below build on **partial** capabilities already listed under *Currently Implemented*.

**High Priority (P0)**:
- 🔴 Load Balancing (multiple backends, algorithms, upstream pools)
- 🔴 Rate Limiting (token bucket / fixed window — not covered by **WAF** v1 signatures)
- 🟡 Access Control — **WAF** already provides **IP** **`deny`/`allow`** (with **`trust_proxy` / `xff_index`**). Still open: CORS, request size caps, HTTP method ACLs, richer policy UX.
- 🔴 Service Governance (circuit breaker, upstream retry/backoff distinct from timeouts)
- 🔴 Traffic Management (canary, A/B, mirroring)

**Incremental on top of current reload (also P0 in TODO)**:
- 🟡 Hot Reload — **`ingress reload`** + **SIGHUP** reload with config re‑**prepare** exist; **`ingress reload`** validates before signaling. Missing: hardened zero‑downtime guarantees, rollback, versioning, REST config API.

**Medium Priority (P1)**:
- 🟡 Compression (Gzip/Brotli) — stack may carry related deps; first-class YAML knobs and response path still TBD
- 🟡 WebSocket (explicit upgrade/proxy tuning and docs)
- 🟡 gRPC — **HTTP/2** is already negotiated on the **HTTPS** listener; dedicated gRPC proxy features (routing, health, LB) remain open
- 🟡 Observability — access logs already include **client / proxy / TLS / upstream** style fields; missing: JSON logs, Prometheus scrape, OpenTelemetry traces, RED-style metrics
- 🟡 Service Discovery (DNS/K8s/Consul dynamic backends)
- 🟡 Connection Management (tunable pools, keep-alive policy as first-class config)
- 🟡 Advanced Authentication (full JWT/OAuth2/OIDC completion beyond current basics)

**Low Priority (P2)**:
- 🔴 Protocol Conversion (HTTP to gRPC/Dubbo)
- 🔴 Request/Response Body Modification
- 🔴 Other enhancements

For complete details, see [TODO.md](docs/TODO.md).

## Documentation

- **Site (VitePress)**: [go-zoox.github.io/ingress](https://go-zoox.github.io/ingress/) — same guides as below, with search
- [AGENTS.md](AGENTS.md) — notes for contributors and AI agents (routing compile, HTTP→HTTPS redirect, WAF wiring, pitfalls)
- [Getting Started](docs/guide/getting-started.md)
- [Configuration Reference](docs/guide/configuration.md)
- [Routing Guide](docs/guide/routing.md)
- [WAF Guide](docs/guide/waf.md) (Chinese: [中文](docs/zh/guide/waf.md))
- [Authentication Guide](docs/guide/authentication.md)
- [SSL/TLS Guide](docs/guide/ssl-tls.md)
- [Health Checks](docs/guide/health-checks.md)
- [Caching](docs/guide/caching.md)
- [Rewriting](docs/guide/rewriting.md)
- [TODO List](docs/TODO.md)

## License

GoZoox is released under the [MIT License](./LICENSE).
