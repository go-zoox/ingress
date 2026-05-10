# Authentication Examples

This page provides examples of different authentication configurations.

Sources: [`examples/authentication/`](https://github.com/go-zoox/ingress/tree/master/examples/authentication).

## Basic authentication

### Single user

<<< @/../examples/authentication/basic-single-user.yaml yaml

### Multiple users

<<< @/../examples/authentication/basic-multi-user.yaml yaml

## Bearer token authentication

### Single token

<<< @/../examples/authentication/bearer-single.yaml yaml

### Multiple tokens

<<< @/../examples/authentication/bearer-multi.yaml yaml

## Path-level authentication

Different authentication for different paths:

<<< @/../examples/authentication/path-level-mixed.yaml yaml

## Testing

### Basic authentication

```bash
curl -u admin:admin123 -H "Host: basic-auth.example.com" http://localhost:8080/
curl -u admin:admin123 -H "Host: basic-auth-multi.example.com" http://localhost:8080/
```

### Bearer token

```bash
curl -H "Host: bearer-auth.example.com" -H "Authorization: Bearer my-secret-token-123" http://localhost:8080/
curl -H "Host: bearer-auth-multi.example.com" -H "Authorization: Bearer token1-abc123xyz" http://localhost:8080/
```

### Path-level authentication

```bash
curl -u default:default123 -H "Host: mixed-auth.example.com" http://localhost:8080/
curl -u admin:admin123 -H "Host: mixed-auth.example.com" http://localhost:8080/admin
curl -H "Host: mixed-auth.example.com" -H "Authorization: Bearer api-token-1" http://localhost:8080/api
```
