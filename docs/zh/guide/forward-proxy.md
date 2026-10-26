# 正向代理（CONNECT + absolute-form）

Ingress 可以终止 TLS，作为 **HTTP 正向代理**使用，并支持两种标准代理请求：

- `https://target` → `CONNECT target:443`
- `http://target` → `GET http://target/...`（absolute-form）

代理请求的 HTTP `Host` 是目标站，因此 ingress 使用 **TLS SNI** 选择路由。

## 配置

```yaml
https:
  port: 443
  ssl:
    - domain: proxy.example.com
      cert:
        certificate: /etc/ssl/proxy.example.com.pem
        certificate_key: /etc/ssl/proxy.example.com.key.pem

rules:
  - host: proxy.example.com
    backend:
      service:
        name: anchor-frontend
        port: 8080
        protocol: http
    connect_proxy:
      enabled: true
      upstream:
        name: anchor-backend
        port: 3128
        protocol: http
      allowed_targets:
        - '^.+:(80|443)$'
      connect_timeout: 10
      response_timeout: 10
      idle_timeout: 600
      max_connections: 500
```

`connect_proxy.upstream` 与普通 `backend` 独立。未配置 `upstream` 时，为兼容旧配置会回退使用该规则的 `backend.service`。

## 认证

Ingress 会透传客户端的 `Proxy-Authorization` 给代理上游。代理上游（Anchor）校验账号；失败时返回 `407 Proxy Authentication Required`，ingress 原样转发状态、`Proxy-Authenticate` 和响应体。Ingress 不保存代理密码。

## 目标白名单

`allowed_targets` 是可选的完整字符串正则列表，匹配规范化后的 `host:port`：

- CONNECT：CONNECT authority（例如 `example.com:443`）
- HTTP absolute-form：`host:port`；未写端口时补 `:80`

空列表表示不限制目标。不匹配的目标直接返回 `403`，不会连接代理上游。

> 建议始终使用 `^` 和 `$`。匹配的是完整 authority，`^example\.com:443$` 不会匹配 `evil-example.com:443`。

## 资源限制

- `connect_timeout`：拨号和写 CONNECT 的超时，默认 10 秒。
- `response_timeout`：读取上游 CONNECT 响应的超时，默认 10 秒。
- `idle_timeout`：隧道空闲超时，默认 600 秒。
- `max_connections`：该路由并发代理请求上限；`0` 表示不限制。

代理请求同时使用规则/全局 `rate_limit`，并写入访问日志。

## 使用示例

```bash
# HTTPS 目标（CONNECT）
curl -x https://proxy.example.com \
  -U 'mp_user:password' \
  https://httpbin.org/ip

# HTTP 目标（absolute-form）
curl -x https://proxy.example.com \
  -U 'mp_user:password' \
  http://httpbin.org/get
```

TLS 必须在 ingress 终止。如果 ingress 前面还有一层会终止 TLS 的负载均衡，ingress 看不到 SNI 和 CONNECT，此功能不可用。

## 管理后台

打开 **配置 → 路由规则**，编辑目标规则，在左侧分区选择 **正向代理**。表单可编辑独立上游、目标白名单、超时和连接上限；普通 Backend 仍在 **Backend** 分区。规则列表摘要会显示 `PROXY → upstream`。
