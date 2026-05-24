# 管理控制台（Admin）

Ingress 可在与反向代理**同一进程**中嵌入**运维控制台**。在 `ingress.yaml` 顶层配置 **`admin:`** 即可启用，**没有**单独的 `ingress admin` 子命令。

控制台提供 HTTP API（默认端口 **9080**）和 React UI，用于查看路由、日志、TLS、缓存、WAF 事件，以及配置的校验 / 发布 / 热重载。

## 快速开始

最小配置：

```yaml
version: v1
port: 8080

admin:
  enabled: true
  port: 9080

rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
```

启动：

```bash
ingress run -c ingress.yaml
```

启动日志示例：

```text
Admin started at http://127.0.0.1:9080
Server started at http://127.0.0.1:8080
```

生产构建下（`admin.web.dev_proxy: false`）在浏览器打开 **http://127.0.0.1:9080** 即可使用内置 UI。

完整示例包：[`examples/admin-console/`](https://github.com/go-zoox/ingress/tree/master/examples/admin-console) — 多路由、示例日志、TLS 证书与 SQLite 状态。参见 [Admin 控制台示例](/zh/examples/admin-console)。

## 配置项

| 字段 | 类型 | 说明 | 默认值 |
|------|------|------|--------|
| `admin.enabled` | bool | 与 `ingress run` 一起启动 admin | `false` |
| `admin.port` | int | Admin 监听端口 | `9080` |
| `admin.database.driver` | string | 审计 / 修订记录数据库驱动 | `sqlite` |
| `admin.database.dsn` | string | SQLite DSN（相对路径相对 ingress 配置文件目录解析） | `file:admin.db?cache=shared&_fk=1` |
| `admin.web.dev_proxy` | bool | 仅 API；UI 由 Vite 开发服务器提供（代理 `/api`） | `false` |
| `admin.access_log_path` | string | 日志页读取的 access 日志路径（覆盖） | 来自 `logging` 文件 transport |
| `admin.error_log_path` | string | 日志页读取的 error 日志路径（覆盖） | 来自 `logging` 文件 transport |

本地 SQLite + UI 开发模式示例：

<<< @/../examples/admin-console/ingress.yaml{yaml}

启用控制台只需 **`admin:`** 段；文件中其余内容为路由演示。

## 日志与日志查看器

Admin **日志**页从磁盘文件读取。默认与 ingress 的 **`logging`** 在 prepare 之后使用的路径一致。

当 **`admin.enabled: true`** 且 **未配置 `logging`**（无 `enable`、`level`、`transports`）时，ingress 默认：

- `logging.enable: true`
- 在配置文件同目录写入 `access.log`、`error.log`

**显式配置的 `logging.*` 始终优先**，包括 `logging.enable: false` 或自定义 `transports`。当 **未启用 admin** 时，若只设置 `logging.enable: true` 且未写 `transports`，仍默认使用 `/var/log/ingress/access.log` 与 `error.log`。

仅覆盖 admin 读取路径（不改变 ingress 自身 logging）：

```yaml
admin:
  enabled: true
  access_log_path: /var/log/ingress/access.log
  error_log_path: /var/log/ingress/error.log
```

访问日志行格式见 [配置参考 · 访问日志字段](/zh/guide/configuration#访问日志字段)。查询支持 `cache_hit`、`waf_block`、`host`、`status` 以及按字节 `offset` 尾部读取。

## UI 开发

前端开发时开启 dev proxy 并单独跑 Vite：

```yaml
admin:
  enabled: true
  web:
    dev_proxy: true
```

```bash
ingress run -c ingress.yaml
cd core/admin/web && pnpm dev
```

Vite 开发服务器将 `/api` 代理到 admin 端口。生产 UI 在 `cd core/admin && make build` 后嵌入 `core/admin/static`。

## HTTP API

基础路径：**`/api/v1`**。响应为 JSON 封装格式。

| 方法 | 路径 | 用途 |
|------|------|------|
| `GET` | `/status` | 进程 / 配置摘要 |
| `GET` | `/routes` | 扁平化路由表 |
| `POST` | `/routes/match` |  dry-run 匹配（JSON：`host`、`path`） |
| `GET` | `/logs` | 检索 / 尾部读取 access 或 error 日志 |
| `GET` | `/metrics/overview` | 基于 access 日志窗口的聚合指标 |
| `GET` | `/waf/events` | 最近 WAF 审计记录（SQLite） |
| `GET` | `/tls/certs` | 配置中证书文件的元数据 |
| `POST` | `/tls/certs/check` | 检查单个域名 |
| `GET` | `/cache/overview` | 缓存引擎 / 键概览 |
| `GET` | `/config` | 读取 ingress YAML |
| `PUT` | `/config` | 保存 YAML（写入修订记录） |
| `POST` | `/config/validate` | 校验 YAML 或磁盘上的文件 |
| `POST` | `/config/preview` | 预览 / diff 待发布变更 |
| `POST` | `/config/publish` | 校验、保存并重载 |
| `POST` | `/config/modules` | 列出配置编辑器模块 |
| `POST` | `/config/modules/merge` | 合并单个模块补丁 |
| `GET` | `/config/revisions` | 修订历史列表 |
| `GET` | `/config/revisions/:id` | 单条修订 |
| `POST` | `/reload` | 校验磁盘配置并重载 ingress |
| `GET` | `/routes/:ri/:pi` | 路由详情（配置 + auth/cache/healthcheck） |
| `GET` | `/routes/:ri/:pi/metrics` | 路由级聚合指标 |
| `GET` | `/events/stream` | SSE 实时事件流（`?channels=...`） |
| `GET` | `/healthcheck` | 健康检查探测结果与汇总 |
| `GET` | `/settings` | Admin 与 ingress 设置快照 |

**在控制台内发布 / 重载**会先校验配置文件，再触发进程内热重载（与 **`ingress reload`** / **SIGHUP** 在 `ingress run` 启动时效果一致）。

## 实时事件（SSE）

管理控制台通过 **Server-Sent Events**（SSE）推送实时更新。连接端点：

```
GET /api/v1/events/stream?channels=metrics,waf,logs,health
```

支持的频道：`metrics`、`waf`、`logs`、`health`。UI 根据当前页面自动订阅。单 IP 最多 **5 个并发 SSE 连接**；超出或不可用时客户端自动降级为轮询。

事件格式：

```
event: channel:action
data: {"key": "value", ...}
```

## 路由详情

点击路由列表中的任意行，进入路由详情页 `/routes/:ruleIndex/:pathIndex`。展示内容：

- 完整路由配置（host、path、backend、auth、cache、healthcheck、WAF）
- 实时指标（QPS、延迟分位数、错误率、缓存命中率）
- 该路由的过滤日志、WAF 事件、缓存数据

## 拓扑图

**拓扑**页面（`/topology`）以纯 SVG 渲染三层图：**Host → Path → Backend**。节点颜色标识健康状态（绿色=正常、黄色=告警、红色=故障）。点击节点可跳转至路由详情或路由列表。

## 健康检查面板

**健康检查**页面（`/health`）展示所有配置了 `healthcheck` 的后端状态。后端每 **30 秒**探测一次，**5 秒**超时。状态变更通过 SSE `health` 频道推送。

## 配置草稿与撤销

配置编辑器追踪编辑历史（最多 **50 步**）。使用 **Ctrl+Z** / **Cmd+Z** 撤销，**Ctrl+Shift+Z** / **Cmd+Shift+Z** 重做。Tab 栏的草稿徽章显示未保存的变更。

## 一键回滚

在**配置版本历史**面板中，每个修订都有**回滚**按钮。点击后弹出确认对话框，然后校验、发布并重载所选版本 — 无需手动复制粘贴。

## 证书到期告警

**总览**页现在读取真实 TLS 证书数据，而非硬编码值。证书在 **30 天**内到期显示黄色告警，**7 天**内显示红色严重告警。

## 版本一致性标识

总览页对比**运行配置 hash** 与**最新修订 hash**。绿色徽章表示"配置一致"，黄色表示"有变更未发布"。

## 安全说明

- v1 **未内置** admin API 认证。请绑定 localhost，或置于可信网络 / VPN / 带认证的路由之后。
- 配置发布会写入 live `ingress.yaml` 并重载代理 — 务必限制 admin 端口的访问面。
- 不要在不可信网络暴露 **`admin.web.dev_proxy: true`**。

## 相关命令

运行或部署前校验：

```bash
ingress validate -c ingress.yaml
```

修改磁盘上的配置后重载：

```bash
ingress reload -c ingress.yaml
# 或：kill -HUP $(cat /tmp/gozoox.ingress.pid)
```

`run`、`validate`、`reload` 的选项见 [快速开始 · 命令行选项](/zh/guide/getting-started#命令行选项)。
