# Authentication

Ingress supports multiple authentication methods to secure your backend services. You can configure authentication at the rule level or path level.

## Supported Authentication Methods

- **Basic Authentication**: Username/password authentication
- **Bearer Token**: Token-based authentication
- **JWT**: JSON Web Token authentication
- **OAuth2**: OAuth 2.0 authentication
- **OIDC**: OpenID Connect authentication

## Enabling / Disabling Authentication

When you configure `auth.type`, authentication is enabled by default. You can
use the `enabled` field to temporarily disable authentication (e.g. for
debugging or gradual rollouts) without removing the auth configuration:

```yaml
auth:
  enabled: false   # set to true to re-enable; omit to default to true when type is set
  type: basic
  basic:
    users:
      - username: admin
        password: admin123
```

| `enabled` value | `type` set? | Behavior |
|-----------------|------------|----------|
| *(omitted)* | No | No auth (default) |
| *(omitted)* | Yes | Auth **enabled** (default) |
| `true` | Any | Auth **enabled** |
| `false` | Any | Auth **disabled** (config preserved) |

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
| `oauth2.redirect_url` | Auto-generated from request host (`/oauth2/callback`) | Custom callback URL; its **path** becomes the callback path (see below) |

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

The callback path defaults to `/oauth2/callback`. When `oauth2.redirect_url` is
set explicitly, **its path is also used as the callback path**, so provider
applications that require a specific callback URL (for example Doreamon's
`/login/doreamon/callback` convention) work:

```yaml
auth:
  type: oauth2
  oauth2:
    provider: doreamon
    client_id: ...
    client_secret: ...
    redirect_url: https://example.com/login/doreamon/callback
```

