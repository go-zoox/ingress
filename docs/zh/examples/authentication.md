# 认证示例

此页面提供不同认证配置的示例。

## Basic 认证

### 单用户

```yaml
version: v1
port: 8080

rules:
  - host: basic-auth.example.com
    backend:
      service:
        name: api-service
        port: 8080
        auth:
          type: basic
          basic:
            users:
              - username: admin
                password: admin123
```

### 多用户

```yaml
version: v1
port: 8080

rules:
  - host: basic-auth.example.com
    backend:
      service:
        name: api-service
        port: 8080
        auth:
          type: basic
          basic:
            users:
              - username: admin
                password: admin123
              - username: user1
                password: user123
              - username: user2
                password: user456
```

## Bearer Token 认证

### 单令牌

```yaml
version: v1
port: 8080

rules:
  - host: bearer-auth.example.com
    backend:
      service:
        name: api-service
        port: 8080
        auth:
          type: bearer
          bearer:
            tokens:
              - my-secret-token-123
```

### 多令牌

```yaml
version: v1
port: 8080

rules:
  - host: bearer-auth.example.com
    backend:
      service:
        name: api-service
        port: 8080
        auth:
          type: bearer
          bearer:
            tokens:
              - token1-abc123xyz
              - token2-def456uvw
              - token3-ghi789rst
```

## 路径级认证

不同路径使用不同的认证：

```yaml
version: v1
port: 8080

rules:
  - host: mixed-auth.example.com
    backend:
      service:
        name: api-service
        port: 8080
        auth:
          type: basic
          basic:
            users:
              - username: default
                password: default123
    paths:
      - path: /admin
        backend:
          service:
            name: admin-service
            port: 8080
            auth:
              type: basic
              basic:
                users:
                  - username: admin
                    password: admin123
                  - username: superadmin
                    password: super123
      - path: /api
        backend:
          service:
            name: api-service
            port: 8080
            auth:
              type: bearer
              bearer:
                tokens:
                  - api-token-1
                  - api-token-2
                  - api-token-3
```

## 测试

### Basic 认证

```bash
curl -u admin:admin123 http://basic-auth.example.com/api
```

### Bearer Token

```bash
curl -H "Authorization: Bearer my-secret-token-123" http://bearer-auth.example.com/api
```

### 路径级认证

```bash
# 使用默认 basic 认证
curl -u default:default123 http://mixed-auth.example.com/

# 使用管理员 basic 认证
curl -u admin:admin123 http://mixed-auth.example.com/admin

# 使用 bearer 令牌
curl -H "Authorization: Bearer api-token-1" http://mixed-auth.example.com/api
```
