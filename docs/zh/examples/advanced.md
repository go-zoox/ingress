# 高级示例

进阶路由、重写与健康检查等示例。

配置文件：[examples/advanced/](https://github.com/go-zoox/ingress/tree/master/examples/advanced)。

## 正则主机与路径重写

<<< @/../examples/advanced/regex-host-path.yaml

此示例：

- 用正则匹配如 `t-myapp.example.work` 的主机
- 在 `service.name` 中使用 `${host.<索引>}`、`${path.<索引>}` 等作用域捕获
- 对特定 API 路径做重写

## 通配符主机

<<< @/../examples/advanced/wildcard.yaml

匹配 `example.work` 的任意子域。

## 复杂路径重写

<<< @/../examples/advanced/complex-path-rewrite.yaml

对第三方 HTTPS 上游在 **`backend.service`** 下使用 **`mode: external`**，无需再写 **`request.host.rewrite`** 即可让 **`Host`** 与各自 **`service.name`** 一致。

**`service.mode`**（internal / external）与 handler 路径示例：

<<< @/../examples/advanced/service-mode-external-mixed.yaml

## 多服务与健康检查

<<< @/../examples/advanced/health-checks.yaml

## HTTP 响应缓存（`backend.cache`）

按 backend 缓存 **service / handler / redirect** 的响应，与顶层 **`cache`** 共用 `ctx.Cache()`。语义、**`skip_vary`** 与 httpbin **`Vary`** 说明见[缓存指南](/zh/guide/caching#http-响应缓存-backendcache)。

<<< @/../examples/advanced/http-response-cache.yaml

## 应用缓存引擎（内存 / Redis）

顶层 **`cache`** 决定匹配器等数据用 Redis 还是内存；**HTTP 响应**条目在开启 **`backend.cache`** 时也写入同一后端。下例在 **service** 上启用 **`backend.cache`**，多实例时可共享缓存。

<<< @/../examples/advanced/redis-cache.yaml

## 综合示例

包含 HTTPS、缓存、健康检查、fallback、认证与路径规则的合成示例：

<<< @/../examples/advanced/full-stack.yaml
