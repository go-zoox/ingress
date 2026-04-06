# Ingress - A Easy, Powerful, Fexible Reverse Proxy

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
# start ingress, cached in memory, default udp port: 80
ingress run

# start ingress with config (see conf/ingress.yml for more options)
ingress run -c ingress.yml
```

## Configuration
See the [configuration file](conf/ingress.yml).

## Features

### ✅ Currently Implemented

- **Flexible Routing**: Exact, regex, and wildcard host matching (with optional automatic `host_type` inference at compile time) and path-based routing
- **Request/Response Rewriting**: Path, header, and query parameter modification
- **Authentication**: Basic Auth, Bearer Token (JWT/OAuth2/OIDC in progress)
- **SSL/TLS**: HTTPS and SSL certificate configuration
- **Health Checks**: Outer and inner service health monitoring
- **Caching**: In-memory and Redis caching support
- **Redirects**: URL redirection with permanent/temporary options
- **Timeout Control**: Request timeout and delay configuration
- **Fallback Service**: Default backend for unmatched requests

### 🚧 Roadmap

We have identified key features needed to make Ingress a production-ready reverse proxy. See our [TODO List](docs/TODO.md) for detailed roadmap.

**High Priority (P0)**:
- 🔴 Load Balancing (multiple backends, algorithms, health checks)
- 🔴 Rate Limiting (IP/user/path-based, token bucket algorithm)
- 🔴 Access Control (IP whitelist/blacklist, CORS, request size limits)
- 🔴 Service Governance (circuit breaker, retry, fallback)
- 🔴 Traffic Management (canary deployment, A/B testing, traffic mirroring)
- 🔴 Hot Reload Optimization (zero-downtime config reload)

**Medium Priority (P1)**:
- 🟡 Compression (Gzip/Brotli)
- 🟡 WebSocket Support
- 🟡 gRPC Support
- 🟡 Observability (structured logging, metrics, tracing)
- 🟡 WAF (Web Application Firewall)
- 🟡 Service Discovery (DNS, Kubernetes, Consul, etc.)
- 🟡 Connection Management
- 🟡 Advanced Authentication (JWT/OAuth2/OIDC completion)

**Low Priority (P2)**:
- 🔴 Protocol Conversion (HTTP to gRPC/Dubbo)
- 🔴 Request/Response Body Modification
- 🔴 Other enhancements

For complete details, see [TODO.md](docs/TODO.md).

## Documentation

- [AGENTS.md](AGENTS.md) — notes for contributors and AI agents (routing compile behavior, pitfalls)
- [Getting Started](docs/guide/getting-started.md)
- [Configuration Reference](docs/guide/configuration.md)
- [Routing Guide](docs/guide/routing.md)
- [Authentication Guide](docs/guide/authentication.md)
- [SSL/TLS Guide](docs/guide/ssl-tls.md)
- [Health Checks](docs/guide/health-checks.md)
- [Caching](docs/guide/caching.md)
- [Rewriting](docs/guide/rewriting.md)
- [TODO List](docs/TODO.md)

## License
GoZoox is released under the [MIT License](./LICENSE).
