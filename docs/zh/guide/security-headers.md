# 安全响应头

Ingress 在路由匹配后，可按**预设 profile** 自动添加 HTTP 安全响应头：HSTS、`X-Frame-Options`、`X-Content-Type-Options`、`Referrer-Policy`、`Content-Security-Policy`，以及 **CORS**（含 **OPTIONS 预检**）。

全局基线写在 **`security:`**，Host 级覆盖写在 **`rules[].security`**，Path 级覆盖写在 **`paths[].security`**（字段逐层合并：全局 → Host → Path）。

## 预设 profile

| Profile | 场景 | HSTS | Frame | CORS |
|---------|------|------|-------|------|
| `strict` | 通用 Web / 管理后台 | auto | DENY | 关 |
| `api` | REST / JSON API | auto | DENY | 开（需配置 `cors.origins`） |
| `embeddable` | 允许 iframe 嵌入 | auto | SAMEORIGIN | 关 |
| `off` | 关闭 | — | — | — |

**HSTS `auto`** 仅在 HTTPS 请求时下发（直连 TLS 或 `X-Forwarded-Proto: https`）。

## 示例

```yaml
security:
  profile: strict

rules:
  - host: api.example.com
    security:
      profile: api
      cors:
        origins:
          - https://portal.example.com
        credentials: true
    backend:
      service:
        name: api
        port: 8080
```

可运行样例：[`examples/security/profiles.yaml`](../../examples/security/profiles.yaml)。

## 字段

| 字段 | 说明 |
|------|------|
| `profile` | `strict` / `api` / `embeddable` / `off` |
| `hsts` | `auto`（默认）、`on`、`off` |
| `frame` | `inherit`、`deny`、`sameorigin`、`off` |
| `content_type_options` | 是否发送 `nosniff` |
| `referrer_policy` | Referrer-Policy 值；`off` 关闭 |
| `csp` | CSP 策略；`off` 关闭 |
| `cors.enabled` | 显式开关 |
| `cors.origins` | 允许的 Origin（启用 CORS 时必填） |
| `cors.methods` | 默认 GET, POST, PUT, PATCH, DELETE, OPTIONS |
| `cors.headers` | 默认 Authorization, Content-Type, Accept, X-Requested-With |
| `cors.credentials` | 是否允许携带 Cookie |
| `cors.max_age` | 预检缓存秒数（默认 86400） |

## 生效范围

- 对 **service / handler / redirect**、WAF 拦截、限流 429、部分错误页均会尝试附加安全头。
- **`backend.service.response.headers`** 与 handler 头先写入；安全头在其后补充（同名字段不覆盖已有值）。
- 未匹配路由（404）仅应用**全局** `security:`。

## Admin 控制台

在配置模块 **「安全」** 中编辑全局 profile；Host / Path 级在规则编辑器侧边栏 **「安全」** 中覆盖，或在 YAML 模式设置 `rules[].security` / `paths[].security`。

英文说明：[Security headers (EN)](../../guide/security-headers.md)。
