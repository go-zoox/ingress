# 开发经验记录

本页记录最近一轮 ingress 改动中的实现经验，方便后续维护与迭代。

## 1) 全局 HTTP -> HTTPS 重定向

重定向配置放在 `https.redirect_from_http` 下：

```yaml
https:
  port: 443
  redirect_from_http:
    enabled: true
    permanent: true
    exclude_paths:
      - /healthz
```

关键约定：

- 重定向在路由匹配前执行。
- 仅当配置了 `https.port` 且设置 `enabled: true` 时，重定向才会生效。
- `enabled: true` 是显式启用。
- 默认保留原始 host/path/query。
- 已识别为 HTTPS 的请求（`TLS` 或 `X-Forwarded-Proto: https`）不重定向。
- `exclude_paths` 为精确路径匹配。

这样设计的原因：

- 不需要在每条 `rules` 上重复写跳转配置。
- 同时保留 `rules[].backend.redirect` 作为按路由定制跳转的能力。

## 2) 常量提取经验

重构时，尽量避免在逻辑分支中直接写协议/类型/请求头字符串字面量。

目前已提取到 `core/constants.go` 的典型常量：

- HostType：`hostTypeExact`、`hostTypeRegex`、`hostTypeWildcard`、`hostTypeAuto`
- Backend.Type：`backendTypeService`、`backendTypeHandler`、`backendTypeRedirect`
- Backend.mode（未设置 `request.host.rewrite` 时的 Host 默认）：`backendModeInternal`、`backendModeExternal`
- 全局 fallback 合成路由主机：`fallbackRuleHost`（`@@fallback`）
- 认证类型与挑战头：`authTypeBasic`、`authTypeBearer`、`authChallengeBasic`、`authChallengeBearer`
- Header 与 scheme：`headerXForwardedProto`、`headerWWWAuthenticate`、`schemeHTTP`、`schemeHTTPS`

收益：

- 减少分支判断中的拼写错误风险。
- 批量重构更安全。
- 代码评审时更容易区分“语义变更”和“文本替换”。

## 3) 校验与热重载安全

- `ingress validate` 现在按错误类别输出：
  - `yaml syntax error ...`
  - `invalid config format ...`
  - `unsupported configuration ...`
- 路由/backend 校验报错中会包含 **`rules[N] host="..." path="..."`**（规则级 backend 为 `path="/"`；子路径为配置的 path 模式）。**回退（fallback）** 使用 **`fallback path="/"`**。
- `ingress reload` 会先校验配置，只有通过后才发送 `SIGHUP`。

这样可以避免运行中加载损坏配置，保证启动和热重载的一致性。
