# Path-Based Routing

This example demonstrates path-based routing to route different paths to different backend services.

## Basic Path Routing

```yaml
version: v1
port: 8080

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

## Explanation

- Requests to `example.com/api/*` are routed to `api-service`
- Requests to `example.com/admin/*` are routed to `admin-service`
- All other requests to `example.com` are routed to `default-service`

## Complex Path Routing

```yaml
version: v1
port: 8080

rules:
  - host: example.com
    backend:
      service:
        name: web-service
        port: 8080
    paths:
      - path: /api/v1
        backend:
          service:
            name: api-v1-service
            port: 8080
      - path: /api/v2
        backend:
          service:
            name: api-v2-service
            port: 8080
      - path: /static
        backend:
          service:
            name: static-service
            port: 8080
```

## Docker Registry Example

This example shows path-based routing for a Docker registry:

```yaml
version: v1
port: 8080

rules:
  - host: docker-registry.example.com
    backend:
      service:
        name: docker-registry
        port: 80
    paths:
      - path: /v2
        backend:
          service:
            name: docker-registry-v2
            port: 80
```

## Testing

Test different paths:

```bash
# Routes to default-service
curl -H "Host: example.com" http://localhost:8080/

# Routes to api-service
curl -H "Host: example.com" http://localhost:8080/api/users

# Routes to admin-service
curl -H "Host: example.com" http://localhost:8080/admin/dashboard
```
