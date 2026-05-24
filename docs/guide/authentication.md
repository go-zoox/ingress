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

OAuth2 authentication enables a redirect-based login flow using third-party identity providers.
When a user visits a protected route without a valid session, ingress redirects them to the
provider's authorization page. After successful login, the provider redirects back to ingress,
which stores the user session and forwards the request to the upstream service.

### Supported Providers

| Provider | Identifier |
|----------|-----------|
| GitHub | `github` |
| GitLab | `gitlab` |
| Google | `google` |
| Microsoft | `microsoft` |
| Feishu (飞书) | `feishu` |
| Slack | `slack` |
| Kakao | `kakao` |
| Auth0 | `auth0` |
| Okta | `okta` |
| Doreamon | `doreamon` |

### Basic Configuration

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        auth:
          type: oauth2
          oauth2:
            provider: github
            client_id: your-github-client-id
            client_secret: your-github-client-secret
            # scopes is optional — defaults are provider-specific
            scopes:
              - user:email
```

**Required fields:**

| Field | Description |
|-------|-------------|
| `type` | Must be `oauth2` |
| `oauth2.provider` | Identity provider name (see table above) |
| `oauth2.client_id` | OAuth application client ID |
| `oauth2.client_secret` | OAuth application client secret |

**Optional fields:**

| Field | Default | Description |
|-------|---------|-------------|
| `oauth2.scopes` | Provider-specific (see below) | OAuth scopes to request |
| `oauth2.redirect_url` | Auto-generated from request host | Custom callback URL |

**Default scopes per provider:**

| Provider | Default scopes |
|----------|---------------|
| GitHub | `user:email` |
| GitLab | `read_user` |
| Google | `openid profile email` |
| Microsoft | `openid profile email` |
| Feishu | `user:email` |
| Slack | `users:read` |
| Auth0 | `openid profile email` |
| Okta | `openid profile email` |
| Doreamon / Kakao | *(provider default)* |

The callback path is always `/oauth2/callback` (fixed). The full redirect URL is
auto-generated from the incoming request's scheme and host, e.g.
`http://example.com/oauth2/callback`.

### Connect JWT Headers

When `connect.enabled` is `true`, ingress injects user identity headers into the upstream
request after successful OAuth2 authentication. This allows the backend service to identify
the authenticated user without re-implementing the OAuth2 flow.

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        auth:
          type: oauth2
          oauth2:
            provider: github
            client_id: your-client-id
            client_secret: your-client-secret
            scopes:
              - user:email
            connect:
              enabled: true
              jwt:
                secret: your-shared-jwt-secret
                algorithm: hs256       # optional, default: hs256
                expires_in: "5m"       # optional, default: 5m
```

**Injected headers:**

| Header | Description |
|--------|-------------|
| `X-Connect-Token` | JWT containing `id`, `username`, `email`, `nickname`, `avatar` |
| `X-Connect-Timestamp` | Unix millisecond timestamp of token issuance |

**JWT decoding by the upstream (example in Go):**

```go
import "github.com/go-zoox/jwt"

token := r.Header.Get("X-Connect-Token")
j := jwt.New("your-shared-jwt-secret")
claims, err := j.Verify(token)
if err != nil {
    // invalid token
}
userID := claims.Get("id").String()
```

### Authentication Flow

1. Client visits `https://example.com/protected`
2. Ingress detects no session → generates CSRF state → redirects to provider login
3. User authenticates with the provider
4. Provider redirects to `https://example.com/oauth2/callback?code=xxx&state=yyy`
5. Ingress validates state, exchanges code for token, fetches user info
6. User info is stored in an encrypted session cookie
7. Client is redirected back to the original URL (`/protected`)
8. On subsequent requests, the session cookie identifies the authenticated user
9. If `connect.enabled: true`, `X-Connect-Token` and `X-Connect-Timestamp` are injected into the upstream request

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
