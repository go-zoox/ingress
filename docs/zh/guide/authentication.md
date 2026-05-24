# 认证

Ingress 支持多种认证方法来保护您的后端服务。您可以在规则级别或路径级别配置认证。

## 支持的认证方法

- **Basic 认证**：用户名/密码认证
- **Bearer Token**：基于令牌的认证
- **JWT**：JSON Web Token 认证
- **OAuth2**：OAuth 2.0 认证
- **OIDC**：OpenID Connect 认证

## Basic 认证

Basic 认证使用用户名和密码凭据。

### 单用户

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

### 多用户

您可以为 Basic 认证配置多个用户：

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

### 使用 Basic 认证

客户端必须在 `Authorization` 头中包含 base64 编码的凭据：

```bash
curl -u admin:admin123 http://example.com/api
```

或手动：

```bash
curl -H "Authorization: Basic $(echo -n 'admin:admin123' | base64)" http://example.com/api
```

## Bearer Token 认证

Bearer Token 认证使用基于令牌的认证。

### 单令牌

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

### 多令牌

您可以配置多个有效令牌：

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

### 使用 Bearer Token

客户端必须在 `Authorization` 头中包含 Bearer 令牌：

```bash
curl -H "Authorization: Bearer my-secret-token-123" http://example.com/api
```

## JWT 认证

JWT（JSON Web Token）认证使用密钥验证 JWT 令牌。

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

### 使用 JWT

客户端必须在 `Authorization` 头中包含有效的 JWT 令牌：

```bash
curl -H "Authorization: Bearer <jwt-token>" http://example.com/api
```

## OAuth2 认证

OAuth2 认证使用第三方身份提供商实现基于重定向的登录流程。
当用户访问受保护的路由且没有有效会话时，ingress 会将其重定向到提供商授权页面。
登录成功后，提供商重定向回 ingress，ingress 存储用户会话并将请求转发到上游服务。

### 支持的提供商

| 提供商 | 标识符 |
|--------|--------|
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

### 基础配置

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
            # scopes 可选，未配置时使用提供商默认值
            scopes:
              - user:email
```

**必填字段：**

| 字段 | 描述 |
|-------|-------------|
| `type` | 必须为 `oauth2` |
| `oauth2.provider` | 身份提供商名称（见上表） |
| `oauth2.client_id` | OAuth 应用 Client ID |
| `oauth2.client_secret` | OAuth 应用 Client Secret |

**可选字段：**

| 字段 | 默认值 | 描述 |
|-------|---------|-------------|
| `oauth2.scopes` | 按提供商决定（见下表） | 请求的 OAuth 权限范围 |
| `oauth2.redirect_url` | 从请求 host 自动生成 | 自定义回调 URL |

**各提供商默认 scopes：**

| 提供商 | 默认 scopes |
|--------|------------|
| GitHub | `user:email` |
| GitLab | `read_user` |
| Google | `openid profile email` |
| Microsoft | `openid profile email` |
| Feishu | `user:email` |
| Slack | `users:read` |
| Auth0 | `openid profile email` |
| Okta | `openid profile email` |
| Doreamon / Kakao | *(提供商默认)* |

回调路径固定为 `/oauth2/callback`。完整的回调 URL 从当前请求的 scheme 和 host 自动生成，
例如 `http://example.com/oauth2/callback`。

### Connect JWT Headers

当 `connect.enabled` 为 `true` 时，OAuth2 认证成功后，ingress 会将用户身份信息注入到
上游请求的 header 中。这使得后端服务无需重复实现 OAuth2 流程即可识别认证用户。

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
                algorithm: hs256       # 可选，默认: hs256
                expires_in: "5m"       # 可选，默认: 5m
```

**注入的 Headers：**

| Header | 描述 |
|--------|-------------|
| `X-Connect-Token` | 包含 `id`、`username`、`email`、`nickname`、`avatar` 的 JWT |
| `X-Connect-Timestamp` | 令牌签发时间的 Unix 毫秒时间戳 |

**上游服务解码 JWT（Go 示例）：**

```go
import "github.com/go-zoox/jwt"

token := r.Header.Get("X-Connect-Token")
j := jwt.New("your-shared-jwt-secret")
claims, err := j.Verify(token)
if err != nil {
    // 令牌无效
}
userID := claims.Get("id").String()
```

### 认证流程

1. 客户端访问 `https://example.com/protected`
2. Ingress 检测到无会话 → 生成 CSRF state → 重定向到提供商登录页
3. 用户在提供商处完成认证
4. 提供商重定向到 `https://example.com/oauth2/callback?code=xxx&state=yyy`
5. Ingress 验证 state，用 code 换取 token，获取用户信息
6. 用户信息存储在加密的 session cookie 中
7. 客户端被重定向回原始 URL（`/protected`）
8. 后续请求中，session cookie 标识已认证的用户
9. 若 `connect.enabled: true`，在向上游转发请求时注入 `X-Connect-Token` 和 `X-Connect-Timestamp`

## OIDC 认证

OpenID Connect (OIDC) 认证扩展了 OAuth2，增加了身份验证。

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

## 路径级认证

您可以为不同路径配置不同的认证方法：

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

在此示例中：
- 对 `/admin` 的请求需要管理员凭据
- 对 `/api` 的请求需要 bearer 令牌
- 所有其他请求使用默认的 basic 认证

## 认证流程

1. 客户端向 Ingress 发送请求
2. Ingress 检查匹配的规则/路径是否需要认证
3. 如果需要认证：
   - Ingress 验证凭据/令牌
   - 如果有效，请求被转发到后端
   - 如果无效，Ingress 返回 401 Unauthorized 响应
4. 如果不需要认证，请求直接转发

## 安全最佳实践

1. **使用 HTTPS**：启用认证时始终使用 SSL/TLS
2. **强密码**：为 Basic 认证使用强且唯一的密码
3. **安全令牌**：为 Bearer 认证生成安全、随机的令牌
4. **令牌轮换**：定期轮换令牌并更新配置
5. **密钥管理**：安全存储密钥，避免在配置文件中硬编码
6. **最小权限**：授予用户最小必要的访问权限
7. **审计日志**：监控认证尝试和失败

## 故障排除

### 401 Unauthorized

- 验证凭据/令牌是否正确
- 检查 `Authorization` 头格式是否正确
- 确保认证类型与配置匹配

### 认证不工作

- 验证认证配置是否正确
- 检查规则/路径是否匹配请求
- 确保认证类型受支持
