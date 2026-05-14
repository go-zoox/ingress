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

The outbound `Host` header is controlled by **`service.request.host.rewrite`** (optional override) and **`backend.mode`** when `rewrite` is omitted. See **[Request and Response Rewriting](./rewriting.md)** for full detail and **fallback** behavior.

Example with explicit `rewrite`:

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

Preferred for a third-party HTTPS origin:

```yaml
rules:
  - host: mirror.example.com
    backend:
      mode: external
      service:
        protocol: https
        name: upstream.example.org
```

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

Use **`backend.redirect`** instead of proxying. **`backend.type` is optional**: when only **`redirect`** is configured on that backend, Ingress infers **`redirect`** automatically—you normally omit **`backend.type`**. Valid explicit values are **`service`**, **`handler`**, and **`redirect`**; only the matching block may be populated. If **`service`**, **`handler`**, and **`redirect`** blocks **together look ambiguous** while **`type` is omitted**, **`ingress validate`** fails until you set **`backend.type`** explicitly.

These two rules behave the same after inference—compare **`type: redirect`** with omission:

```yaml
rules:
  - host: old-explicit.example.com
    backend:
      type: redirect
      redirect:
        url: https://new.example.com
        permanent: true
  - host: old-inferred.example.com
    backend:
      redirect:
        url: https://new.example.com
        permanent: true
```

Runnable twin-host sample: **`examples/ssl-tls/route-redirect.yaml`**.

Fields:

- **`url`**: Target URL. If it does not start with `http://` or `https://`, Ingress treats the value as a host (optional port) and builds the full URL with the incoming request’s scheme, original path, and query string.
- **`permanent`**: When `false`, uses **302**; when `true`, uses **301**—unless `with_origin_method_and_body` is enabled (below).
- **`with_origin_method_and_body`** (default `false`): When `true`, uses **307** / **308** so clients keep the original HTTP method and body (temporary vs permanent follows `permanent`). When `false`, uses **302** / **301** as above.

**Capture templates** in `url` follow the same rules as `service.name`: `${host.N}` and `${path.N}` from regex captures; for regex/wildcard hosts you can also use legacy **`$1`-style** substitution from the host pattern. Path captures apply when the redirect is chosen from a matched `paths[].path` entry.

```yaml
rules:
  - host: '^bigscreen-([^.]+)\.ys\.example\.com$'
    host_type: regex
    backend:
      type: redirect
      redirect:
        url: https://bigscreen-$1.other.example.com
```

For host-level redirect combined with path-specific proxies or path-only redirects, **`examples/redirect/capture-and-mixed.yaml`** mixes **explicit `backend.type`** on some backends with **omission** on others so you can compare styles in one file.

Forced HTTP→HTTPS uses `https.redirect_from_http` (including optional `with_origin_method_and_body`); see the [SSL/TLS guide](/guide/ssl-tls).

## Handler Backend

Path backends can answer from **`backend.handler`** instead of proxying. **`backend.type` is optional**: when only **`handler`** is configured, Ingress infers **`handler`**. The snippet below keeps **`type: handler`** on the first path only so it contrasts with the paths that omit **`backend.type`**.

Use **`handler.type`** to choose one of:

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
          handler:
            type: file_server
            root_dir: /app/public
            index_file: index.html
      - path: /custom/handler/templates
        backend:
          handler:
            type: templates
            root_dir: /app/templates
      - path: /custom/handler/script/js
        backend:
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
          handler:
            type: script
            engine: go
            script: |
              ctx.SetHeader("X-Handler-Engine", "go")
              ctx.String(200, "%s %s", ctx.Method, ctx.Path)
```

- `backend.type`: optional—Ingress infers **`service`**, **`handler`**, or **`redirect`** from which block is configured when unambiguous; set **`backend.type` explicitly** only when **`ingress validate`** reports ambiguity
- `backend.mode`: `internal` (default) or `external`—default **`Host`** toward the upstream when **`service.request.host.rewrite`** is omitted (**`external`** aligns **`Host`** to **`service.name`**; see [Rewriting](./rewriting.md))
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

The fallback service is useful for handling unmatched requests or providing a default backend. Unless you set **`fallback.service.request.host.rewrite`**, the upstream **`Host`** aligns to the fallback service (see [Rewriting](./rewriting.md)).

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
