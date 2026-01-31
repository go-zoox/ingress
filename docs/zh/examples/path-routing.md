# 基于路径的路由

此示例演示基于路径的路由，将不同路径路由到不同的后端服务。

## 基本路径路由

```yaml
version: v1
port: 8080

rules:
  - host: example.com
    backend:
      service:
        name: default-service
        port: 8080
    paths:
      - path: /api
        backend:
          service:
            name: api-service
            port: 8080
      - path: /admin
        backend:
          service:
            name: admin-service
            port: 8080
```

## 说明

- 对 `example.com/api/*` 的请求路由到 `api-service`
- 对 `example.com/admin/*` 的请求路由到 `admin-service`
- 对 `example.com` 的所有其他请求路由到 `default-service`

## 复杂路径路由

```yaml
version: v1
port: 8080

rules:
  - host: example.com
    backend:
      service:
        name: web-service
        port: 8080
    paths:
      - path: /api/v1
        backend:
          service:
            name: api-v1-service
            port: 8080
      - path: /api/v2
        backend:
          service:
            name: api-v2-service
            port: 8080
      - path: /static
        backend:
          service:
            name: static-service
            port: 8080
```

## Docker Registry 示例

此示例展示 Docker registry 的基于路径的路由：

```yaml
version: v1
port: 8080

rules:
  - host: docker-registry.example.com
    backend:
      service:
        name: docker-registry
        port: 80
    paths:
      - path: /v2
        backend:
          service:
            name: docker-registry-v2
            port: 80
```

## 测试

测试不同的路径：

```bash
# 路由到 default-service
curl -H "Host: example.com" http://localhost:8080/

# 路由到 api-service
curl -H "Host: example.com" http://localhost:8080/api/users

# 路由到 admin-service
curl -H "Host: example.com" http://localhost:8080/admin/dashboard
```