The callback request must be routed to a service that has the **same**
`auth.oauth2` config (host-level backend, or a path rule that covers the callback
path), and the exact URL must be registered in the identity provider (Doreamon
compares `redirect_uri` against the application's registered redirect URI list).

When `oauth2.redirect_url` is unset, the callback URL is auto-generated from the
incoming request's scheme and host, e.g. `http://example.com/oauth2/callback`.

The callback path is reserved for the OAuth2 flow: requests to it without
`code`/`state` are rejected with **401**, so point it at a path that no backend
needs.

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

#### HTTP → HTTPS behind a load balancer

The OAuth2 **`redirect_uri`** is auto-generated from the request scheme and host (`auth_oauth2.go`). If ingress sits behind a TLS-terminating reverse proxy (so it receives plain HTTP internally), it honors **`X-Forwarded-Proto: https`**. For public/multi-replica deployments, we recommend setting **`oauth2.redirect_url`** explicitly to the public HTTPS URL so the identity provider always sees the same callback:

```yaml
auth:
  type: oauth2
  oauth2:
    provider: github
    client_id: ...
    client_secret: ...
    redirect_url: https://example.com/oauth2/callback
```

If your provider application requires a non-default callback path, keep that path
in `redirect_url`; it becomes a callback path **in addition to** the default
`/oauth2/callback` (both are accepted, so in-flight logins survive the change).

#### Multi-replica deployments

The OAuth2/OIDC **CSRF `state`** and the **post-login redirect URL** are carried in an **encrypted session cookie**. The cookie is encrypted with `app.Config.SecretKey`, which defaults to a **per-process random** key when you do not configure it. If you run **multiple ingress replicas behind a load balancer**, the callback request can land on a different replica than the one that issued the redirect: that replica cannot decrypt the cookie, so the CSRF `state` is empty and the callback fails with **401**.

To make OAuth2 work across replicas, set an identical session key on **every** replica:

```yaml
# ingress.yaml
secret_key: a-long-random-shared-secret
```

or

```bash
# the SECRET_KEY env var overrides the config value
export SECRET_KEY="a-long-random-shared-secret"
```

Generate one with `openssl rand -hex 32`. Because the session lives entirely in the cookie, you do **not** need sticky sessions once the key is stable.

At startup ingress logs `session key: no secret_key/SECRET_KEY configured, using a per-process key …`
when neither is set, so you can spot a deployment that will force a re-login on every restart.

The admin console keeps its **own** session cookie (`ingress_admin_session`) and no
longer shares the gateway's `gsession` value; the key is still the same
(`secret_key`/`SECRET_KEY`), so the two apps share one secret but not one payload.
After upgrading, gateway OAuth2 sessions are preserved (same cookie name and key),
while admin console operators have to log in once because of the cookie rename.

#### Concurrent logins

`state`/redirect are tracked **per flow** (keyed by `state`), so a user who starts two logins at once (two tabs, a refresh, etc.) no longer has the two flows overwrite each other: each callback resolves its own `state` and redirects back to its own original URL. A legacy single-state fallback keeps in-flight flows from older versions working across a rolling upgrade.

#### Stale CSRF state and the ~4KB session cookie limit

The CSRF `state`, the post-login URL and the user profile all live in one
**client-side** encrypted session cookie (`gsession`). Browsers silently **drop**
a `Set-Cookie` whose name+value exceeds ~4KB. If that happens, a new `state` can
never be persisted, so every callback fails with **401** — and it does not recover
by itself. This is why restarting ingress used to be the only cure: without a
configured `secret_key` the session key is per-process, so a restart makes the old
cookie unreadable and a small one is written again (and everyone is logged out).

Ingress now keeps the OAuth2 session payload bounded (max 8 in-flight flows,
ordered oldest-first, redirect URLs capped, access token and permissions/groups
not stored) and **self-heals** a callback whose `state` is missing: it deletes the
stale gateway session cookie (a single small `Set-Cookie`, instantly accepted by
the browser instead of the oversized value it had been rejecting) and restarts the
login by redirecting to `/?_ingress_oauth2_retry=1`. The retried login gets a CSRF
state with an `r1.` prefix; a second stale callback is answered with **401** instead
of looping `/authorize` ⇄ `/callback` forever. The bound uses two markers — the
retry query parameter/state prefix and a short-lived `ingress_oauth2_retry` cookie
(60s) for redirect chains that drop the query — so it does not depend on a cookie
the browser may be dropping.

Notes on the automatic recovery:

- it deletes the whole gateway OAuth2 session cookie, so in-flight logins from
  other tabs are dropped too — they simply restart;
- it redirects to `/`, so the original deep link is not preserved;
- if `/` itself is not protected, the redirect is a no-op page load, but the
  stale cookie is already gone, so the next protected navigation logs in normally.

For **multi-replica** deployments a shared `secret_key` is still required: without
it the retry cannot succeed and the user gets the bounded 401 after one attempt.

#### Troubleshooting: 401 on the OAuth2 callback

Check the ingress log line `oauth2 authentication failed for host: …`:

| Message | Cause / action |
|---------|----------------|
| `state mismatch …` | No matching flow state in the browser cookie: cookie issued by another replica/process, oversized cookie dropped by the browser, or callback opened from history. Ingress self-heals once (see above); a repeated failure is logged as `state mismatch …` again — check `secret_key` (multi-replica) and clear the site cookies. |
| `identity provider returned "…"` | The provider rejected the authorization request (`access_denied`, scope, or `redirect_uri` not registered). Fix the provider application settings. |
| `exchange failed: …` | The `/token` or `/user` call to the provider failed (wrong client secret, provider unreachable/TLS). The flow state is only consumed after a successful exchange, so reloading the callback URL retries it — unless the authorization code was already redeemed at `/token` (codes are single-use); in that case start a new login. |
| `missing code or state` | The callback request did not carry `code`/`state` (truncated redirect, manual URL edit, or the provider returned an error without `code`). |
| `exchange panic: …` | The provider answered `/token` with an unexpected payload (go-zoox/oauth2 nil-error dereference, e.g. HTTP 200 with a JSON `code`). Check the provider's token endpoint; the request is answered with 401 instead of a 500. |

With **path-level auth** (`paths[].backend.service.auth`), the callback path
(`/oauth2/callback` or the path from `redirect_url`) must be covered by the same
path rule; otherwise the callback is proxied upstream instead of being handled by
ingress. Use host-level auth, or add a path rule for the callback path.

## OIDC Authentication

OpenID Connect (OIDC) authentication extends OAuth2 with identity verification.
The session flow reuses the OAuth2 engine, so the note about custom callback paths
applies here too: the path of `redirect_url` becomes the callback path (the default
`/oauth2/callback` stays accepted).

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
