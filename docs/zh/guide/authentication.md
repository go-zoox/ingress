# 认证

Ingress 支持多种认证方法来保护您的后端服务。您可以在规则级别或路径级别配置认证。

## 支持的认证方法

- **Basic 认证**：用户名/密码认证
- **Bearer Token**：基于令牌的认证
- **JWT**：JSON Web Token 认证
- **OAuth2**：OAuth 2.0 认证
- **OIDC**：OpenID Connect 认证

## 启用/禁用认证

配置 `auth.type` 后，认证默认启用。您可以使用 `enabled` 字段临时禁用认证（如调试、灰度发布等），而无需删除认证配置：

```yaml
auth:
  enabled: false   # 设为 true 重新启用；省略时若 type 已设置则默认启用
  type: basic
  basic:
    users:
      - username: admin
        password: admin123
```

| `enabled` 值 | `type` 是否设置 | 行为 |
|--------------|---------------|------|
| *(省略)* | 否 | 无认证（默认） |
| *(省略)* | 是 | 认证**启用**（默认） |
| `true` | 任意 | 认证**启用** |
| `false` | 任意 | 认证**禁用**（配置保留） |

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
| `oauth2.redirect_url` | 从请求 host 自动生成（`/oauth2/callback`） | 自定义回调 URL；其**路径**同时作为回调路径（见下） |

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

回调路径默认是 `/oauth2/callback`。显式设置 `oauth2.redirect_url` 时，**其路径也会被当作回调路径**，
因此像 Doreamon 这种要求特定回调地址（`/login/doreamon/callback`）的提供方也能正常工作：

```yaml
auth:
  type: oauth2
  oauth2:
    provider: doreamon
    client_id: ...
    client_secret: ...
    redirect_url: https://example.com/login/doreamon/callback
```

该回调请求必须被路由到**使用同一份 `auth.oauth2` 配置**的服务（host 级 backend，或覆盖该回调路径的 path 规则），
并且该 URL 必须登记在身份提供方的回调地址白名单里（Doreamon 会对 `redirect_uri` 做精确匹配）。

未设置 `oauth2.redirect_url` 时，回调 URL 从当前请求的 scheme 和 host 自动生成，
例如 `http://example.com/oauth2/callback`。

回调路径是 OAuth2 流程专用的：不带 `code`/`state` 的请求访问该路径会返回 **401**，
因此请把它指向后端服务不需要的路径。

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

#### 负载均衡后的 HTTP → HTTPS

OAuth2 的 **`redirect_uri`** 根据请求的 scheme 和 host 自动生成（`auth_oauth2.go`）。若 ingress 位于 TLS 终结的反向代理之后（内部收到的是明文 HTTP），它会根据 **`X-Forwarded-Proto: https`** 判定 scheme。对于公网/多副本部署，建议显式设置 **`oauth2.redirect_url`** 为公网 HTTPS 地址，确保身份提供方始终看到同一个回调地址：

```yaml
auth:
  type: oauth2
  oauth2:
    provider: github
    client_id: ...
    client_secret: ...
    redirect_url: https://example.com/oauth2/callback
```

如果提供方要求非默认回调路径，把该路径写进 `redirect_url` 即可：它会在默认 `/oauth2/callback`
**之外**再增加一个回调路径（两者都接受，因此切换配置时在途登录不会中断）。

#### 多副本部署

OAuth2/OIDC 的 **CSRF `state`** 与 **登录后跳转 URL** 都存在**加密的 session cookie** 中，而该 cookie 是用 `app.Config.SecretKey` 加密的。当你没有配置 `SecretKey` 时，会退化为**每个进程随机生成**的密钥。如果**多副本 + 负载均衡**部署，回调请求可能落到与发起跳转不同的副本上：该副本无法解密 cookie，CSRF `state` 为空，回调返回 **401**。

要让 OAuth2 在多副本下正常工作，请在**每个副本**上设置相同的会话密钥：

```yaml
# ingress.yaml
secret_key: a-long-random-shared-secret
```

或

```bash
# SECRET_KEY 环境变量的优先级高于配置
export SECRET_KEY="a-long-random-shared-secret"
```

可用 `openssl rand -hex 32` 生成一个。由于会话完全存储在 cookie 中，密钥稳定后**无需**开启 sticky session。

未配置 `secret_key`/`SECRET_KEY` 时，ingress 启动会打印
`session key: no secret_key/SECRET_KEY configured, using a per-process key …`，
提示该部署每次重启都会让用户重新登录。

管理控制台使用**独立**的 session cookie（`ingress_admin_session`），不再与网关共用 `gsession`；
两者仍然使用同一个密钥（`secret_key`/`SECRET_KEY`），即共享密钥但不共享 cookie 负载。
升级后网关的 OAuth2 会话不受影响（cookie 名与密钥不变），但管理控制台用户需要重新登录一次（cookie 重命名）。

