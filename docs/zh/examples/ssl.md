# SSL/TLS 示例

此页面提供 SSL/TLS 配置的示例。

## 基本 HTTPS 配置

```yaml
version: v1
port: 8080

https:
  port: 8443
  ssl:
    - domain: example.com
      cert:
        certificate: /etc/ssl/example.com/fullchain.pem
        certificate_key: /etc/ssl/example.com/privkey.pem
```

## 多域名

```yaml
version: v1
port: 8080

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

## Let's Encrypt 证书

```yaml
version: v1
port: 8080

https:
  port: 8443
  ssl:
    - domain: example.com
      cert:
        certificate: /etc/letsencrypt/live/example.com/fullchain.pem
        certificate_key: /etc/letsencrypt/live/example.com/privkey.pem
```

## 带后端服务的 HTTPS

```yaml
version: v1
port: 8080

https:
  port: 8443
  ssl:
    - domain: example.com
      cert:
        certificate: /etc/ssl/example.com/fullchain.pem
        certificate_key: /etc/ssl/example.com/privkey.pem

rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        protocol: http  # 后端使用 HTTP，TLS 在 Ingress 终止
```

## 全局 HTTP 到 HTTPS 强跳

配置了 `https.port` 时，可在**路由匹配之前**把明文 HTTP 强制跳到 HTTPS。使用 **`https.redirect_from_http`**（不要用 `rules[].backend.redirect` 代替全局强跳）：

```yaml
version: v1
port: 8080

https:
  port: 8443
  redirect_from_http:
    permanent: true
    # with_origin_method_and_body: false   # true -> 308/307，false -> 301/302
    # exclude_paths:
    #   - /healthz
  ssl:
    - domain: example.com
      cert:
        certificate: /etc/ssl/example.com/fullchain.pem
        certificate_key: /etc/ssl/example.com/privkey.pem

rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        protocol: http
```

## 按路由重定向（`rules[].backend.redirect`）

当某个 **host 或 path** 需要直接返回跳转而不是反代时，使用 `backend.redirect`。同一 **`backend` 下 `service` 与 `redirect` 互斥**；仅重定向时可不写 `service`。

```yaml
rules:
  - host: old.example.com
    backend:
      redirect:
        url: https://new.example.com
        permanent: true
        # with_origin_method_and_body: false
```

## 测试

### HTTPS 请求

```bash
curl https://example.com:8443/api
```

### 验证证书

```bash
openssl s_client -connect example.com:8443 -servername example.com
```

### 证书重新加载

更新证书后，重新加载配置：

```bash
kill -HUP $(cat /tmp/gozoox.ingress.pid)
```
