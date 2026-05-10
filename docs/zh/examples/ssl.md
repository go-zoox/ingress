# SSL/TLS 示例

SSL/TLS 相关配置示例。

配置文件：[examples/ssl-tls/](https://github.com/go-zoox/ingress/tree/master/examples/ssl-tls)。

## 基本 HTTPS

<<< @/../examples/ssl-tls/https-basic.yaml

## 多域名

<<< @/../examples/ssl-tls/https-multi-domain.yaml

## Let's Encrypt

<<< @/../examples/ssl-tls/https-letsencrypt.yaml

## 带后端服务的 HTTPS

<<< @/../examples/ssl-tls/https-with-backends.yaml

## 全局 HTTP → HTTPS 强跳

配置了 `https.port` 时，可在**路由匹配之前**把明文 HTTP 强制跳到 HTTPS。使用 **`https.redirect_from_http`**（不要用 `rules[].backend.redirect` 代替全局强跳）：

<<< @/../examples/ssl-tls/https-global-redirect.yaml

可选字段（在自有配置里按需写上注释即可）：

- `with_origin_method_and_body`：`false` → 301/302 系列；`true` → 307/308
- `exclude_paths`：跳过强跳的精确路径列表

## 按路由重定向（`rules[].backend.redirect`）

当某个 **host 或 path** 需要直接返回跳转而不是反代时，使用 `backend.redirect`。**通常省略 `backend.type`**——仅在配置了 `redirect` 时会推断。**可运行对照：** **`examples/ssl-tls/route-redirect.yaml`** 用两个 host 分别演示 **`type: redirect`** 与省略；校验报告歧义时再显式写 **`backend.type: redirect`**。详见 [路由](/zh/guide/routing) 中 **`service`、`handler`、`redirect`** 与各配置块的对应关系。

<<< @/../examples/ssl-tls/route-redirect.yaml

在 `redirect.url` 中使用正则捕获占位（如 `$1`、`${path.1}`）的示例见 [重定向](./redirect)。

## 测试

### HTTPS 请求

```bash
curl https://example.com:8443/api
```

### 验证证书

```bash
openssl s_client -connect example.com:8443 -servername example.com
```

### 证书热加载

```bash
kill -HUP $(cat /tmp/gozoox.ingress.pid)
```
