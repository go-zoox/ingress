# 配置参考

Ingress 使用 YAML 配置文件来定义路由规则、认证、SSL 证书和其他设置。

## 配置结构

```yaml
version: v1                    # 配置版本
port: 8080                     # HTTP 端口（默认：8080）

# 缓存配置
cache:
  ttl: 30                      # 缓存 TTL（秒）
  # engine: redis              # 可选：使用 Redis 缓存
  # host: 127.0.0.1
  # port: 6379
  # password: '123456'
  # db: 2

# HTTPS 配置
https:
  port: 8443                   # HTTPS 端口
  ssl:
    - domain: example.com
      cert:
        certificate: /path/to/cert.pem
        certificate_key: /path/to/key.pem

# 健康检查配置
healthcheck:
  outer:
    enable: true               # 启用外部健康检查
    path: /healthz             # 健康检查端点路径
    ok: true                   # 始终返回 OK
  inner:
    enable: true               # 启用内部服务健康检查
    interval: 30               # 检查间隔（秒）
    timeout: 5                 # 检查超时（秒）

# 回退服务（当没有规则匹配时使用）
fallback:
  service:
    name: httpbin.org
    port: 443

# 路由规则
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
```

## 配置字段

### 顶级字段

| 字段 | 类型 | 描述 | 默认值 |
|------|------|------|--------|
| `version` | string | 配置版本 | `v1` |
| `port` | int | 监听的 HTTP 端口 | `8080` |
| `cache` | object | 缓存配置 | - |
| `https` | object | HTTPS 配置 | - |
| `healthcheck` | object | 健康检查配置 | - |
| `fallback` | object | 回退后端 | - |
| `rules` | array | 路由规则 | `[]` |

### 缓存配置

| 字段 | 类型 | 描述 | 默认值 |
|------|------|------|--------|
| `ttl` | int | 缓存 TTL（秒） | `60` |
| `host` | string | Redis 主机（如果使用 Redis） | - |
| `port` | int | Redis 端口 | `6379` |
| `password` | string | Redis 密码 | - |
| `db` | int | Redis 数据库编号 | `0` |
| `prefix` | string | 缓存键前缀 | - |

### HTTPS 配置

| 字段 | 类型 | 描述 |
|------|------|------|
| `port` | int | 监听的 HTTPS 端口 |
| `ssl` | array | SSL 证书配置 |

#### SSL 证书

| 字段 | 类型 | 描述 |
|------|------|------|
| `domain` | string | 证书的域名 |
| `cert.certificate` | string | 证书文件路径 |
| `cert.certificate_key` | string | 私钥文件路径 |

### 健康检查配置

#### 外部健康检查

| 字段 | 类型 | 描述 | 默认值 |
|------|------|------|--------|
| `enable` | bool | 启用外部健康检查 | `false` |
| `path` | string | 健康检查端点路径 | `/healthz` |
| `ok` | bool | 始终返回 OK | `false` |

#### 内部健康检查

| 字段 | 类型 | 描述 | 默认值 |
|------|------|------|--------|
| `enable` | bool | 启用内部健康检查 | `false` |
| `interval` | int | 检查间隔（秒） | `30` |
| `timeout` | int | 检查超时（秒） | `5` |

### 回退配置

当没有路由规则匹配请求时，使用回退后端。

```yaml
fallback:
  service:
    name: fallback-service
    port: 8080
    protocol: http              # http 或 https
    request:
      host:
        rewrite: true           # 重写 Host 头
```

### 规则配置

规则定义如何将请求路由到后端服务。详细信息请参阅[路由指南](/zh/guide/routing)。

```yaml
rules:
  - host: example.com           # 要匹配的主机
    host_type: exact            # 匹配类型：exact、regex、wildcard
    backend:
      service:
        name: backend-service
        port: 8080
        protocol: http          # http 或 https
        auth:                   # 认证（可选）
          type: basic
          basic:
            users:
              - username: admin
                password: admin123
        healthcheck:            # 服务健康检查（可选）
          enable: true
          method: GET
          path: /health
          status: [200]
        request:
          host:
            rewrite: true       # 重写 Host 头
          path:
            rewrites:          # 路径重写规则
              - ^/api/v1:/api/v2
          headers:             # 附加头
            X-Custom-Header: value
          query:               # 查询参数
            key: value
          delay: 0              # 延迟（毫秒）
          timeout: 30           # 超时（秒）
      redirect:                 # 重定向配置（服务的替代方案）
        url: https://example.com
        permanent: false
    paths:                      # 基于路径的路由（可选）
      - path: /api
        backend:
          service:
            name: api-service
            port: 8080
```

## 环境变量

您可以使用环境变量覆盖某些配置：

- `CONFIG`: 配置文件路径
- `PORT`: HTTP 端口号

## 配置验证

Ingress 在启动时验证配置文件。如果有任何错误，服务器将不会启动并显示错误消息。

## 重新加载配置

您可以在不重启服务器的情况下重新加载配置：

1. 发送 SIGHUP 信号：`kill -HUP $(cat /tmp/gozoox.ingress.pid)`
2. 使用 reload 命令：`ingress reload`

服务器将重新加载配置文件并应用更改，而不会断开连接。
