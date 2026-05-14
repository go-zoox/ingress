# Advanced Examples

This page provides advanced configuration examples.

Sources: [`examples/advanced/`](https://github.com/go-zoox/ingress/tree/master/examples/advanced).

## Regex host matching with path rewriting

<<< @/../examples/advanced/regex-host-path.yaml

This example:

- Matches hosts like `t-myapp.example.work` using regex
- Routes using scoped captures in `service.name` (`${host.<index>}` and `${path.<index>}`)
- Rewrites paths for specific API routes

## Wildcard host matching

<<< @/../examples/advanced/wildcard.yaml

This matches any subdomain of `example.work`.

## Complex path rewriting

<<< @/../examples/advanced/complex-path-rewrite.yaml

## Health checks with multiple services

<<< @/../examples/advanced/health-checks.yaml

## Redis caching

<<< @/../examples/advanced/redis-cache.yaml

## Complete configuration

A composite example with HTTPS, cache, health checks, fallback, auth, and path rules:

<<< @/../examples/advanced/full-stack.yaml
