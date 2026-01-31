# Authentication Examples

This page provides examples of different authentication configurations.

## Basic Authentication

### Single User

```yaml
version: v1
port: 8080

rules:
  - host: basic-auth.example.com
    backend:
      service:
        name: api-service
        port: 8080
        auth:
          type: basic
          basic:
            users:
              - username: admin
                password: admin123
```

### Multiple Users

```yaml
version: v1
port: 8080

rules:
  - host: basic-auth.example.com
    backend:
      service:
        name: api-service
        port: 8080
        auth:
          type: basic
          basic:
            users:
              - username: admin
                password: admin123
              - username: user1
                password: user123
              - username: user2
                password: user456
```

## Bearer Token Authentication

### Single Token

```yaml
version: v1
port: 8080

rules:
  - host: bearer-auth.example.com
    backend:
      service:
        name: api-service
        port: 8080
        auth:
          type: bearer
          bearer:
            tokens:
              - my-secret-token-123
```

### Multiple Tokens

```yaml
version: v1
port: 8080

rules:
  - host: bearer-auth.example.com
    backend:
      service:
        name: api-service
        port: 8080
        auth:
          type: bearer
          bearer:
            tokens:
              - token1-abc123xyz
              - token2-def456uvw
              - token3-ghi789rst
```

## Path-Level Authentication

Different authentication for different paths:

```yaml
version: v1
port: 8080

rules:
  - host: mixed-auth.example.com
    backend:
      service:
        name: api-service
        port: 8080
        auth:
          type: basic
          basic:
            users:
              - username: default
                password: default123
    paths:
      - path: /admin
        backend:
          service:
            name: admin-service
            port: 8080
            auth:
              type: basic
              basic:
                users:
                  - username: admin
                    password: admin123
                  - username: superadmin
                    password: super123
      - path: /api
        backend:
          service:
            name: api-service
            port: 8080
            auth:
              type: bearer
              bearer:
                tokens:
                  - api-token-1
                  - api-token-2
                  - api-token-3
```

## Testing

### Basic Authentication

```bash
curl -u admin:admin123 http://basic-auth.example.com/api
```

### Bearer Token

```bash
curl -H "Authorization: Bearer my-secret-token-123" http://bearer-auth.example.com/api
```

### Path-Level Authentication

```bash
# Uses default basic auth
curl -u default:default123 http://mixed-auth.example.com/

# Uses admin basic auth
curl -u admin:admin123 http://mixed-auth.example.com/admin

# Uses bearer token
curl -H "Authorization: Bearer api-token-1" http://mixed-auth.example.com/api
```
