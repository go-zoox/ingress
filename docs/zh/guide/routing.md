# 路由

Ingress 提供灵活的路由功能来匹配请求并将它们路由到相应的后端服务。您可以按主机名和路径匹配请求，支持精确、正则表达式和通配符匹配。

## 主机匹配

主机匹配是路由请求的主要方式。Ingress 支持三种类型的主机匹配：

### 精确匹配

精确匹配（默认）完全匹配主机名：

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
```

这将精确匹配 `Host: example.com` 的请求。

### 正则表达式匹配

正则表达式匹配允许您使用正则表达式来匹配主机名：

```yaml
rules:
  - host: ^t-(\w+).example.work
    host_type: regex
    backend:
      service:
        name: task.$1.svc
        port: 8080
```

在此示例中，`$1` 引用正则表达式模式中的第一个捕获组。对 `t-myapp.example.work` 的请求将被路由到 `task.myapp.svc`。

### 通配符匹配

通配符匹配使用 `*` 作为通配符：

```yaml
rules:
  - host: '*.example.work'
    host_type: wildcard
    backend:
      service:
        name: wildcard-service
        port: 8080
```

这将匹配 `example.work` 的任何子域，例如 `app.example.work`、`api.example.work` 等。

## 基于路径的路由

您可以在主机规则内定义基于路径的路由规则。路径使用正则表达式模式匹配：

```yaml
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

路径匹配使用正则表达式模式。路径 `/api` 将匹配 `/api`、`/api/`、`/api/users` 等。

### 路径匹配优先级

路径按定义的顺序匹配。将使用第一个匹配的路径。如果没有路径匹配，将使用主机级后端。

## 请求重写

在路由到后端服务时，您可以重写请求路径、头和查询参数。

### 路径重写

使用正则表达式模式重写请求路径：

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        request:
          path:
            rewrites:
              - ^/api/v1/(.*):/api/v2/$1
              - ^/old:/new
```

重写格式为 `pattern:replacement`，其中 `pattern` 是正则表达式，`replacement` 是新路径。可以使用 `$1`、`$2` 等引用捕获组。

### Host 头重写

重写发送到后端的 Host 头：

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        request:
          host:
            rewrite: true
```

当 `rewrite: true` 时，Host 头将设置为后端服务名称和端口。

### 头修改

添加或修改请求头：

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        request:
          headers:
            X-Forwarded-Proto: https
            X-Custom-Header: value
```

### 查询参数修改

添加或修改查询参数：

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        request:
          query:
            api_key: secret-key
            version: v2
```

## 重定向

除了代理到后端服务，您还可以重定向请求：

```yaml
rules:
  - host: old.example.com
    backend:
      redirect:
        url: https://new.example.com
        permanent: true
```

- `url`: 重定向目标 URL
- `permanent`: 如果为 `true`，返回 301 重定向；如果为 `false`，返回 302 重定向

## 回退服务

如果没有规则匹配请求，则使用回退服务：

```yaml
fallback:
  service:
    name: fallback-service
    port: 8080
```

回退服务对于处理未匹配的请求或提供默认后端很有用。

## 路由示例

### 同一主机上的多个服务

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: web-service
        port: 8080
    paths:
      - path: /api
        backend:
          service:
            name: api-service
            port: 8081
      - path: /admin
        backend:
          service:
            name: admin-service
            port: 8082
```

### 带路径重写的正则主机

```yaml
rules:
  - host: ^t-(\w+).example.work
    host_type: regex
    backend:
      service:
        name: task.$1.svc
        port: 8080
    paths:
      - path: /api/v1/([^/]+)
        backend:
          service:
            name: $1.example.work
            port: 8080
            request:
              path:
                rewrites:
                  - ^/api/v1/([^/]+):/api/v1/task/$1
```

### 通配符主机匹配

```yaml
rules:
  - host: '*.example.work'
    host_type: wildcard
    backend:
      service:
        name: wildcard-service
        port: 8080
```

这匹配 `example.work` 的任何子域并路由到同一个后端服务。

## 最佳实践

1. **顺序很重要**：将更具体的规则放在通用规则之前
2. **尽可能使用精确匹配**：比正则表达式或通配符匹配更快
3. **测试正则表达式模式**：确保您的正则表达式模式按预期匹配
4. **使用路径路由**：按路径组织路由以提高可维护性
5. **设置回退**：始终为未匹配的请求配置回退服务
