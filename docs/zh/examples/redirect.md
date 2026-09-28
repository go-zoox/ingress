# 重定向示例

通常 **省略 `backend.type`**：仅在配置了 **`backend.redirect`** 时会推断为 **`redirect`**。**`examples/ssl-tls/route-redirect.yaml`** 用两个 host 分别演示 **`type: redirect`** 与省略。**`examples/redirect/capture-and-mixed.yaml`** 在同一文件里混用「部分 backend 显式写 **`backend.type`**、部分省略」。若 **`ingress validate`** 提示歧义再显式写 **`backend.type: redirect`**。不要在**同一个** `backend` 上同时配置 **`backend.service`** / **`backend.handler`** 与 **`backend.redirect`**——同一 host 需要反代与跳转时请拆到不同 **`paths`**（见下例）。

全局 HTTP→HTTPS（在路由之前）请用 `https.redirect_from_http`，见 [SSL/TLS](./ssl)。

配置文件：[examples/redirect/](https://github.com/go-zoox/ingress/tree/master/examples/redirect)；最简单的 host 级跳转见 [route-redirect.yaml](https://github.com/go-zoox/ingress/blob/master/examples/ssl-tls/route-redirect.yaml)。

## 跳转状态码（`duration` + `preserve_request`）

用 **`duration`**（`temporary` | `permanent`）和 **`preserve_request`**（`true` | `false`）选择 **302 / 301 / 307 / 308**，无需死记数字。旧字段 **`permanent`** / **`with_origin_method_and_body`** 仍兼容，请优先用新字段。

| duration | preserve_request | 状态码 |
|----------|------------------|--------|
| temporary | false | 302 |
| permanent | false | 301 |
| temporary | true | 307 |
| permanent | true | 308 |

<<< @/../examples/redirect/redirect-status.yaml

示例中每个 host 在 GET 时返回不同状态码。POST/API 场景若需保留方法与 body，请设 **`preserve_request: true`**（307/308）。

## 路径正则 redirect

`paths[].path` 为 Go 正则（编译时隐式加前缀 `^`）。path 级 **`backend.redirect`** 与 service/handler 共用同一套匹配；在 `redirect.url` 中用 **`${path.N}`** 引用捕获组，需要精确匹配时在 pattern 末尾加 **`$`**。Path 级 `url` 默认**按配置原样使用**（不追加请求的 path/query），需要保留时在该 path backend 上写 **`preserve_path: true`**；host 级 redirect 始终保留。

<<< @/../examples/redirect/path-regex.yaml

`/legacy/` 为前缀匹配并跳到固定 URL；`/root-only/` 演示 path 级默认行为（`Location` 就是配置的 `url`）；`/mirror/` 用 `preserve_path: true` 找回保留语义；`/go/([^/]+)$` 用 `${path.1}` 展开；`/promo$` 仅匹配 `/promo` 并跳到明确 landing path。未命中任何 path 时回退到 host 级 redirect（保留 path 与 query）。

## 正则 host：`redirect.url` 中的捕获

与 `service.name` 相同的占位规则：`$1`、`${host.1}` 等。

下例同时演示：正则 host 重定向、host 默认重定向 + 路径反代、以及路径捕获 `${path.N}`：

<<< @/../examples/redirect/capture-and-mixed.yaml

### 每条规则在演示什么

1. **正则 host**：`^bigscreen-([^.]+)\.example\.com$`，在 `redirect.url` 里使用 `$1`。
2. **Host 默认重定向 + 路径服务**：未匹配的 path 走跳转；匹配 `^/api/` 的走后端。
3. **`${path.N}`**：路径正则捕获填入 `redirect.url`。
