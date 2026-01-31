# Health Checks

Ingress supports health checks at two levels: outer health checks (for the Ingress service itself) and inner health checks (for backend services).

## Outer Health Checks

Outer health checks allow external systems to check if Ingress is running and healthy.

### Configuration

```yaml
healthcheck:
  outer:
    enable: true
    path: /healthz
    ok: true
```

### Configuration Fields

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `enable` | bool | Enable outer health check | `false` |
| `path` | string | Health check endpoint path | `/healthz` |
| `ok` | bool | Always return OK status | `false` |

### Usage

When enabled, Ingress responds to health check requests at the configured path:

```bash
curl http://localhost:8080/healthz
```

If `ok: true`, the endpoint always returns a successful response. Otherwise, it may return the actual health status based on internal checks.

## Inner Health Checks

Inner health checks monitor the health of backend services and can be used for load balancing and failover.

### Global Inner Health Check Configuration

```yaml
healthcheck:
  inner:
    enable: true
    interval: 30    # Check interval in seconds
    timeout: 5      # Check timeout in seconds
```

### Service-Level Health Checks

You can configure health checks for individual services:

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        healthcheck:
          enable: true
          method: GET
          path: /health
          status: [200]
          ok: false
```

### Health Check Configuration Fields

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `enable` | bool | Enable health check for this service | `false` |
| `method` | string | HTTP method for health check | `GET` |
| `path` | string | Health check endpoint path | `/health` |
| `status` | array | List of valid HTTP status codes | `[200]` |
| `ok` | bool | Always consider service healthy | `false` |

### Health Check Methods

Supported HTTP methods:
- `GET` (default)
- `POST`
- `HEAD`

### Health Check Status Codes

The `status` field specifies which HTTP status codes are considered healthy. For example:

```yaml
healthcheck:
  enable: true
  method: GET
  path: /health
  status: [200, 201]  # Both 200 and 201 are considered healthy
```

### Health Check Interval and Timeout

The global inner health check configuration controls how often services are checked:

```yaml
healthcheck:
  inner:
    enable: true
    interval: 30    # Check every 30 seconds
    timeout: 5       # Timeout after 5 seconds
```

- `interval`: How often to check the service (in seconds)
- `timeout`: Maximum time to wait for a response (in seconds)

## Health Check Examples

### Basic Service Health Check

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        healthcheck:
          enable: true
          method: GET
          path: /health
          status: [200]
```

### Custom Health Check Path

```yaml
rules:
  - host: api.example.com
    backend:
      service:
        name: api-service
        port: 8080
        healthcheck:
          enable: true
          method: GET
          path: /api/health
          status: [200, 204]
```

### Multiple Status Codes

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        healthcheck:
          enable: true
          method: GET
          path: /health
          status: [200, 201, 204]  # Accept multiple success codes
```

### POST Health Check

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        healthcheck:
          enable: true
          method: POST
          path: /health/check
          status: [200]
```

### Always Healthy (Skip Actual Check)

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        healthcheck:
          enable: true
          ok: true  # Always consider healthy, skip actual check
```

## Health Check Behavior

When a service fails health checks:

1. Ingress continues to route traffic (health checks are informational)
2. Health check status can be used for monitoring and alerting
3. Failed health checks are logged for debugging

## Monitoring Health Checks

You can monitor health check status through:

1. **Logs**: Health check failures are logged
2. **Metrics**: Health check metrics (if metrics are enabled)
3. **External Monitoring**: Use the outer health check endpoint

## Best Practices

1. **Use Appropriate Intervals**: Balance between timely detection and resource usage
2. **Set Reasonable Timeouts**: Avoid timeouts that are too short or too long
3. **Use Standard Paths**: Use common health check paths like `/health` or `/healthz`
4. **Monitor Health Status**: Set up alerts for failed health checks
5. **Test Health Endpoints**: Ensure backend services have working health check endpoints
6. **Handle Graceful Degradation**: Design services to handle health check failures gracefully

## Troubleshooting

### Health Check Always Failing

- Verify the health check path exists on the backend service
- Check that the HTTP method matches what the backend expects
- Ensure the backend service is running and accessible
- Verify network connectivity between Ingress and backend

### Health Check Timeout

- Increase the timeout value if the backend is slow to respond
- Check backend service performance
- Verify network latency

### Health Check Not Running

- Ensure `enable: true` is set
- Check that the global inner health check is enabled
- Verify the configuration is correct
