# Routing

Ingress provides flexible routing capabilities to match requests and route them to appropriate backend services. You can match requests by hostname and path, with support for exact, regex, and wildcard matching.

## Host Matching

Host matching is the primary way to route requests. Ingress supports three types of host matching: **exact**, **regex**, and **wildcard**.

### Automatic `host_type` (default)

If you **omit** `host_type` or set `host_type: auto`, Ingress picks the matcher at **compile time** (when the process starts or config reloads) from the `host` string:

1. If `host` contains regexp metacharacters `( ) [ ] ^ $ | + ? \` → treat as **regex**
2. Else if `host` contains `*` → treat as **wildcard**
3. Else → **exact**

Regex is detected before `*`, so full-regex hosts such as `^.*\.example\.com$` are not treated as wildcards.

The resolved type is stored on the rule as `host_type` for the rest of the runtime (service name captures, error handling, etc.). Use **`host_type: exact`** when you must match `host` as a literal string even if it contains characters that look like a pattern.

Examples without explicit `host_type`:

```yaml
rules:
  # Compiled as regex (parentheses, \w, etc.)
  - host: ^([a-z0-9-]+)\.inlets\.example\.com$
    backend:
      service:
        name: inlets
        port: 8080
  # Compiled as wildcard
  - host: '*.api.example.com'
    backend:
      service:
        name: api-gateway
        port: 8080
  # Compiled as exact
  - host: idp.example.com
    backend:
      service:
        name: idp
        port: 443
```

### Exact Matching

Exact matching matches the hostname literally. With automatic `host_type`, a plain hostname (no regexp metacharacters and no `*`) is treated as exact:

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
```

This will match requests with `Host: example.com` exactly.

### Regex Matching

Regex matching allows you to use regular expressions to match hostnames. You can set `host_type: regex` explicitly, or omit `host_type` when the pattern contains regexp metacharacters so it is inferred automatically.

```yaml
rules:
  - host: ^t-(\w+).example.work
    host_type: regex
    backend:
      service:
        name: task.$1.svc
        port: 8080
```

In this example, `$1` refers to the first capture group in the host regex pattern. A request to `t-myapp.example.work` will be routed to `task.myapp.svc`.

**Note:** In Go’s regexp engine, `\w` is `[0-9A-Za-z_]` and does **not** include `-`. Subdomains with hyphens (e.g. `my-app.example.work`) need a pattern that allows hyphens, such as `^t-([a-zA-Z0-9-]+).example.work`, not only `(\w+)`.

### Service Name Capture Templates

When `host_type: regex` is used, you can compose service names with scoped capture templates (advanced usage):

- `${host.<index>}`: capture groups from the host regex
- `${path.<index>}`: capture groups from the matched path regex

```yaml
rules:
  - host: ^t-(\w+)-(dev|prod).example.work$
    host_type: regex
    backend:
      service:
        name: task.${host.1}.${host.2}.svc
        port: 8080
    paths:
      - path: ^/api/v1/([^/]+)/([^/]+)$
        backend:
          service:
            name: ${path.2}.${path.1}.${host.2}.${host.1}.svc
            port: 8080
```

Compatibility notes:

- Legacy `$1`, `$2`, ... in `service.name` are the preferred baseline and remain fully supported for host-regex captures.
- Path rewrite rules keep using the rewrite syntax with `$1`, `$2`, ... (for example `^/api/(.*):/v2/$1`).

### Wildcard Matching

