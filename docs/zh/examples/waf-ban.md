# 运行时封禁名单示例

源代码位于 [`examples/waf-ban/`](https://github.com/go-zoox/ingress/tree/master/examples/waf-ban)。

封禁是**运维态**而非配置：通过管理后台或 `/api/v1/waf/bans` 接口创建，保存在后台数据库，
**不会写入 `ingress.yaml`**。

## 配置（启用后台与受信代理）

<<< @/../examples/waf-ban/ingress.yaml

## 创建、试算与解封

```bash
# 封禁前试算影响面
curl -s -X POST http://127.0.0.1:9080/api/v1/waf/bans/preview \
  -H 'Content-Type: application/json' \
  -d '{"kind":"ip","value":"203.0.113.42"}'

# 封禁 IP 24 小时
curl -s -X POST http://127.0.0.1:9080/api/v1/waf/bans \
  -H 'Content-Type: application/json' \
  -d '{"kind":"ip","value":"203.0.113.42","ttl_seconds":86400,"reason":"端口扫描"}'

# 按 UA 封禁（Go 正则语法）
curl -s -X POST http://127.0.0.1:9080/api/v1/waf/bans \
  -H 'Content-Type: application/json' \
  -d '{"kind":"ua_regex","value":"(?i)(nikto|sqlmap)","ttl_seconds":604800}'

# 解封（立即恢复访问）
curl -s -X DELETE http://127.0.0.1:9080/api/v1/waf/bans/1
```

## 校验

```bash
ingress validate -c examples/waf-ban/ingress.yaml
```

语义、安全护栏与限制见 [WAF 指南](../guide/waf.md#运行时封禁名单)。
