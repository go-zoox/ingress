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

OAuth2 认证支持 OAuth 2.0 流程。

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
