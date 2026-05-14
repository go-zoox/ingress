# 认证示例

不同认证方式的配置示例。

配置文件：[examples/authentication/](https://github.com/go-zoox/ingress/tree/master/examples/authentication)。

## HTTP Basic（单个用户）

<<< @/../examples/authentication/basic-single-user.yaml

## HTTP Basic（多个用户）

<<< @/../examples/authentication/basic-multi-user.yaml

## Bearer token（单个）

<<< @/../examples/authentication/bearer-single.yaml

## Bearer token（多个）

<<< @/../examples/authentication/bearer-multi.yaml

## 路径级认证

<<< @/../examples/authentication/path-level-mixed.yaml

## 测试

### Basic

```bash
curl -u admin:admin123 -H "Host: basic-auth.example.com" http://localhost:8080/
curl -u admin:admin123 -H "Host: basic-auth-multi.example.com" http://localhost:8080/
```

### Bearer

```bash
curl -H "Host: bearer-auth.example.com" -H "Authorization: Bearer my-secret-token-123" http://localhost:8080/
curl -H "Host: bearer-auth-multi.example.com" -H "Authorization: Bearer token1-abc123xyz" http://localhost:8080/
```

### 路径级

```bash
curl -u default:default123 -H "Host: mixed-auth.example.com" http://localhost:8080/
curl -u admin:admin123 -H "Host: mixed-auth.example.com" http://localhost:8080/admin
curl -H "Host: mixed-auth.example.com" -H "Authorization: Bearer api-token-1" http://localhost:8080/api
```
