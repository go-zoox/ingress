# Basic Setup

This example shows a basic Ingress configuration for a simple reverse proxy setup.

## Configuration

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

## Explanation

- **port**: Ingress listens on port 8080
- **rules**: Defines routing rules
- **host**: Matches requests with `Host: example.com`
- **backend.service**: Routes to `backend-service` on port 8080

## Testing

Start Ingress:

```bash
ingress run -c ingress.yaml
```

Test the setup:

```bash
curl -H "Host: example.com" http://localhost:8080
```

## Multiple Services

You can configure multiple services:

```yaml
version: v1
port: 8080

rules:
  - host: web.example.com
    backend:
      service:
        name: web-service
        port: 8080
  - host: api.example.com
    backend:
      service:
        name: api-service
        port: 8081
```
