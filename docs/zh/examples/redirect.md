# 重定向示例

通常 **省略 `backend.type`**：仅在配置了 **`backend.redirect`** 时会推断为 **`redirect`**。**`examples/ssl-tls/route-redirect.yaml`** 用两个 host 分别演示 **`type: redirect`** 与省略。**`examples/redirect/capture-and-mixed.yaml`** 在同一文件里混用「部分 backend 显式写 **`backend.type`**、部分省略」。若 **`ingress validate`** 提示歧义再显式写 **`backend.type: redirect`**。不要在**同一个** `backend` 上同时配置 **`backend.service`** / **`backend.handler`** 与 **`backend.redirect`**——同一 host 需要反代与跳转时请拆到不同 **`paths`**（见下例）。

全局 HTTP→HTTPS（在路由之前）请用 `https.redirect_from_http`，见 [SSL/TLS](./ssl)。

配置文件：[examples/redirect/](https://github.com/go-zoox/ingress/tree/master/examples/redirect)；最简单的 host 级跳转见 [route-redirect.yaml](https://github.com/go-zoox/ingress/blob/master/examples/ssl-tls/route-redirect.yaml)。

## 正则 host：`redirect.url` 中的捕获

与 `service.name` 相同的占位规则：`$1`、`${host.1}` 等。

下例同时演示：正则 host 重定向、host 默认重定向 + 路径反代、以及路径捕获 `${path.N}`：

<<< @/../examples/redirect/capture-and-mixed.yaml

### 每条规则在演示什么

1. **正则 host**：`^bigscreen-([^.]+)\.example\.com$`，在 `redirect.url` 里使用 `$1`。
2. **Host 默认重定向 + 路径服务**：未匹配的 path 走跳转；匹配 `^/api/` 的走后端。
3. **`${path.N}`**：路径正则捕获填入 `redirect.url`。
