# 基础设置

此示例展示了一个简单的反向代理设置的基本 Ingress 配置。

## 配置

```yaml
version: v1
port: 8080

rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
```

## 说明

- **port**: Ingress 监听端口 8080
- **rules**: 定义路由规则
- **host**: 匹配 `Host: example.com` 的请求
- **backend.service**: 路由到端口 8080 上的 `backend-service`

## 测试

启动 Ingress：

```bash
ingress run -c ingress.yaml
```

测试设置：

```bash
curl -H "Host: example.com" http://localhost:8080
```

## 多个服务

您可以配置多个服务：

```yaml
version: v1
port: 8080

rules:
  - host: web.example.com
    backend:
      service:
        name: web-service
        port: 8080
  - host: api.example.com
    backend:
      service:
        name: api-service
        port: 8081
```
