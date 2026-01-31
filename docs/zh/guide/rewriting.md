# 请求和响应重写

Ingress 为请求和响应提供灵活的重写功能，允许您在转发到后端服务之前修改头、路径、查询参数等。

## 请求重写

请求重写在发送到后端服务之前修改请求。

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

重写格式为 `pattern:replacement`：
- `pattern`：要匹配的正则表达式模式
- `replacement`：替换字符串（可以使用捕获组，如 `$1`、`$2`）

#### 路径重写示例

**简单路径重写：**
```yaml
request:
  path:
    rewrites:
      - ^/api:/v2/api
```

**带捕获组的路径重写：**
```yaml
request:
  path:
    rewrites:
      - ^/api/v1/(.*):/api/v2/$1
```

这将 `/api/v1/users` 重写为 `/api/v2/users`。

**多个路径重写：**
```yaml
request:
  path:
    rewrites:
      - ^/ip3/(.*):/$1
      - ^/ip2:/ip
```

重写按顺序应用。使用第一个匹配的重写。

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

当 `rewrite: true` 时：
- Host 头设置为 `{service-name}:{port}`
- 当后端服务期望特定主机名时很有用

当 `rewrite: false`（默认）时：
- 保留原始 Host 头
- 后端接收来自客户端的原始主机名

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
            X-User-ID: "12345"
```

头会被添加或覆盖。常见用例：
- 设置 `X-Forwarded-Proto` 用于 HTTPS 检测
- 添加认证头
- 传递用户信息
- 为后端服务设置自定义头

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

查询参数会被添加到请求中。如果参数已存在，可能会被覆盖。

### 请求延迟

在转发请求之前添加延迟：

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        request:
          delay: 100  # 延迟（毫秒）
```

用于：
- 速率限制模拟
- 测试超时行为
- 限制请求

### 请求超时

为后端请求设置超时：

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        request:
          timeout: 30  # 超时（秒）
```

如果后端在超时内没有响应，请求将失败。

## 响应重写

响应重写在发送到客户端之前修改响应。

### 响应头修改

修改响应头：

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        response:
          headers:
            X-Custom-Header: value
            Cache-Control: no-cache
```

常见用例：
- 添加安全头
- 修改缓存头
- 为客户端添加自定义头

## 完整重写示例

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
          path:
            rewrites:
              - ^/api/v1/(.*):/api/v2/$1
          headers:
            X-Forwarded-Proto: https
            X-Custom-Header: value
          query:
            version: v2
          delay: 0
          timeout: 30
        response:
          headers:
            X-Response-Header: value
```

## 使用正则表达式的路径重写

### 捕获组

在路径重写中使用捕获组：

```yaml
request:
  path:
    rewrites:
      - ^/api/v1/([^/]+)/(.*):/api/v2/$1/$2
```

这捕获两个组并重新排序它们。

### 复杂路径重写

```yaml
rules:
  - host: httpbin.example.work
    backend:
      service:
        name: httpbin.zcorky.com
        port: 443
        request:
          host:
            rewrite: true
          path:
            rewrites:
              - ^/ip3/(.*):/$1
              - ^/ip2:/ip
    paths:
      - path: /httpbin.org
        backend:
          service:
            name: httpbin.org
            port: 443
            request:
              path:
                rewrites:
                  - ^/httpbin.org/(.*):/$1
```

## 最佳实践

1. **测试重写模式**：验证正则表达式模式按预期匹配
2. **顺序很重要**：将更具体的重写放在通用重写之前
3. **使用捕获组**：利用正则表达式捕获组进行灵活重写
4. **保留重要头**：注意不要覆盖关键头
5. **记录重写**：记录复杂的重写规则以便维护
6. **监控影响**：监控重写如何影响后端服务

## 常见用例

### API 版本迁移

```yaml
request:
  path:
    rewrites:
      - ^/api/v1/(.*):/api/v2/$1
```

### 路径规范化

```yaml
request:
  path:
    rewrites:
      - ^/old-path/(.*):/new-path/$1
```

### 添加认证头

```yaml
request:
  headers:
    Authorization: Bearer token-here
```

### 设置协议信息

```yaml
request:
  headers:
    X-Forwarded-Proto: https
    X-Forwarded-For: $remote_addr
```

## 故障排除

### 重写不工作

- 验证正则表达式模式是否匹配路径
- 检查重写顺序（第一个匹配获胜）
- 确保重写语法正确（`pattern:replacement`）
- 单独测试正则表达式模式

### 头未设置

- 验证头名称是否正确
- 检查头值中的拼写错误
- 确保头在正确的部分（请求 vs 响应）

### 路径重写问题

- 使用正则表达式测试器测试正则表达式模式
- 验证捕获组引用（`$1`、`$2` 等）
- 检查特殊字符的转义问题
