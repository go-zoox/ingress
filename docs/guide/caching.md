# Caching

Ingress supports caching to improve performance and reduce load on backend services. You can use in-memory caching or Redis for distributed caching.

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

For example, with `prefix: ingress:`, a key `match.host:example.com` becomes `ingress:match.host:example.com`.

## What Gets Cached

Ingress caches:

1. **Routing Decisions**: Host and path matching results
2. **Service Configurations**: Parsed service configurations
3. **DNS Resolutions**: Resolved backend service addresses

Caching routing decisions is particularly beneficial when:
- You have complex routing rules
- DNS resolution is slow
- Backend service discovery is expensive

## Cache Invalidation

Cache entries are automatically invalidated when:

1. **TTL Expires**: Entries expire after the configured TTL
2. **Configuration Reload**: Reloading configuration clears relevant cache entries
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
