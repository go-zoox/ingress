# Path-Based Routing

This example demonstrates path-based routing to route different paths to different backend services.

Sources: [`examples/path-routing/`](https://github.com/go-zoox/ingress/tree/master/examples/path-routing).

## Basic path routing

<<< @/../examples/path-routing/basic-paths.yaml yaml

## Explanation

- Requests to `example.com/api/*` are routed to `api-service`
- Requests to `example.com/admin/*` are routed to `admin-service`
- All other requests to `example.com` are routed to `default-service`

## Complex path routing

<<< @/../examples/path-routing/complex-paths.yaml yaml

## Docker registry example

<<< @/../examples/path-routing/docker-registry.yaml yaml

## Testing

```bash
# Routes to default-service
curl -H "Host: example.com" http://localhost:8080/

# Routes to api-service
curl -H "Host: example.com" http://localhost:8080/api/users

# Routes to admin-service
curl -H "Host: example.com" http://localhost:8080/admin/dashboard
```
