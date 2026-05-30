# 维护模式

Ingress 可在计划内停机时对匹配的流量返回 **503 Service Unavailable**。维护判定在 **路由匹配与 WAF 之后**、**重定向 / Handler / 上游代理之前** 执行。

两层配置可叠加：

1. **全局 `maintenance:`** — 维护域名列表（每条可设时间窗）及默认 503 文案 / 放行规则。
2. **路由 `rules[].backend.service.maintenance`** — 仅 **Host 级 service 后端**；`scope: all`（规则下全部 Host）或 `scope: listed`（仅列表内 Host）。

任一层命中即可进入维护。两层同时命中时，**路由级 `title` / `subtitle` / `retry_after` 覆盖全局**；**bypass 规则合并**（全局 + 路由）。

## 全局维护

```yaml
maintenance:
  hosts:
    - host: app.example.com
    - host: staging-*.example.com
      window:
        start: "2026-05-30T02:00:00+08:00"
        end: "2026-05-30T06:00:00+08:00"
  retry_after: 3600
  title: 计划维护
  subtitle: 我们很快回来。
  bypass:
    allow_ips:
      - 10.0.0.0/8
    paths:
      - /healthz
      - /metrics/*
    header:
      name: X-Maintenance-Bypass
      value: secret-token
```

### `maintenance.hosts[]`

每条为 `{ host, window? }` 对象（`host` 必填）。**未配置 `window`**（或起止均为空）时，Host 匹配即 **始终处于维护**。

时间格式为 **RFC3339**（如 `2026-05-30T02:00:00+08:00`）。`end` 早于 `start` 会在校验阶段报错。

| 字段 | 说明 |
|------|------|
| `host` | Host 模式（精确、`*` 通配符或 Go 正则，推断规则与路由 `host` 相同） |
| `window.start` | 可选 RFC3339；省略表示无下限 |
| `window.end` | 可选 RFC3339；省略表示无上限 |

## 路由级维护

仅 **`rules[].backend.service`**（不支持 path 后端或 fallback）。需 **`backend.type: service`**（或可推断为 service）。

```yaml
rules:
  - host: "*.example.com"
    backend:
      type: service
      service:
        name: backend.internal
        port: 8080
        maintenance:
          enabled: true
          scope: listed          # all | listed（默认 all）
          hosts:
            - host: legacy.example.com
              window:
                start: "2026-05-31T00:00:00+08:00"
                end: "2026-05-31T01:00:00+08:00"
          title: 旧栈维护
          retry_after: 1800
          bypass:
            paths:
              - /healthz
```

| 字段 | 说明 | 默认 |
|------|------|------|
| `enabled` | 启用路由维护 | `false` |
| `scope` | `all` — 规则匹配的全部 Host；`listed` — 仅 `hosts[]` | `all` |
| `hosts` | `scope: listed` 时必填；格式同全局 `maintenance.hosts` | — |
| `retry_after` | 响应头 `Retry-After`（秒） | `0`（不发送） |
| `title` / `subtitle` | 覆盖内置 503 标题 / 说明 | 内置文案 |
| `bypass` | 同全局 bypass | — |

`scope: all` **不能** 配置 `hosts`；`scope: listed` **必须** 至少一条 host。

## 放行（bypass）

维护生效时，满足 bypass 的请求仍走正常后端：

| 类型 | 语义 |
|------|------|
| `allow_ips` | 客户端 IP 或 CIDR（解析 IP 时使用 `RemoteAddr`；必要时读 `X-Forwarded-For` 最左段） |
| `paths` | 精确路径，或后缀 `*` 前缀匹配（如 `/metrics/*` → 前缀 `/metrics/`） |
| `header` | 请求头名/值精确匹配 |

全局与路由的 bypass **取并集**。

### 维护响应头（`response_header`）

在维护 **503** 与 **`GET /_/ingress/status`**（Host 处于维护）时发送。

| 字段 | 说明 | 默认 |
|------|------|------|
| `name` | 响应头名称 | `X-Ingress-Maintenance` |
| `value` | 响应头值 | `true` |

省略整块配置则使用默认值。只填 `name` 或只填 `value` 时，另一项仍用默认。路由级 `response_header` 在路由维护命中时覆盖全局。

## 区分维护 503 与上游 503

两者 HTTP 状态码都可能是 **503**，但来源不同：

| 信号 | 维护 503 | 上游 503 |
|------|----------|----------|
| 响应头 | **`X-Ingress-Maintenance: true`**（默认；可通过 `response_header` 自定义） | _(无)_ |
| 访问日志 | **`maintenance_block=1`**，`upstream_response_length=-1` | `maintenance_block=0`，有真实上游长度/RTT |
| 响应体 | Ingress 错误页（可配 `title` / `subtitle`） | 上游原始 body |
| 是否连上游 | **否**（代理前短路） | **是** |

负载均衡 / 监控可用 **`GET {status_path}`**（默认 `/_/ingress/status`）探测 Host 级维护状态（不受 path bypass 影响）。

## 响应与日志

- HTTP **503**，HTML 错误页（`Accept` 偏好 JSON 时返回 JSON）。
- 维护 503 默认附带 **`X-Ingress-Maintenance: true`**（可通过 `maintenance.response_header` / `service.maintenance.response_header` 自定义）。
- `retry_after` > 0 时设置 **`Retry-After`**。
- 访问日志附加 **`maintenance_block=1`**。

## Ingress 状态探测

默认 **`GET /_/ingress/status`** — 在路由、WAF 与 bypass **之前**处理。可用 **`maintenance.status_path`** 自定义路径（须以 `/` 开头）。

| 条件 | HTTP | JSON `status` | 维护响应头 |
|------|------|---------------|------------|
| Host 未维护 | `200` | `"ok"` | _(无)_ |
| Host 维护中 | `503` | `"maintenance"`（含可选字段） | 已配置（默认 `X-Ingress-Maintenance: true`） |

JSON 响应含 `maintenance_header_name`、`maintenance_header_value`，与 `response_header` 一致。

示例（默认路径）：

```bash
curl -sS -D - http://app.example.com/_/ingress/status
```

自定义路径：

```yaml
maintenance:
  status_path: /internal/ingress-status
```

## Admin 控制台

**`admin.enabled: true`** 时，在 **维护** 菜单编辑全局配置；路由编辑器中配置规则级维护。

## 示例

可运行样例：[`examples/maintenance/`](https://github.com/go-zoox/ingress/tree/master/examples/maintenance)（见 [维护示例](../examples/maintenance.md)）。字段表见 [配置参考](./configuration.md#maintenance-维护)。

```bash
ingress validate -c examples/maintenance/global-always-on.yaml
```
