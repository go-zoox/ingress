# SSL/TLS 配置

Ingress 支持 SSL/TLS 终止，允许您提供 HTTPS 流量并在代理级别终止 TLS 连接。

## HTTPS 配置

要启用 HTTPS，请在配置文件中配置 `https` 部分：

```yaml
version: v1
port: 8080

https:
  port: 8443
  ssl:
    - domain: example.com
      cert:
        certificate: /path/to/certificate.pem
        certificate_key: /path/to/private-key.pem
    - domain: api.example.com
      cert:
        certificate: /path/to/api-certificate.pem
        certificate_key: /path/to/api-private-key.pem
```

### 配置字段

| 字段 | 类型 | 描述 |
|------|------|------|
| `port` | int | 监听的 HTTPS 端口（默认：8443） |
| `enable_http3` | bool | 在已配置 TLS 时启用 HTTP/3（QUIC，UDP） |
| `http3_port` | int | HTTP/3 的 UDP 端口；省略或 `0` 时与 HTTPS 端口相同（TCP 与 UDP 可同时监听同端口号） |
| `http3_altsvc_max_age` | int | HTTPS 响应中 `Alt-Svc` 的 `ma=`（秒）；`0` 使用服务端默认；负数为不发送 `Alt-Svc` |
| `ssl` | array | SSL 证书配置数组 |

### SSL 证书配置

每个 SSL 条目需要：

| 字段 | 类型 | 描述 |
|------|------|------|
| `domain` | string | 证书的域名 |
| `cert.certificate` | string | 证书文件路径（PEM 格式） |
| `cert.certificate_key` | string | 私钥文件路径（PEM 格式） |

## HTTP/2 与 HTTP/3

- **TLS 上的 HTTP/2**：启用 HTTPS 后，服务端通过 ALPN（`h2`）与支持的客户端协商 HTTP/2，无需额外配置项。
- **明文 HTTP/2（h2c）**：在顶层设置 `enable_h2c: true`，可在明文 HTTP 的 `port` 上提供 h2c。仅适用于可信网络（例如终止 TLS 的上游反向代理与 Ingress 之间使用 h2c）。
- **HTTP/3**：设置 `https.enable_http3: true`。进程在 UDP 上监听 QUIC（默认与 `https.port` 相同，或通过 `https.http3_port` 指定）。HTTPS 响应可携带 `Alt-Svc` 以便客户端升级到 HTTP/3；用 `https.http3_altsvc_max_age` 调整或关闭。请在防火墙中放行对应 UDP 端口。

HTTP/3 示例：

```yaml
port: 8080

https:
  port: 443
  enable_http3: true
  # http3_port: 443
  # http3_altsvc_max_age: 86400
  ssl:
    - domain: example.com
      cert:
        certificate: /path/to/fullchain.pem
        certificate_key: /path/to/privkey.pem
```

若配置项未设置，zoox 也可能从环境变量读取 `ENABLE_H2C`、`ENABLE_HTTP3`、`HTTP3_PORT`、`HTTP3_ALTSVC_MAX_AGE`。

## 证书格式

Ingress 期望证书为 PEM 格式。证书和私钥都应为 PEM 格式：

```
-----BEGIN CERTIFICATE-----
...
-----END CERTIFICATE-----
```

```
-----BEGIN PRIVATE KEY-----
...
-----END PRIVATE KEY-----
```

## 多域名

您可以为不同的域名配置多个 SSL 证书：

```yaml
https:
  port: 8443
  ssl:
    - domain: example.com
      cert:
        certificate: /etc/ssl/example.com/fullchain.pem
        certificate_key: /etc/ssl/example.com/privkey.pem
    - domain: api.example.com
      cert:
        certificate: /etc/ssl/api.example.com/fullchain.pem
        certificate_key: /etc/ssl/api.example.com/privkey.pem
    - domain: admin.example.com
      cert:
        certificate: /etc/ssl/admin.example.com/fullchain.pem
        certificate_key: /etc/ssl/admin.example.com/privkey.pem
```

## 证书文件路径

证书文件可以使用以下方式指定：

- **绝对路径**：`/etc/ssl/example.com/cert.pem`
- **相对路径**：相对于 Ingress 启动时的工作目录

确保 Ingress 对证书文件具有读取权限。

## 使用 Let's Encrypt

您可以使用 Let's Encrypt 的证书。通常，Let's Encrypt 证书存储在如下位置：

```yaml
https:
  port: 8443
  ssl:
    - domain: example.com
      cert:
        certificate: /etc/letsencrypt/live/example.com/fullchain.pem
        certificate_key: /etc/letsencrypt/live/example.com/privkey.pem
```

## 证书重新加载

当您更新证书文件时，可以在不重启的情况下重新加载配置：

```bash
# 发送 SIGHUP 信号
kill -HUP $(cat /tmp/gozoox.ingress.pid)

# 或使用 reload 命令
ingress reload
```

Ingress 将从配置的路径重新加载证书。

## HTTP 到 HTTPS 重定向

要将 HTTP 流量重定向到 HTTPS，您可以配置重定向规则：

```yaml
rules:
  - host: example.com
    backend:
      redirect:
        url: https://example.com
        permanent: true
```

或在应用程序级别通过检查 `X-Forwarded-Proto` 头来处理。

## SNI（服务器名称指示）

Ingress 支持 SNI，允许在同一端口上为不同域名提供不同的证书。证书根据 TLS 握手中的域名选择。

## 后端通信

当 Ingress 终止 TLS 并转发到后端服务时：

- 后端服务可以使用 HTTP（不需要 TLS）
- 原始协议信息保存在 `X-Forwarded-Proto: https` 等头中
- 如果需要，后端服务仍可以使用 HTTPS

配置示例：

```yaml
https:
  port: 8443
  ssl:
    - domain: example.com
      cert:
        certificate: /path/to/cert.pem
        certificate_key: /path/to/key.pem

rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        protocol: http  # 后端使用 HTTP，TLS 在 Ingress 终止
```

## 安全最佳实践

1. **使用强密码套件**：确保您的证书使用强加密
2. **保持证书更新**：在过期前定期续订证书
3. **使用有效证书**：在生产环境中避免使用自签名证书
4. **保护私钥**：使用适当的权限保护私钥文件（例如，600）
5. **TLS 版本**：使用 TLS 1.2 或更高版本
6. **证书链**：在证书文件中包含完整的证书链
7. **监控过期**：设置证书过期警报

## 故障排除

### 证书未加载

- 验证证书文件路径是否正确
- 检查文件权限（Ingress 需要读取权限）
- 确保证书为 PEM 格式
- 检查证书文件语法

### 证书不匹配

- 验证证书中的域名是否与请求域名匹配
- 检查 SNI 是否正常工作
- 确保证书未过期

### 连接被拒绝

- 验证 HTTPS 端口是否已被使用
- 检查防火墙规则是否允许 HTTPS 端口的流量
- 确保 Ingress 在正确的端口上监听
