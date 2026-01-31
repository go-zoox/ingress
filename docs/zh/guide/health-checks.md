# 健康检查

Ingress 支持两个级别的健康检查：外部健康检查（用于 Ingress 服务本身）和内部健康检查（用于后端服务）。

## 外部健康检查

外部健康检查允许外部系统检查 Ingress 是否正在运行且健康。

### 配置

```yaml
healthcheck:
  outer:
    enable: true
    path: /healthz
    ok: true
```

### 配置字段

| 字段 | 类型 | 描述 | 默认值 |
|------|------|------|--------|
| `enable` | bool | 启用外部健康检查 | `false` |
| `path` | string | 健康检查端点路径 | `/healthz` |
| `ok` | bool | 始终返回 OK 状态 | `false` |

### 使用

启用后，Ingress 在配置的路径响应健康检查请求：

```bash
curl http://localhost:8080/healthz
```

如果 `ok: true`，端点始终返回成功响应。否则，它可能根据内部检查返回实际健康状态。

## 内部健康检查

内部健康检查监控后端服务的健康状态，可用于负载均衡和故障转移。

### 全局内部健康检查配置

```yaml
healthcheck:
  inner:
    enable: true
    interval: 30    # 检查间隔（秒）
    timeout: 5      # 检查超时（秒）
```

### 服务级健康检查

您可以为单个服务配置健康检查：

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        healthcheck:
          enable: true
          method: GET
          path: /health
          status: [200]
          ok: false
```

### 健康检查配置字段

| 字段 | 类型 | 描述 | 默认值 |
|------|------|------|--------|
| `enable` | bool | 为此服务启用健康检查 | `false` |
| `method` | string | 健康检查的 HTTP 方法 | `GET` |
| `path` | string | 健康检查端点路径 | `/health` |
| `status` | array | 有效 HTTP 状态代码列表 | `[200]` |
| `ok` | bool | 始终认为服务健康 | `false` |

### 健康检查方法

支持的 HTTP 方法：
- `GET`（默认）
- `POST`
- `HEAD`

### 健康检查状态代码

`status` 字段指定哪些 HTTP 状态代码被视为健康。例如：

```yaml
healthcheck:
  enable: true
  method: GET
  path: /health
  status: [200, 201]  # 200 和 201 都被视为健康
```

### 健康检查间隔和超时

全局内部健康检查配置控制检查服务的频率：

```yaml
healthcheck:
  inner:
    enable: true
    interval: 30    # 每 30 秒检查一次
    timeout: 5      # 5 秒后超时
```

- `interval`：检查服务的频率（秒）
- `timeout`：等待响应的最长时间（秒）

## 健康检查示例

### 基本服务健康检查

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        healthcheck:
          enable: true
          method: GET
          path: /health
          status: [200]
```

### 自定义健康检查路径

```yaml
rules:
  - host: api.example.com
    backend:
      service:
        name: api-service
        port: 8080
        healthcheck:
          enable: true
          method: GET
          path: /api/health
          status: [200, 204]
```

### 多个状态代码

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        healthcheck:
          enable: true
          method: GET
          path: /health
          status: [200, 201, 204]  # 接受多个成功代码
```

### POST 健康检查

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        healthcheck:
          enable: true
          method: POST
          path: /health/check
          status: [200]
```

### 始终健康（跳过实际检查）

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        healthcheck:
          enable: true
          ok: true  # 始终认为健康，跳过实际检查
```

## 健康检查行为

当服务健康检查失败时：

1. Ingress 继续路由流量（健康检查是信息性的）
2. 健康检查状态可用于监控和警报
3. 失败的健康检查会被记录以供调试

## 监控健康检查

您可以通过以下方式监控健康检查状态：

1. **日志**：健康检查失败会被记录
2. **指标**：健康检查指标（如果启用了指标）
3. **外部监控**：使用外部健康检查端点

## 最佳实践

1. **使用适当的间隔**：在及时检测和资源使用之间取得平衡
2. **设置合理的超时**：避免超时太短或太长
3. **使用标准路径**：使用常见的健康检查路径，如 `/health` 或 `/healthz`
4. **监控健康状态**：为失败的健康检查设置警报
5. **测试健康端点**：确保后端服务有工作的健康检查端点
6. **处理优雅降级**：设计服务以优雅地处理健康检查失败

## 故障排除

### 健康检查始终失败

- 验证健康检查路径是否存在于后端服务上
- 检查 HTTP 方法是否与后端期望的匹配
- 确保后端服务正在运行且可访问
- 验证 Ingress 和后端之间的网络连接

### 健康检查超时

- 如果后端响应缓慢，增加超时值
- 检查后端服务性能
- 验证网络延迟

### 健康检查未运行

- 确保设置了 `enable: true`
- 检查全局内部健康检查是否已启用
- 验证配置是否正确
