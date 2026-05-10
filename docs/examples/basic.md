# Basic Setup

This example shows a basic Ingress configuration for a simple reverse proxy setup.

Source files live under [`examples/basic/`](https://github.com/go-zoox/ingress/tree/master/examples/basic).

## Minimal configuration

<<< @/../examples/basic/ingress.yaml yaml

## Explanation

- **port**: Ingress listens on port 8080
- **rules**: Defines routing rules
- **host**: Matches requests with `Host: example.com`
- **backend.service**: Routes to `backend-service` on port 8080

## Testing

```bash
ingress run -c examples/basic/ingress.yaml
```

```bash
curl -H "Host: example.com" http://localhost:8080
```

## Multiple services

<<< @/../examples/basic/multi-host.yaml yaml
