# Admin 控制台

可运行的示例包：内嵌 admin、示例 access/error 日志、TLS 证书，以及 SQLite 审计状态。

源码目录：[`examples/admin-console/`](https://github.com/go-zoox/ingress/tree/master/examples/admin-console)。

## 配置

<<< @/../examples/admin-console/ingress.yaml{yaml}

要点：

- **`admin.enabled: true`** — 与代理（**8080** / **8443**）同进程，API 监听 **9080**。
- **`admin.auth.type: basic`** — 本演示包显式启用登录（`admin` / `admin`）；默认 auth 类型为 **`none`** — 见 [Admin 认证示例](/zh/examples/admin-auth)。
- **`admin.web.dev_proxy: true`** — 仅 API；UI 需 `cd core/admin/web && pnpm dev`。
- 省略 **`logging`** 时，默认在 YAML 同目录写入 **`./access.log`**、**`./error.log`**（无需 `/var/log/ingress`）。

## 校验与运行

```bash
ingress validate -c examples/admin-console/ingress.yaml
ingress run -c examples/admin-console/ingress.yaml
```

预期启动日志：

```text
Admin started at http://127.0.0.1:9080
Server started at http://127.0.0.1:8080
```

## 示例数据

| 资源 | 用途 |
|------|------|
| `access.log` / `error.log` | 日志 UI 的预生成行 |
| `admin.db` | 首次启动创建；空库会写入 bootstrap WAF 事件与审计记录 |
| `certs/` | `https.ssl` 引用的示例证书 |

在仓库根目录重新生成：

```bash
python3 examples/admin-console/scripts/gen_sample_data.py
go run ./examples/admin-console/scripts/gen_sample_certs/main.go
```

详见 [Admin 控制台指南](/zh/guide/admin)。
