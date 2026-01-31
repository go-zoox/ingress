# Authentication

Ingress supports multiple authentication methods to secure your backend services. You can configure authentication at the rule level or path level.

## Supported Authentication Methods

- **Basic Authentication**: Username/password authentication
- **Bearer Token**: Token-based authentication
- **JWT**: JSON Web Token authentication
- **OAuth2**: OAuth 2.0 authentication
- **OIDC**: OpenID Connect authentication

## Basic Authentication

Basic Authentication uses username and password credentials.

### Single User

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        auth:
          type: basic
          basic:
            users:
              - username: admin
                password: admin123
```

### Multiple Users

You can configure multiple users for Basic Authentication:

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
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

### Using Basic Auth

Clients must include the `Authorization` header with base64-encoded credentials:

```bash
curl -u admin:admin123 http://example.com/api
```

Or manually:

```bash
curl -H "Authorization: Basic $(echo -n 'admin:admin123' | base64)" http://example.com/api
```

## Bearer Token Authentication

Bearer Token authentication uses token-based authentication.

### Single Token

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        auth:
          type: bearer
          bearer:
            tokens:
              - my-secret-token-123
```

### Multiple Tokens

You can configure multiple valid tokens:

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        auth:
          type: bearer
          bearer:
            tokens:
              - token1-abc123xyz
              - token2-def456uvw
              - token3-ghi789rst
```

### Using Bearer Token

Clients must include the `Authorization` header with the Bearer token:

```bash
curl -H "Authorization: Bearer my-secret-token-123" http://example.com/api
```

## JWT Authentication

JWT (JSON Web Token) authentication validates JWT tokens using a secret key.

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        auth:
          type: jwt
          secret: your-secret-key
```

### Using JWT

Clients must include a valid JWT token in the `Authorization` header:

```bash
curl -H "Authorization: Bearer <jwt-token>" http://example.com/api
```

## OAuth2 Authentication

OAuth2 authentication supports OAuth 2.0 flow.

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        auth:
          type: oauth2
          provider: google
          client_id: your-client-id
          client_secret: your-client-secret
          redirect_url: https://example.com/callback
          scopes:
            - openid
            - profile
            - email
```

## OIDC Authentication

OpenID Connect (OIDC) authentication extends OAuth2 with identity verification.

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        auth:
          type: oidc
          provider: google
          client_id: your-client-id
          client_secret: your-client-secret
          redirect_url: https://example.com/callback
          scopes:
            - openid
            - profile
            - email
```

## Path-Level Authentication

You can configure different authentication methods for different paths:

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
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

In this example:
- Requests to `/admin` require admin credentials
- Requests to `/api` require a bearer token
- All other requests use the default basic authentication

## Authentication Flow

1. Client makes a request to Ingress
2. Ingress checks if authentication is required for the matched rule/path
3. If authentication is required:
   - Ingress validates the credentials/token
   - If valid, the request is forwarded to the backend
   - If invalid, Ingress returns a 401 Unauthorized response
4. If no authentication is required, the request is forwarded directly

## Security Best Practices

1. **Use HTTPS**: Always use SSL/TLS when authentication is enabled
2. **Strong Passwords**: Use strong, unique passwords for Basic Auth
3. **Secure Tokens**: Generate secure, random tokens for Bearer authentication
4. **Token Rotation**: Regularly rotate tokens and update configurations
5. **Secret Management**: Store secrets securely, avoid hardcoding in configuration files
6. **Least Privilege**: Grant minimum necessary access to users
7. **Audit Logging**: Monitor authentication attempts and failures

## Troubleshooting

### 401 Unauthorized

- Verify credentials/tokens are correct
- Check that the `Authorization` header is properly formatted
- Ensure the authentication type matches the configuration

### Authentication Not Working

- Verify the authentication configuration is correct
- Check that the rule/path matches the request
- Ensure the authentication type is supported
