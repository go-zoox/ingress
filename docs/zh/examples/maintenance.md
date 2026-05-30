# 维护模式示例

源码位于 [`examples/maintenance/`](https://github.com/go-zoox/ingress/tree/master/examples/maintenance)。

## 全局常开维护（状态探测）

`maintenance.hosts` 中**未配置 `window`** 的条目在 Host 匹配时始终生效。可用 **`GET /_/ingress/status`** 探测该 Host 是否处于维护（返回 `{"status":"ok"}` 或 `{"status":"maintenance",...}`）。

<<< @/../examples/maintenance/global-always-on.yaml

```bash
curl -sS http://app.example.com/_/ingress/status
curl -sS -D - http://app.example.com/api   # 维护中时 503 + X-Ingress-Maintenance: true
```

## 全局维护 + 放行（bypass）

<<< @/../examples/maintenance/global-bypass.yaml

维护生效期间，**`/healthz`**（handler）及带 **`X-Maintenance-Bypass: secret-token`** 的请求仍可访问；其余路径返回 503。

## 路由 `scope: all`

<<< @/../examples/maintenance/route-scope-all.yaml

## 路由 `scope: listed` + 分 host 时间窗

<<< @/../examples/maintenance/route-scope-listed.yaml

## 全局 + 路由级组合

<<< @/../examples/maintenance/ingress.yaml

## 校验

```bash
ingress validate -c examples/maintenance/global-always-on.yaml
ingress validate -c examples/maintenance/global-bypass.yaml
ingress validate -c examples/maintenance/route-scope-all.yaml
ingress validate -c examples/maintenance/route-scope-listed.yaml
ingress validate -c examples/maintenance/ingress.yaml
```

详解见 [维护模式指南](../guide/maintenance.md)（响应头、访问日志字段等）。