#### 并发登录

`state`/跳转地址按**每次登录流程**（以 `state` 为键）分别记录，因此同一用户同时发起两次登录（多标签页、刷新等）不再互相覆盖：每个回调各自解析自己的 `state`，并跳回各自原始的 URL。同时保留旧的单值 `state` 回退逻辑，保证滚动升级过程中旧的在途流程仍可用。

#### CSRF state 丢失与 ~4KB session cookie 上限

CSRF `state`、登录后跳转 URL 和用户信息都放在同一个**客户端**加密 session cookie（`gsession`）里。
浏览器对单个 cookie 有约 **4096 字节**的上限，超限的 `Set-Cookie` 会被**静默丢弃**。一旦发生，
新的 `state` 再也存不进去，之后每个回调都会 **401**，而且不会自行恢复。这正是“重启 ingress 才能恢复”的原因：
未配置 `secret_key` 时会话密钥是**每进程随机**的，重启让旧 cookie 无法解密，浏览器才会接受一个新的小 cookie
（代价是所有人都被登出）。

现在 ingress 会限制 OAuth2 会话负载（最多 8 个在途流程、按最先入先淘汰、跳转 URL 截断、不保存 access token 与 permissions/groups），
并在回调 `state` 找不到时**自动恢复**：删除过期的网关 session cookie（只回一个很小的 `Set-Cookie`，
浏览器会立刻接受，而不是继续拒绝那个超限值），然后重定向到 `/?_ingress_oauth2_retry=1` 重新发起一次登录。
重试登录的 CSRF state 会带 `r1.` 前缀；再次失败则直接返回 **401**，不会陷入
`/authorize` ⇄ `/callback` 的无限重定向。限制同时使用两个标记——重试 query/state 前缀，以及
60 秒有效期的 `ingress_oauth2_retry` cookie（覆盖会丢掉 query 的重定向链）——因此不依赖某个可能被浏览器丢弃的 cookie。

关于自动恢复的几点说明：

- 会删除整个网关 OAuth2 session cookie，因此其它标签页中在途的登录也会丢失（重新发起即可）；
- 会跳转到 `/`，不保留原始深链；
- 若 `/` 本身未受保护，这次跳转相当于一次空页面加载，但过期 cookie 已被删除，
  用户下一次访问受保护路径即可正常登录。

**多副本**部署仍必须共享同一个 `secret_key`：否则自动重试不可能成功，用户会在一次重试后得到 401。

#### 排障：OAuth2 callback 返回 401

查看 ingress 日志中的 `oauth2 authentication failed for host: …`：

| 日志信息 | 原因 / 处理 |
|---------|-------------|
| `state mismatch …` | 浏览器 cookie 中没有对应流程：来自其它副本/进程的 cookie、cookie 超限被浏览器丢弃、或从历史记录打开了旧的 callback。ingress 会自动恢复一次（见上）；自动重试后仍失败时日志里会再次出现 `state mismatch …`，此时检查 `secret_key`（多副本）并清理站点 cookie。 |
| `identity provider returned "…"` | 提供方拒绝了授权请求（`access_denied`、scope 不合法、或 `redirect_uri` 未登记）。修改提供方的应用配置。 |
| `exchange failed: …` | 调用提供方 `/token` 或 `/user` 失败（client secret 错误、提供方不可达/TLS 问题）。现在**不会在换取 token 之前消费 flow**，重新加载该 callback URL 即可重试；但若 code 已在 `/token` 被兑换（授权码一次性），则需重新发起登录。 |
| `missing code or state` | 回调请求没有携带 `code`/`state`（重定向被截断、手工修改 URL，或提供方返回了不带 `code` 的错误）。 |
| `exchange panic: …` | 提供方 `/token` 返回了非预期负载（go-zoox/oauth2 的空指针问题，例如 HTTP 200 且 JSON 带 `code`）。检查提供方的 token 接口；该请求会返回 401 而不是 500。 |

使用**路径级认证**（`paths[].backend.service.auth`）时，回调路径（`/oauth2/callback` 或 `redirect_url` 的路径）
必须被同一条 path 规则覆盖，否则该回调会被转发给上游而不是由 ingress 处理。可改用 host 级认证，或为回调路径单独加一条 path 规则。

## OIDC 认证

OpenID Connect (OIDC) 认证扩展了 OAuth2，增加了身份验证。
OIDC 的会话流程复用同一套 OAuth2 引擎，因此上面的“自定义回调路径”结论同样适用：
`redirect_url` 的路径会成为回调路径（默认 `/oauth2/callback` 仍然接受）。

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
