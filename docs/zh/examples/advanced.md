# 高级示例

进阶路由、重写与健康检查等示例。

配置文件：[examples/advanced/](https://github.com/go-zoox/ingress/tree/master/examples/advanced)。

## 正则主机与路径重写

<<< @/../examples/advanced/regex-host-path.yaml yaml

此示例：

- 用正则匹配如 `t-myapp.example.work` 的主机
- 在 `service.name` 中使用 `${host.<索引>}`、`${path.<索引>}` 等作用域捕获
- 对特定 API 路径做重写

## 通配符主机

<<< @/../examples/advanced/wildcard.yaml yaml

匹配 `example.work` 的任意子域。

## 复杂路径重写

<<< @/../examples/advanced/complex-path-rewrite.yaml yaml

## 多服务与健康检查

<<< @/../examples/advanced/health-checks.yaml yaml

## Redis 缓存

<<< @/../examples/advanced/redis-cache.yaml yaml

## 综合示例

包含 HTTPS、缓存、健康检查、fallback、认证与路径规则的合成示例：

<<< @/../examples/advanced/full-stack.yaml yaml
