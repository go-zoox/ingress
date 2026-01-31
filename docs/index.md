---
layout: home

hero:
  name: Ingress
  text: Reverse Proxy
  tagline: An Easy, Powerful, Flexible Reverse Proxy
  actions:
    - theme: brand
      text: Get Started
      link: /guide/getting-started
    - theme: alt
      text: View on GitHub
      link: https://github.com/go-zoox/ingress

features:
  - icon: 🚀
    title: Easy to Use
    details: Simple configuration with YAML files. Get started in minutes with minimal setup.
  - icon: 🔒
    title: Secure
    details: Built-in authentication support (Basic, Bearer, JWT, OAuth2, OIDC) and SSL/TLS termination.
  - icon: ⚡
    title: High Performance
    details: Efficient routing with caching support (in-memory or Redis) for optimal performance.
  - icon: 🎯
    title: Flexible Routing
    details: Support for exact, regex, and wildcard host matching with path-based routing.
  - icon: 🏥
    title: Health Checks
    details: Built-in health check support for both outer and inner service monitoring.
  - icon: 🔄
    title: Request Rewriting
    details: Flexible request and response rewriting for headers, paths, and query parameters.

---

## Quick Start

Install Ingress:

```bash
go install github.com/go-zoox/ingress@latest
```

Start the server:

```bash
# Start with default configuration (port 8080)
ingress run

# Start with custom configuration file
ingress run -c ingress.yaml
```

Basic configuration example:

```yaml
version: v1
port: 8080

rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
```

For more details, see the [Getting Started Guide](/guide/getting-started).
