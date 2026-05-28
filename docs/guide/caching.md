# Caching

Ingress supports caching to improve performance and reduce load on backend services. You can use in-memory caching or Redis for distributed caching.

**Two layers:** top-level **`cache`** configures the shared **`ctx.Cache()`** engine (matcher/DNS, and so on). Optional per-backend **`backend.cache`** stores **HTTP responses** in that same engine when enabled—see [below](#http-response-cache-backendcache).

## Cache Configuration

### In-Memory Caching

By default, Ingress uses in-memory caching:

```yaml
cache:
  ttl: 30  # Cache TTL in seconds (default: 60)
```

This caches routing decisions and other data in memory. The cache is local to each Ingress instance.

### Redis Caching

For distributed caching across multiple Ingress instances, use Redis:

```yaml
cache:
  ttl: 30
  engine: redis
  host: 127.0.0.1
  port: 6379
  password: '123456'
  db: 2
  prefix: ingress:
```

### Configuration Fields

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `ttl` | int | Cache TTL in seconds | `60` |
| `engine` | string | Cache engine: `memory` or `redis` | `memory` |
| `host` | string | Redis host (for Redis engine) | `127.0.0.1` |
| `port` | int | Redis port (for Redis engine) | `6379` |
| `password` | string | Redis password (for Redis engine) | - |
| `db` | int | Redis database number (for Redis engine) | `0` |
| `prefix` | string | Cache key prefix (for Redis engine) | - |

## Cache TTL

The `ttl` (Time To Live) determines how long cached items remain valid:

```yaml
cache:
  ttl: 30  # Cache items expire after 30 seconds
```

- Shorter TTL: More up-to-date data, but more backend requests
- Longer TTL: Fewer backend requests, but potentially stale data

Choose a TTL that balances freshness with performance for your use case.

## In-Memory Cache

In-memory caching is the simplest option and works well for single-instance deployments:

```yaml
cache:
  ttl: 60
```

**Advantages:**
- No external dependencies
- Fast access
- Simple configuration

**Disadvantages:**
- Not shared across instances
- Lost on restart
- Limited by available memory

## Redis Cache

Redis caching is recommended for multi-instance deployments or when you need persistent caching:

```yaml
cache:
  ttl: 60
  engine: redis
  host: redis.example.com
  port: 6379
  password: your-password
  db: 0
  prefix: ingress:
```

### Redis Configuration

- **host**: Redis server hostname or IP address
- **port**: Redis server port (default: 6379)
- **password**: Redis password (optional, omit if no password)
- **db**: Redis database number (0-15, default: 0)
- **prefix**: Prefix for all cache keys (useful for namespacing)

### Redis Connection

Ingress connects to Redis when it starts. If Redis is unavailable:

- Ingress will log an error
- It may fall back to in-memory caching (depending on configuration)
- Health checks may fail

### Redis Key Format

With a prefix, cache keys are formatted as:

```
{prefix}{key}
```

For example, with `prefix: ingress:`, a host routing cache key `match.host:v2:example.com` becomes `ingress:match.host:v2:example.com`. HTTP response cache keys look like `ingress:httpcache:v1:<digest>`. The `v2` segment denotes the matcher cache value shape; it may change in future releases.

## What Gets Cached

Ingress caches:

1. **Routing Decisions**: Host and path matching results
2. **Service Configurations**: Parsed service configurations
3. **DNS Resolutions**: Resolved backend service addresses
4. **HTTP responses** (optional): When **`backend.cache.enabled: true`** on a backend, eligible **GET**/**HEAD** responses for **service**, **handler**, and **redirect** can be stored in the same `ctx.Cache()` backend—see [HTTP response cache (`backend.cache`)](#http-response-cache-backendcache).

Caching routing decisions is particularly beneficial when:
- You have complex routing rules
- DNS resolution is slow
- Backend service discovery is expensive

## HTTP response cache (`backend.cache`)

**Different from** top-level `cache`: the root `cache` block only chooses **Redis vs memory** and connection defaults for **`ctx.Cache()`**. **`backend.cache`** is **per route** under `rules[].backend` or `paths[].backend`.

When **`cache.enabled: true`**:

- **Service** backends cache successful upstream bodies (after buffering) for **client GET** requests that populate the shared cache key; **HEAD** can **hit** the same entry.
- **Handler** backends cache static responses when storage rules pass (no `Vary`, size limits, and so on).
- **Redirect** backends cache the final status and `Location` after template expansion (`GET` population path).

**Rationale for skipping some responses:** Ingress does **not** store responses with a non-empty **`Vary`** header unless **`cache.skip_vary: true`** (then **`Vary` is dropped** when storing and on cache hits). Without `skip_vary`, Ingress does not compute separate cache keys per `Vary`—see the [Configuration](configuration.md#backendcache-http-response-cache) note. It also skips responses that include **`Set-Cookie`** when **`skip_when_set_cookie`** is **true** (default), to avoid caching session-specific payloads. Many public httpbin endpoints return **`Vary: Origin`**, so they will not populate the cache unless you opt in with **`skip_vary`** and accept a single shared variant.

### Per-path rules (`backend.cache.paths`)

When **`paths`** is configured on a backend, Ingress evaluates rules **in order** (first match wins). Each rule can **`cache`** or **`bypass`** HTTP response caching for matching request paths. Unmatched paths follow **`default`** (`cache` or `bypass`; default **`cache`**). Optional per-rule **`ttl`** and **`max_body_bytes`** override the backend defaults for that path only. Omit **`paths`** to cache all paths on the backend (previous behavior).

Details, bypass rules, and field reference: [Configuration — `backend.cache`](configuration.md#backendcache-http-response-cache). Runnable YAML: [`examples/advanced/http-response-cache.yaml`](https://github.com/go-zoox/ingress/blob/master/examples/advanced/http-response-cache.yaml), Redis-backed storage: [`examples/advanced/redis-cache.yaml`](https://github.com/go-zoox/ingress/blob/master/examples/advanced/redis-cache.yaml), path rules: [`examples/advanced/http-response-cache-paths.yaml`](https://github.com/go-zoox/ingress/blob/master/examples/advanced/http-response-cache-paths.yaml).

## Cache Invalidation

Cache entries are automatically invalidated when:

1. **TTL Expires**: Entries expire after the configured TTL
2. **Configuration Reload**: `Reload` runs `prepare()`, which clears the configured cache backend (same as startup). Stale host entries are therefore removed on reload; rely on TTL for gradual expiry if you rely on external cache tooling.
3. **Manual Invalidation**: Some cache entries may be invalidated on specific events

## Cache Performance

### Monitoring Cache Performance

You can monitor cache performance through:

- **Cache Hit Rate**: Percentage of requests served from cache
- **Cache Miss Rate**: Percentage of requests that required backend lookup
- **Cache Size**: Number of items in cache (for in-memory)

### Optimizing Cache Performance

1. **Adjust TTL**: Find the right balance between freshness and performance
2. **Use Redis for Multi-Instance**: Share cache across instances
3. **Monitor Cache Usage**: Track hit/miss rates
4. **Set Appropriate Prefixes**: Organize cache keys with prefixes

## Best Practices

1. **Choose the Right Engine**: Use in-memory for single instances, Redis for multiple instances
2. **Set Appropriate TTL**: Balance freshness with performance
3. **Use Redis Prefixes**: Namespace cache keys to avoid conflicts
4. **Monitor Redis**: Ensure Redis is available and performing well
5. **Secure Redis**: Use passwords and network security for Redis
6. **Plan for Cache Misses**: Design systems to handle cache misses gracefully

## Troubleshooting

### Cache Not Working

- Verify cache configuration is correct
- Check TTL is set appropriately
- Ensure Redis is accessible (if using Redis)
- Check logs for cache-related errors

### Redis Connection Issues

- Verify Redis host and port are correct
- Check network connectivity to Redis
- Verify Redis password (if required)
- Ensure Redis is running and accessible

### High Cache Miss Rate

- Consider increasing TTL
- Check if routing rules are too dynamic
- Verify cache is actually being used
- Review cache key patterns

### Memory Usage (In-Memory Cache)

- Monitor memory usage
- Consider reducing TTL if memory is constrained
- Switch to Redis for better memory management
- Review what's being cached
