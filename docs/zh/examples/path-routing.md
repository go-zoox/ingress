# 基于路径的路由

此示例演示如何把不同路径路由到不同后端。

配置文件：[examples/path-routing/](https://github.com/go-zoox/ingress/tree/master/examples/path-routing)。

## 基本路径路由

<<< @/../examples/path-routing/basic-paths.yaml yaml

## 说明

- 对 `example.com/api/*` 的请求路由到 `api-service`
- 对 `example.com/admin/*` 的请求路由到 `admin-service`
- 对 `example.com` 的其他请求路由到 `default-service`

## 复杂路径路由

<<< @/../examples/path-routing/complex-paths.yaml yaml

## Docker Registry 示例

<<< @/../examples/path-routing/docker-registry.yaml yaml

## 测试

```bash
curl -H "Host: example.com" http://localhost:8080/
curl -H "Host: example.com" http://localhost:8080/api/users
curl -H "Host: example.com" http://localhost:8080/admin/dashboard
```
