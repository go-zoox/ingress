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

Third-party HTTPS upstreams use **`service.mode: external`** under **`backend.service`** so **`Host`** matches each **`service.name`** without **`request.host.rewrite`**.

Runnable sample for **`service.mode`** (internal vs external) and handler paths:

<<< @/../examples/advanced/service-mode-external-mixed.yaml

For **all handler types** (`file_server`, `templates`, `script`), see [Handler Backend Examples](/examples/handler).

## Health checks with multiple services

<<< @/../examples/advanced/health-checks.yaml

## HTTP response cache (`backend.cache`)

Per-backend response caching (proxy, handler, **and** redirect) uses the same `ctx.Cache()` engine as top-level **`cache`**. See the [Caching guide](/guide/caching#http-response-cache-backendcache) for semantics, **`skip_vary`**, and httpbin-style **`Vary: Origin`** notes.

<<< @/../examples/advanced/http-response-cache.yaml

## Application cache engine (memory / Redis)

Top-level **`cache`** selects Redis or in-memory storage for matcher data **and** HTTP cache entries. The sample below adds **`backend.cache`** on a **service** backend so response bodies can be shared across instances when Redis is enabled.

<<< @/../examples/advanced/redis-cache.yaml

## Complete configuration

A composite example with HTTPS, cache, health checks, fallback, auth, and path rules:

<<< @/../examples/advanced/full-stack.yaml
