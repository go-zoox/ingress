# Request and Response Rewriting

Ingress provides flexible rewriting capabilities for requests and responses, allowing you to modify headers, paths, query parameters, and more before forwarding to backend services.

## Request Rewriting

Request rewriting modifies the request before it's sent to the backend service.

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

The rewrite format is `pattern:replacement`:
- `pattern`: Regex pattern to match
- `replacement`: Replacement string (can use capture groups like `$1`, `$2`)

#### Path Rewrite Examples

**Simple Path Rewrite:**
```yaml
request:
  path:
    rewrites:
      - ^/api:/v2/api
```

**Path Rewrite with Capture Groups:**
```yaml
request:
  path:
    rewrites:
      - ^/api/v1/(.*):/api/v2/$1
```

This rewrites `/api/v1/users` to `/api/v2/users`.

**Multiple Path Rewrites:**
```yaml
request:
  path:
    rewrites:
      - ^/ip3/(.*):/$1
      - ^/ip2:/ip
```

Rewrites are applied in order. The first matching rewrite is used.

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

When `rewrite: true`:
- The Host header is set to `{service-name}:{port}`
- Useful when backend services expect a specific hostname

When `rewrite: false` (default):
- The original Host header is preserved
- Backend receives the original hostname from the client

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
            X-User-ID: "12345"
```

Headers are added or overwritten. Common use cases:
- Set `X-Forwarded-Proto` for HTTPS detection
- Add authentication headers
- Pass user information
- Set custom headers for backend services

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

Query parameters are added to the request. If a parameter already exists, it may be overwritten.

### Request Delay

Add a delay before forwarding the request:

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        request:
          delay: 100  # Delay in milliseconds
```

Useful for:
- Rate limiting simulation
- Testing timeout behavior
- Throttling requests

### Request Timeout

Set a timeout for backend requests:

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        request:
          timeout: 30  # Timeout in seconds
```

If the backend doesn't respond within the timeout, the request fails.

## Response Rewriting

Response rewriting modifies the response before sending it to the client.

### Response Header Modification

Modify response headers:

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        response:
          headers:
            X-Custom-Header: value
            Cache-Control: no-cache
```

Common use cases:
- Add security headers
- Modify caching headers
- Add custom headers for clients

## Complete Rewriting Example

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
          path:
            rewrites:
              - ^/api/v1/(.*):/api/v2/$1
          headers:
            X-Forwarded-Proto: https
            X-Custom-Header: value
          query:
            version: v2
          delay: 0
          timeout: 30
        response:
          headers:
            X-Response-Header: value
```

## Path Rewriting with Regex

### Capture Groups

Use capture groups in path rewrites:

```yaml
request:
  path:
    rewrites:
      - ^/api/v1/([^/]+)/(.*):/api/v2/$1/$2
```

This captures two groups and reorders them.

### Complex Path Rewrites

```yaml
rules:
  - host: httpbin.example.work
    backend:
      service:
        name: httpbin.zcorky.com
        port: 443
        request:
          host:
            rewrite: true
          path:
            rewrites:
              - ^/ip3/(.*):/$1
              - ^/ip2:/ip
    paths:
      - path: /httpbin.org
        backend:
          service:
            name: httpbin.org
            port: 443
            request:
              path:
                rewrites:
                  - ^/httpbin.org/(.*):/$1
```

## Best Practices

1. **Test Rewrite Patterns**: Verify regex patterns match as expected
2. **Order Matters**: Place more specific rewrites before general ones
3. **Use Capture Groups**: Leverage regex capture groups for flexible rewrites
4. **Preserve Important Headers**: Be careful not to overwrite critical headers
5. **Document Rewrites**: Document complex rewrite rules for maintainability
6. **Monitor Impact**: Monitor how rewrites affect backend services

## Common Use Cases

### API Version Migration

```yaml
request:
  path:
    rewrites:
      - ^/api/v1/(.*):/api/v2/$1
```

### Path Normalization

```yaml
request:
  path:
    rewrites:
      - ^/old-path/(.*):/new-path/$1
```

### Adding Authentication Headers

```yaml
request:
  headers:
    Authorization: Bearer token-here
```

### Setting Protocol Information

```yaml
request:
  headers:
    X-Forwarded-Proto: https
    X-Forwarded-For: $remote_addr
```

## Troubleshooting

### Rewrite Not Working

- Verify the regex pattern matches the path
- Check the rewrite order (first match wins)
- Ensure the rewrite syntax is correct (`pattern:replacement`)
- Test the regex pattern separately

### Headers Not Set

- Verify header names are correct
- Check for typos in header values
- Ensure headers are in the correct section (request vs response)

### Path Rewrite Issues

- Test regex patterns with a regex tester
- Verify capture group references (`$1`, `$2`, etc.)
- Check for escaping issues in special characters