Wildcard matching uses `*` as a wildcard character. You can set `host_type: wildcard` explicitly, or omit `host_type` when the host contains `*` and no regexp metacharacters (see [Automatic `host_type`](#automatic-host_type-default)).

```yaml
rules:
  - host: '*.example.work'
    host_type: wildcard
    backend:
      service:
        name: wildcard-service
        port: 8080
```

This will match any subdomain of `example.work`, such as `app.example.work`, `api.example.work`, etc.

## Path-Based Routing

You can define path-based routing rules within a host rule. Paths are matched using regex patterns:

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: default-service
        port: 8080
    paths:
      - path: /api
        backend:
          service:
            name: api-service
            port: 8080
      - path: /admin
        backend:
          service:
            name: admin-service
            port: 8080
```

Path matching uses regex patterns. The path `/api` will match `/api`, `/api/`, `/api/users`, etc.

### Path Matching Priority

Paths are matched in the order they are defined. The first matching path will be used. If no path matches, the host-level backend will be used.

## Request Rewriting

You can rewrite request paths, headers, and query parameters when routing to backend services.

### Path Rewriting

Rewrite request paths using regex patterns:

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        request:
          path:
            rewrites:
              - ^/api/v1/(.*):/api/v2/$1
              - ^/old:/new
```

The rewrite format is `pattern:replacement`, where `pattern` is a regex and `replacement` is the new path. Capture groups can be referenced using `$1`, `$2`, etc.

### Host Header Rewriting

Rewrite the Host header sent to the backend:

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        request:
          host:
            rewrite: true
```

When `rewrite: true`, the Host header will be set to the backend service name and port.

### Header Modification

Add or modify request headers:

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        request:
          headers:
            X-Forwarded-Proto: https
            X-Custom-Header: value
```

### Query Parameter Modification

Add or modify query parameters:

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        request:
          query:
            api_key: secret-key
            version: v2
```

## Redirects

Instead of proxying to a backend service, you can redirect requests:

```yaml
rules:
  - host: old.example.com
    backend:
      redirect:
        url: https://new.example.com
        permanent: true
```

- `url`: The redirect target URL
- `permanent`: If `true`, returns a 301 redirect; if `false`, returns a 302 redirect

## Handler Backend

In addition to proxying to a service, you can set `backend.type: handler` and use `handler.type` to choose one of:

- `static_response` (default)
- `file_server`
- `templates`
- `script`

```yaml
rules:
  - host: handler.example.com
    backend:
      service:
        name: api-service
        port: 8080
    paths:
      - path: /custom/handler/json
        backend:
          type: handler
          handler:
            type: static_response
            status_code: 200
            headers:
              Content-Type: application/json
            body: |
              {"message":"Hello, World!"}
      - path: /custom/handler/files
        backend:
          type: handler
          handler:
            type: file_server
            root_dir: /app/public
            index_file: index.html
      - path: /custom/handler/templates
        backend:
          type: handler
          handler:
            type: templates
            root_dir: /app/templates
      - path: /custom/handler/script/js
        backend:
          type: handler
          handler:
            type: script
            engine: javascript
            script: |
              ctx.response.status_code = 200
              ctx.type = "application/json"
              ctx.body = JSON.stringify({ method: ctx.method, path: ctx.path })
              ctx.setHeader("X-Handler-Engine", "javascript")
      - path: /custom/handler/script/go
        backend:
          type: handler
          handler:
            type: script
            engine: go
            script: |
              ctx.SetHeader("X-Handler-Engine", "go")
              ctx.String(200, "%s %s", ctx.Method, ctx.Path)
```

- `backend.type`: `service` (default) or `handler`
- `handler.type`: `static_response` (default), `file_server`, `templates`, or `script`
- when `handler.type=static_response`: `status_code`, `headers`, `body`
- when `handler.type=file_server`: `root_dir`, `index_file`
- when `handler.type=templates`: `root_dir`
- when `handler.type=script`: `engine`, `script`
  - `engine=javascript`: powered by `goja`; `ctx` includes:
    - `ctx.request` / `ctx.response`
    - aliases: `ctx.method`, `ctx.path`, `ctx.headers`
    - response aliases: `ctx.status` (`ctx.response.status_code`), `ctx.type` (`ctx.response.content_type`), `ctx.body` (`ctx.response.body`)
    - methods: `ctx.setHeader(key, value)` and `ctx.response.setHeader(key, value)`
  - `engine=go`: powered by `yaegi`; `ctx` is the original `*zoox.Context` (e.g. `ctx.SetHeader(...)`, `ctx.String(...)`, `ctx.Fetch()`)

## Fallback Service

If no rule matches a request, the fallback service is used:

```yaml
fallback:
  service:
    name: fallback-service
    port: 8080
```

The fallback service is useful for handling unmatched requests or providing a default backend.

## Routing Examples

### Multiple Services on Same Host

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: web-service
        port: 8080
    paths:
      - path: /api
        backend:
          service:
            name: api-service
            port: 8081
      - path: /admin
        backend:
          service:
            name: admin-service
            port: 8082
```

### Regex Host with Path Rewriting

```yaml
rules:
  - host: ^t-(\w+).example.work
    host_type: regex
    backend:
      service:
        name: task.$1.svc
        port: 8080
    paths:
      - path: /api/v1/([^/]+)
        backend:
          service:
            name: $1.example.work
            port: 8080
            request:
              path:
                rewrites:
                  - ^/api/v1/([^/]+):/api/v1/task/$1
```

### Wildcard Host Matching

```yaml
rules:
  - host: '*.example.work'
    host_type: wildcard
    backend:
      service:
        name: wildcard-service
        port: 8080
```

This matches any subdomain of `example.work` and routes to the same backend service.

## How matching is built (precompilation)

Ingress does **not** compile regular expressions on every request for host and path rules.

- When the process **starts** or when configuration is **reloaded**, `prepare()` builds an internal **router index** (`core/compile.go`): for each rule, the effective `host_type` is resolved (including **automatic** inference when omitted or `auto`), then each `host` that is regex or wildcard and each `paths[].path` pattern is compiled once with Go’s `regexp` package.
- **Rule order in your config is preserved.** Matching walks rules in order; the **first** matching host rule wins, and within a host the **first** matching path wins (same semantics as before this optimization).
- If any pattern is **invalid** (e.g. bad regex in `host` or `path`), **startup or `Reload` fails** with an error. You must fix the configuration before Ingress accepts traffic. This replaces the older behavior where some invalid patterns might only surface on the first matching request.

The per-request proxy path still uses the precompiled index. Separately, if caching is enabled, **per-host** routing decisions may be stored under a key shaped like `match.host:v2:<hostname>` (see [Caching](./caching.md)) until `cache.ttl` expires.

## Best Practices

1. **Order matters**: Place more specific rules before general ones
2. **Use exact matching when possible**: Plain hostnames infer as exact and are faster than regex or wildcard matching
3. **Omit or set `auto` for `host_type` when convenient**: Regex- or wildcard-looking `host` values are inferred at compile time; use explicit `host_type` when you need to override (for example `exact` for a literal host that contains `*` or parentheses)
4. **Test regex patterns**: Ensure your regex patterns match as expected; invalid patterns fail at startup or reload
5. **Use path routing**: Organize routes by path for better maintainability
6. **Set up fallback**: Always configure a fallback service for unmatched requests
